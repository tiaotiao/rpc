package rpc

import (
	"encoding/gob"
	"fmt"
	"io"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	enc  *gob.Encoder
	dec  *gob.Decoder
}

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	c := new(GobCodec)
	c.conn = conn
	c.enc = gob.NewEncoder(conn)
	c.dec = gob.NewDecoder(conn)
	return c
}

func (c *GobCodec) WriteRequest(id int64, method string, params []interface{}) (err error) {
	d := rpcdata{Id: id, Method: method, Params: params}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *GobCodec) WriteResponse(id int64, result interface{}, e *Error) (err error) {
	d := rpcdata{Id: id, Result: result, Error: e}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *GobCodec) Read() (req *Request, resp *Response, err error) {
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

func (c *GobCodec) OnRegister(method string, in []interface{}, out interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("register type: %v", er.Error())
				return
			}
			if s, ok := e.(string); ok {
				err = fmt.Errorf("register type: %v", s)
				return
			}
			err = fmt.Errorf("register type: %v", e)
		}
	}()

	for _, v := range in {
		gob.Register(v)
	}
	gob.Register(out)

	return
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
