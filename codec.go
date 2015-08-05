package rpc

import (
	"encoding/json"
	"fmt"
	"io"
)

type Codec interface {
	WriteRequest(*Request) (err error)
	WriteResponse(*Response) (err error)

	Read() (req *Request, resp *Response, err error)
	ParseValue(val interface{}, dst interface{}) error

	Close() error
}

///////////////////////////////////////////////////////////////////////////////

type jsonCodec struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder
}

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	c := new(jsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	return c
}

func (c *jsonCodec) WriteRequest(req *Request) (err error) {
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

	return err
}

func (c *jsonCodec) WriteResponse(resp *Response) (err error) {
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

func (c *jsonCodec) Read() (req *Request, resp *Response, err error) {
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

		req = &Request{Id: r.Id, Method: r.Method, Params: r.RawParams}

	} else { // it's a response

		resp = &Response{Id: r.Id, Error: r.Error, Result: r.RawResult}
	}

	return
}

func (c *jsonCodec) ParseValue(raw interface{}, ptr interface{}) error {
	switch v := raw.(type) {
	case json.RawMessage:
		return json.Unmarshal(v, ptr)
	default:
		return fmt.Errorf("unknonw raw data type")
	}
}

func (c *jsonCodec) Close() error {
	return c.conn.Close()
}

var _ Codec = (*jsonCodec)(nil)
