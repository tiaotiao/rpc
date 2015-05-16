package rpc

import (
	"encoding/json"
	"io"
)

//////////////////////////////////////////////////////////////////

type JsonCodec struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder
}

func NewJsonCodec(conn io.ReadWriteCloser) *JsonCodec {
	c := new(JsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	return c
}

func (c *JsonCodec) WriteRequest(id int64, method string, params []interface{}) (err error) {
	d := rpcdata{Id: id, Method: method, Params: params}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *JsonCodec) WriteResponse(id int64, result interface{}, e *Error) (err error) {
	d := rpcdata{Id: id, Result: result, Error: e}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *JsonCodec) Read() (req *Request, resp *Response, err error) {
	var r rpcdata
	err = c.dec.Decode(&r) // read and decode
	if err != nil {
		return
	}

	if r.Method != "" {
		req = &Request{Id: r.Id, Method: r.Method, Params: r.Params}
	} else {
		resp = &Response{Id: r.Id, Result: r.Result, Error: r.Error}
	}
	return
}

func (c *JsonCodec) OnRegister(method string, f interface{}) error {
	// TODO
	return nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}
