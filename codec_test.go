package rpc

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type fooType struct {
	Name  string
	Point float64
}

func TestGobCodec(t *testing.T) {
	var buf = new(buffer)
	c := NewGobCodec(buf)
	s := NewGobCodec(buf)

	v := fooType{}

	c.RegisterType(v)
	s.RegisterType(v)

	testCodec(c, s, t)
}

func TestJsonCodec(t *testing.T) {
	// TODO bug fix: Decode json data to an interface{} results a map[string]interface{} but struct.
	// To solve this, convert the map[string]interface{} to a struct, or unmarshal data to json.RawMessage.
	var buf = new(buffer)
	c := NewJsonCodec(buf)
	s := NewJsonCodec(buf)

	v := fooType{}

	c.RegisterType(v)
	s.RegisterType(v)

	testCodec(c, s, t)
}

func testCodec(c, s Codec, t *testing.T) {
	var id int64 = 123
	var method string = "foo"
	var params = []interface{}{"tom", 10, 20}

	writeAndCheckRequest(c, s, id, method, params, t)

	writeAndCheckRequest(c, s, id, method, []interface{}{[]int{10, 20, 30}, fooType{"tom", 3.14}}, t)

	writeAndCheckResponse(c, s, id, nil, nil, t)

	//writeAndCheckResponse(c, s, id, 30, nil, t)

	//writeAndCheckResponse(c, s, id, nil, ErrInvalidParams, t)
}

func writeAndCheckRequest(c, s Codec, id int64, method string, params []interface{}, t *testing.T) {
	// client write request
	err := c.WriteRequest(id, method, params)
	if err != nil {
		t.Fatal(err.Error(), method, params)
	}

	// server read
	req, resp, err := s.Read()
	if err != nil {
		t.Fatal(err.Error(), req, resp)
	}
	if resp != nil {
		t.Fatal("should not be resp:", resp)
	}

	// check request
	if req.Id != id {
		t.Fatal("id not match", req.Id, req, id)
	}
	if req.Method != method {
		t.Fatal("method not match", req.Method, req, method)
	}
	if req.Len() != len(params) {
		t.Fatal("params not match", req.params, req, params)
	}

	for i := 0; i < req.Len(); i++ {
		v := params[i]
		p := reflect.New(reflect.TypeOf(v))
		pv := p.Interface()

		err = req.Param(i, pv)
		if err != nil {
			t.Fatal(err.Error(), i, req.params, params)
		}

		err = equal(p.Elem().Interface(), params[i])
		if err != nil {
			t.Fatal(err.Error(), i, req.params, params)
		}
	}
}

func writeAndCheckResponse(c, s Codec, id int64, result interface{}, e *Error, t *testing.T) {
	// server write response
	err := s.WriteResponse(id, result, e)
	if err != nil {
		t.Fatal(err.Error())
	}

	// read response
	req, resp, err := c.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	if req != nil {
		t.Fatal("should not be req:", req)
	}

	// check response
	if resp.Id != id {
		t.Fatal("id not match")
	}

	rt := reflect.TypeOf(result)

	if rt != nil {
		rp := reflect.New(reflect.TypeOf(result))
		res := rp.Interface()
		err = resp.Result(res)
		if err != nil {
			t.Fatal(err.Error(), resp, id, result, e)
		}

		if err = equal(result, rp.Elem().Interface()); err != nil {
			t.Fatal(err.Error(), resp, id, result, e)
		}
	} else {
		var res interface{}
		err = resp.Result(&res)
		if err != nil {
			t.Fatal(err.Error(), resp.result)
		}
		if res != nil {
			t.Fatalf("result is nil %v, %v %v, %v", id, resp.result, result, e)
		}
	}

	if resp.Error != nil && resp.Error.Error() != e.Error() {
		t.Fatal("Error: resp.Error", resp)
	}
}

func equal(v, p interface{}) error {
	vv := reflect.ValueOf(v)
	pv := reflect.ValueOf(p)
	if vv == pv {
		return nil
	}
	// if !vv.Type().ConvertibleTo(pv.Type()) {
	// 	return fmt.Errorf("param type not match %v, %v", vv.Type(), pv.Type())
	// }
	if !reflect.DeepEqual(vv.Convert(pv.Type()).Interface(), p) {
		return fmt.Errorf("params not equal")
	}
	return nil
}

type buffer struct {
	bytes.Buffer
}

func (b *buffer) Close() error {
	return nil
}
