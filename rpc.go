package rpc

import (
	"errors"
	// "fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Rpc struct {
	codec Codec

	methods *Methods

	// server
	handler Handler

	// client
	reqMap  map[int64]chan *Response
	reqLock sync.RWMutex

	reqId   int64
	timeout time.Duration
}

type Handler func(method string, input interface{}) (output interface{}, err error)

func Dial(network, address string) (*Rpc, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewRpc(conn), nil
}

func Accept(l net.Listener) (*Rpc, error) {
	conn, err := l.Accept()
	if err != nil {
		return nil, err
	}
	return NewRpc(conn), nil
}

func NewRpc(conn io.ReadWriteCloser) *Rpc {
	codec := NewJsonCodec(conn)
	return NewRpcCodec(codec)
}

func NewRpcCodec(codec Codec) *Rpc {
	r := new(Rpc)
	r.codec = codec
	r.methods = newMethods()
	r.reqMap = make(map[int64]chan *Response)
	r.timeout = time.Second * 10

	jsonCodec, ok := codec.(*JsonCodec)
	if ok {
		jsonCodec.methods = r.methods
	}
	return r
}

func (r *Rpc) WriteRequest(req *Request) error {
	return r.codec.WriteRequest(req)
}

func (r *Rpc) WriteResponse(resp *Response) error {
	return r.codec.WriteResponse(resp)
}

func (r *Rpc) Read() (*Request, *Response, error) {
	return r.codec.Read()
}

func (r *Rpc) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

func (r *Rpc) SetHandler(handler Handler) {
	r.handler = handler
}

func (r *Rpc) CallRemote(method string, input interface{}) (interface{}, error) {
	id := atomic.AddInt64(&r.reqId, 1)
	ch := make(chan *Response, 1)

	// record request id
	r.reqLock.Lock()
	r.reqMap[id] = ch
	r.reqLock.Unlock()

	defer func() {
		r.reqLock.Lock()
		delete(r.reqMap, id)
		r.reqLock.Unlock()
	}()

	// send data
	err := r.codec.WriteRequest(&Request{id, method, input})
	if err != nil {
		return nil, err
	}

	var resp *Response

	// wait for response
	if r.timeout > 0 {
		select {
		case resp = <-ch:
		case <-time.After(r.timeout):
			return nil, ErrTimeout
		}
	} else {
		resp = <-ch
	}

	// return
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}

func (r *Rpc) onResponse(resp *Response, err error) error {
	r.reqLock.Lock()
	ch, ok := r.reqMap[resp.Id]
	delete(r.reqMap, resp.Id) // delete here to avoid duplicated response
	r.reqLock.Unlock()

	if !ok {
		// TODO report an error or not?
		println("Error: rpc client id not found", resp.Id, resp.Result, resp.Error)
		return nil
	}

	if err != nil {
		// something wrong with decoding this response
		resp.Error = NewError(CodeParseError, err.Error())
	}

	ch <- resp

	return nil
}

func (r *Rpc) onRequest(req *Request, err error) error {
	var result interface{}
	var e *Error
	var ok bool

	if err == nil {
		if r.handler != nil {
			result, err = r.handler(req.Method, req.Params) // handle request
		} else {
			err = NewError(CodeInternalError, "handler is nil")
		}

	}

	if err != nil {
		e, ok = err.(*Error)
		if !ok {
			e = NewError(CodeFunctionError, err.Error())
		}
	}

	return r.codec.WriteResponse(&Response{req.Id, result, e})
}

func (r *Rpc) Run() error {
	for {
		req, resp, err := r.Read()

		if req != nil {
			err = r.onRequest(req, err)

		} else if resp != nil {
			err = r.onResponse(resp, err)

		} else {
			println("rpc read error: ", err.Error())
			break
		}

		// TODO error handler
		if err != nil {
			println("rpc run error: ", err.Error())
			break
		}
	}
	return nil
}

func (r *Rpc) Close() error {
	return r.codec.Close()
}

///////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////

type Request struct {
	Id     int64
	Method string
	Params interface{}
}

type Response struct {
	Id     int64
	Result interface{}
	Error  *Error
}

type Error struct {
	Code    int    `"json:"code"`
	Message string `"json:"message"`
}

func NewError(code int, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

func (e *Error) Error() string {
	return e.Message
}

///////////////////////////////////////////////////////////

var (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	CodeFunctionError  = -32604
)

var (
	ErrDisconnected = errors.New("disconnected")
	ErrTimeout      = errors.New("timeout")
)

var (
	ErrParseError     = NewError(CodeParseError, "parse error")
	ErrInvalidRequest = NewError(CodeInvalidRequest, "invalid request")
	ErrMethodNotFound = NewError(CodeMethodNotFound, "method not found")
	ErrInvalidParams  = NewError(CodeInvalidParams, "invalid params")
	ErrInternalError  = NewError(CodeInternalError, "internal error")
)
