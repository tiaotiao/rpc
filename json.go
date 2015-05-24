package rpc

import (
	"encoding/json"
	"fmt"
	"io"
)

// //////////////////////////////////////////////////////////////////

type JsonCodec struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder
}

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	c := new(JsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
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

		req = &Request{Id: r.Id, Method: r.Method, Params: r.RawParams}

	} else { // it's a response

		resp = &Response{Id: r.Id, Error: r.Error, Result: r.RawResult}
	}

	return
}

func (c *JsonCodec) ParseValue(raw interface{}, ptr interface{}) error {
	switch v := raw.(type) {
	case json.RawMessage:
		return json.Unmarshal(v, ptr)
	default:
		return fmt.Errorf("unknonw raw data type")
	}
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

var _ Codec = (*JsonCodec)(nil)
