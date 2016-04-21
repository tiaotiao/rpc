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

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	c := new(JsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	return c
}

func (c *JsonCodec) WriteRequest(id int64, method string, params []interface{}) (err error) {
	d := jsondata{Id: id, Method: method}

	var raw json.RawMessage
	for _, param := range params {
		raw, err = json.Marshal(param)
		if err != nil {
			return err
		}
		d.Params = append(d.Params, raw)
	}

	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *JsonCodec) WriteResponse(id int64, result interface{}, e *Error) (err error) {
	d := jsondata{Id: id, Error: e}
	d.Result, err = json.Marshal(result)
	if err != nil {
		return err
	}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *JsonCodec) Read() (req *Request, resp *Response, err error) {
	var r jsondata
	err = c.dec.Decode(&r) // read and decode
	if err != nil {
		return
	}

	if r.Method != "" {
		req = &Request{Id: r.Id, Method: r.Method, codec: c}
		for _, p := range r.Params {
			req.params = append(req.params, p)
		}
	} else {
		resp = &Response{Id: r.Id, result: r.Result, Error: r.Error, codec: c}
	}
	return
}

func (c *JsonCodec) Unmarshal(data interface{}, pv interface{}) error {
	d := data.(json.RawMessage)
	return json.Unmarshal(d, pv)
}

func (c *JsonCodec) RegisterType(v interface{}) error {
	// DO NOTHING
	return nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

////////////////////////////////////////////////////////////////////////////////

// Combine Request and Response for decode
type jsondata struct {
	Id     int64             `json:"id"`
	Method string            `json:"method,omitempty"`
	Params []json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage   `json:"result,omitempty"`
	Error  *Error            `json:"error,omitempty"`
}
