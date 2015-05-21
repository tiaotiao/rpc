package rpc

import (
	"errors"
	"io"
	"net"
)

type Rpc struct {
	codec  Codec
	Client *Client
	Server *Server
}

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
	codec := newDefCodec(conn)
	return NewRpcWithCodec(codec)
}

func NewRpcWithCodec(codec Codec) *Rpc {
	r := new(Rpc)
	r.codec = codec
	r.Client = newClientWithCodec(codec)
	r.Server = newServerWithCodec(codec)
	go r.run()
	return r
}

func (r *Rpc) Close() error {
	return r.codec.Close()
}

func (r *Rpc) run() error {
	for {
		req, resp, err := r.codec.Read()

		if req != nil {
			err = r.Server.onRequest(req, err)

		} else if resp != nil {
			err = r.Client.onResponse(resp)

		} else {
			println("rpc read error: ", err.Error())
			break
		}

		if err != nil {
			println("rpc run error: ", err.Error())
			break
		}
	}
	return nil
}

///////////////////////////////////////////////////////////

type Request struct {
	Id     int64
	Method string
	Params []interface{}
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
