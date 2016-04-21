package rpc

import (
	"fmt"
	"reflect"
)

type Codec interface {
	WriteRequest(id int64, method string, params []interface{}) (err error)
	WriteResponse(id int64, result interface{}, e *Error) (err error)

	Read() (req *Request, resp *Response, err error)

	Unmarshal(data interface{}, pv interface{}) error
	RegisterType(v interface{}) error

	Close() error
}

///////////////////////////////////////////////////////////

type Request struct {
	Id     int64
	Method string

	codec  Codec
	params []interface{}
}

type Response struct {
	Id    int64
	Error *Error

	codec  Codec
	result interface{}
}

func (req *Request) Len() int {
	return len(req.params)
}

func (req *Request) Param(i int, pv interface{}) error {
	if i >= len(req.params) {
		return fmt.Errorf("index out of range")
	}

	return req.codec.Unmarshal(req.params[i], pv)
}

func (resp *Response) Result(pv interface{}) error {
	return resp.codec.Unmarshal(resp.result, pv)
}

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////

func codecRegisterFuncTypes(c Codec, f interface{}) (err error) {
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

	t := reflect.TypeOf(f)

	for i := 0; i < t.NumIn(); i++ {
		it := t.In(i)
		v := codecMakeValue(it)
		if v == nil {
			continue
		}
		err := c.RegisterType(v.Interface())
		if err != nil {
			return err
		}
	}

	for i := 0; i < t.NumOut(); i++ {
		ot := t.Out(i)
		v := codecMakeValue(ot)
		if v == nil {
			continue
		}
		err := c.RegisterType(v.Interface())
		if err != nil {
			return err
		}
	}

	return
}

func codecMakeValue(t reflect.Type) *reflect.Value {
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
