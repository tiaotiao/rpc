package rpc

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestGobCodec(t *testing.T) {
	var buf = new(buffer)
	c := NewGobCodec(buf)
	s := NewGobCodec(buf)
	testCodec(c, s, t)
}

func TestJsonCodec(t *testing.T) {
	var buf = new(buffer)
	c := NewJsonCodec(buf)
	s := NewJsonCodec(buf)
	testCodec(c, s, t)
}

func testCodec(c, s Codec, t *testing.T) {
	// basic
	requestAndResponse(c, s, []interface{}{1}, 2, nil, t)
	requestAndResponse(c, s, []interface{}{123, "hello", true, 3.14}, "ok", nil, t)

	// error
	requestAndResponse(c, s, []interface{}{"my bad"}, nil, NewError(400, "bad request"), t)

	// empty
	requestAndResponse(c, s, nil, nil, nil, t)
	requestAndResponse(c, s, []interface{}{}, nil, nil, t)

	// array
	requestAndResponse(c, s, []interface{}{[]int64{10, 20, 30}}, 60, nil, t)
	requestAndResponse(c, s, []interface{}{"hello,world"}, []string{"hello", "world"}, nil, t)

	// struct input
	requestAndResponse(c, s, []interface{}{struct {
		Name  string
		Float float64
	}{"tom", 3.14}}, nil, nil, t)

	// struct output
	requestAndResponse(c, s, []interface{}{"book", 2231}, struct {
		Book  string
		Price float64
	}{"harry potter", 49.5}, nil, t)

	// array of struct
	requestAndResponse(c, s, []interface{}{"all books"}, []struct {
		Book  string
		Price float64
	}{{"harry potter", 49.5}, {"c++ programming", 60}, {"don't make me think", 22.5}}, nil, t)
}

var gid int64 = 1000

func requestAndResponse(c, s Codec, params []interface{}, result interface{}, e *Error, t *testing.T) {
	gid += 1
	id := gid
	method := fmt.Sprintf("method_%v", id)

	// register type
	c.OnRegister(method, params, result)
	s.OnRegister(method, params, result)

	// request
	writeAndCheckRequest(c, s, id, method, params, t)

	// response
	writeAndCheckResponse(c, s, id, method, result, e, t)
}

func writeAndCheckRequest(c, s Codec, id int64, method string, params []interface{}, t *testing.T) {
	// client write request
	err := c.WriteRequest(id, method, params)
	if err != nil {
		t.Fatal(err.Error())
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
	if len(req.Params) != len(params) {
		t.Fatal("params not match", req.Params, req, params)
	}
	for i, v := range req.Params {
		err = equal(v, params[i])
		if err != nil {
			t.Fatal(err.Error(), i, req.Params, params)
		}
	}

	// fmt.Printf("check request OK. %v: %v -> %v\n", method, params, req.Params)
}

func writeAndCheckResponse(c, s Codec, id int64, method string, result interface{}, e *Error, t *testing.T) {
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
		t.Fatal("should not be req:", method, req)
	}

	// check response
	if resp.Id != id {
		t.Fatal("id not match", method)
	}

	if err = equal(result, resp.Result); err != nil {
		t.Fatal(fmt.Sprintf("%v [%v %v]: %#v != %#v, %v", err.Error(), method, id, resp, result, e))
	}
	if resp.Error != nil && resp.Error.Error() != e.Error() {
		t.Fatal("Error: resp.Error", resp)
	}

	// fmt.Printf("check response OK. %v: %v(%v) -> %v(%v)\n", method, result, e, resp.Result, resp.Error)
}

func equal(v, p interface{}) error {
	vv := reflect.ValueOf(v)
	pv := reflect.ValueOf(p)
	if vv == pv {
		return nil
	}
	if !vv.Type().ConvertibleTo(pv.Type()) {
		return fmt.Errorf("param type not match")
	}
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
