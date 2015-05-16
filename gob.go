package rpc

import (
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
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

func (c *GobCodec) OnRegister(method string, f interface{}) error {
	return c.registerFuncTypes(f)
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}

//////////////////////////////////////////////////////////////////

func (c *GobCodec) registerFuncTypes(f interface{}) (err error) {

	t := reflect.TypeOf(f)

	for i := 0; i < t.NumIn(); i++ {
		it := t.In(i)
		v := c.makeValue(it)
		if v == nil {
			continue
		}
		err := c.registerType(v.Interface())
		if err != nil {
			return err
		}
	}

	for i := 0; i < t.NumOut(); i++ {
		ot := t.Out(i)
		v := c.makeValue(ot)
		if v == nil {
			continue
		}
		err := c.registerType(v.Interface())
		if err != nil {
			return err
		}
	}

	return
}

func (c *GobCodec) registerType(v interface{}) (err error) {
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

	gob.Register(v)

	return nil
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
