package rpc

import (
	"io"
)

type Codec interface {
	WriteRequest(id int64, method string, params []interface{}) (err error)
	WriteResponse(id int64, result interface{}, e *Error) (err error)

	Read() (req *Request, resp *Response, err error)

	OnRegister(method string, f interface{}) error

	Close() error
}

var newDefCodec func(io.ReadWriteCloser) Codec = NewGobCodec

func SetDefaultCodec(newCodec func(io.ReadWriteCloser) Codec) {
	newDefCodec = newCodec
}

////////////////////////////////////////////////////////////////////////////////

// Combine Request and Response for decode
type rpcdata struct {
	Id     int64         `json:"id"`
	Method string        `json:"method,omitempty"`
	Params []interface{} `json:"params,omitempty"`
	Result interface{}   `json:"result,omitempty"`
	Error  *Error        `json:"error,omitempty"`
}
