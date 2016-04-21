package rpc

import (
	"encoding/gob"
	"io"

	"github.com/tiaotiao/go/util"
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
	d := gobdata{Id: id, Method: method, Params: params}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *GobCodec) WriteResponse(id int64, result interface{}, e *Error) (err error) {
	d := gobdata{Id: id, Result: result, Error: e}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *GobCodec) Read() (req *Request, resp *Response, err error) {
	var r gobdata
	err = c.dec.Decode(&r) // read and decode
	if err != nil {
		return
	}

	if r.Method != "" {
		req = &Request{Id: r.Id, Method: r.Method, params: r.Params, codec: c}
	} else {
		resp = &Response{Id: r.Id, result: r.Result, Error: r.Error, codec: c}
	}
	return
}

func (c *GobCodec) Unmarshal(data interface{}, pv interface{}) error {
	return util.Assign(pv, data)
}

func (c *GobCodec) RegisterType(v interface{}) error {
	gob.Register(v)
	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}

////////////////////////////////////////////////////////////////////////////////

// Combine Request and Response for decode
type gobdata struct {
	Id     int64
	Method string
	Params []interface{}
	Result interface{}
	Error  *Error
}
