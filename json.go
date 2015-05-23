package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// //////////////////////////////////////////////////////////////////

type JsonCodec struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder

	methods *Methods
	reqIds  map[int64]string // map[reqId]method
	reqLock sync.RWMutex
}

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	c := new(JsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)

	c.reqIds = make(map[int64]string)
	return c
}

func (c *JsonCodec) WriteRequest(req *Request) (err error) {
	d := struct {
		Id     int64       `json:"id"`
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}{
		req.Id, req.Method, req.Params,
	}

	err = c.enc.Encode(&d) // encode and write
	if err != nil {
		return err
	}

	// record the request id
	c.reqLock.Lock()
	c.reqIds[req.Id] = req.Method
	c.reqLock.Unlock()

	return err
}

func (c *JsonCodec) WriteResponse(resp *Response) (err error) {
	d := struct {
		Id     int64       `json:"id"`
		Result interface{} `json:"result"`
		Error  *Error      `json:"error,omitempty"`
	}{
		resp.Id, resp.Result, resp.Error,
	}

	err = c.enc.Encode(&d)

	return err
}

func (c *JsonCodec) Read() (req *Request, resp *Response, err error) {
	var r = struct {
		Id        int64           `json:"id"`
		Method    string          `json:"method"`
		RawParams json.RawMessage `json:"params"`
		RawResult json.RawMessage `json:"result"`
		Error     *Error          `json:"error"`
	}{}

	err = c.dec.Decode(&r)
	if err != nil {
		return
	}

	if r.Method != "" { // it's a request

		req = &Request{Id: r.Id, Method: r.Method}

		// looking for method
		in, ok := c.methods.In(r.Method)
		if !ok {
			// method is not found, bad request
			err = ErrMethodNotFound
			return
		}

		// unmarshal params
		in, err = c.unmarshal(r.RawParams, in)
		if err != nil {
			err = ErrInvalidParams
			return
		}

		req.Params = in

	} else { // it's a response

		resp = &Response{Id: r.Id, Error: r.Error}

		// looking for method
		c.reqLock.Lock()
		method, ok := c.reqIds[r.Id]
		delete(c.reqIds, r.Id)
		c.reqLock.Unlock()
		if !ok {
			// id is not found, bad response
			err = fmt.Errorf("id is not found, bad response. %v", r)
			return
		}

		out, ok := c.methods.Out(method)
		if !ok {
			err = ErrMethodNotFound
			return
		}

		// unmarshal result
		out, err = c.unmarshal(r.RawResult, out)
		if err != nil {
			err = ErrInvalidParams
			return
		}

		resp.Result = out
	}

	return
}

func (c *JsonCodec) unmarshal(raw json.RawMessage, val interface{}) (res interface{}, err error) {
	if val == nil {
		return nil, nil
	}

	t := reflect.TypeOf(val)
	pv := reflect.New(t) // pointer of t

	err = json.Unmarshal(raw, pv.Interface())
	if err != nil {
		return nil, err
	}
	return pv.Elem().Interface(), nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

var _ Codec = (*JsonCodec)(nil)
