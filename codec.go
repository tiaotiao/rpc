package rpc

import (
	"fmt"
	"io"
	"reflect"
)

type Codec interface {
	WriteRequest(id int64, method string, params []interface{}) (err error)
	WriteResponse(id int64, result interface{}, e *Error) (err error)

	Read() (req *Request, resp *Response, err error)

	OnRegister(method string, in []interface{}, out interface{}) error

	Close() error
}

var newDefCodec func(io.ReadWriteCloser) Codec = NewGobCodec

func SetDefaultCodec(newCodec func(io.ReadWriteCloser) Codec) {
	newDefCodec = newCodec
}

func OnRegisterFunc(c Codec, method string, f interface{}) error {
	t := reflect.TypeOf(f)

	var in []interface{}
	for i := 0; i < t.NumIn(); i++ {
		it := t.In(i)
		v := reflect.New(it).Elem()
		in = append(in, v.Interface())
	}

	var out interface{}
	numOut := t.NumOut() - 1 // ignore the last output param which is known as an error type
	if numOut > 1 {
		return fmt.Errorf("too many output params")
	}
	if numOut == 1 {
		it := t.Out(0)
		v := reflect.New(it).Elem()
		out = v.Interface()
	}

	return c.OnRegister(method, in, out)
}

func (c *GobCodec) makeValue(t reflect.Type) *reflect.Value {
	var v reflect.Value
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.UnsafePointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface:
		return nil
	case reflect.Map:
		v = reflect.MakeMap(t)
	case reflect.Slice, reflect.Array:
		v = reflect.MakeSlice(t, 0, 0)
	case reflect.Struct, reflect.String:
		v = reflect.Zero(t)
	default:
		v = reflect.Zero(t)
	}
	return &v
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
