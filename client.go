package rpc

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	codec   Codec
	reqid   int64
	reqMap  map[int64]chan *Response
	lock    sync.RWMutex
	timeout time.Duration
}

func newClientWithCodec(codec Codec) *Client {
	c := new(Client)
	c.codec = codec
	c.reqid = 0
	c.reqMap = make(map[int64]chan *Response)
	c.timeout = time.Second * 5
	return c
}

func (c *Client) MakeClient(client interface{}) error {
	t := reflect.TypeOf(client)
	v := reflect.ValueOf(client)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("Arg 'client' must be a point.")
	}
	if v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Arg 'client' must be a point of struct! eg. &myrpc{}")
	}

	var count int
	for i := 0; i < t.Elem().NumField(); i++ {
		tf := t.Elem().Field(i)
		vf := v.Elem().Field(i)
		if vf.Kind() != reflect.Func {
			continue
		}
		if !vf.CanAddr() || !vf.Addr().CanInterface() {
			continue
		}
		err := c.MakeFunc(tf.Name, vf.Addr().Interface())
		if err != nil {
			return err
		}
		count += 1
	}

	if count <= 0 {
		return fmt.Errorf("Make rpc failed, no func field been found")
	}

	return nil
}

func (c *Client) MakeFunc(method string, fptr interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = fmt.Errorf("make rpc: %v", er.Error())
				return
			}
			if s, ok := e.(string); ok {
				err = fmt.Errorf("make rpc: %v", s)
				return
			}
			err = fmt.Errorf("make rpc: %v", e)
		}
	}()

	fn := reflect.ValueOf(fptr).Elem()

	// f must return error as last param
	nOut := fn.Type().NumOut()
	if nOut == 0 || fn.Type().Out(nOut-1).Kind() != reflect.Interface {
		err = fmt.Errorf("%s return final output param must be error interface", method)
		return
	}

	_, b := fn.Type().Out(nOut - 1).MethodByName("Error")
	if !b {
		err = fmt.Errorf("%s return final output param must be error interface", method)
		return
	}

	// make func
	f := func(in []reflect.Value) []reflect.Value {
		out := c.call(fn, method, in)
		return out
	}

	v := reflect.MakeFunc(fn.Type(), f)
	fn.Set(v)

	// register type
	err = c.codec.OnRegister(method, v.Interface())
	if err != nil {
		return err
	}

	return
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *Client) CallRemote(method string, params []interface{}) (interface{}, error) {
	codec := c.codec
	if codec == nil {
		return nil, ErrDisconnected
	}

	id := atomic.AddInt64(&c.reqid, 1)
	ch := make(chan *Response)

	c.lock.Lock()
	c.reqMap[id] = ch
	c.lock.Unlock()

	err := codec.WriteRequest(id, method, params)

	if err != nil {
		c.lock.Lock()
		delete(c.reqMap, id)
		c.lock.Unlock()
		return nil, err
	}

	var resp *Response

	if c.timeout > 0 {
		select {
		case resp = <-ch:
		case <-time.After(c.timeout):
			c.lock.Lock()
			delete(c.reqMap, id)
			c.lock.Unlock()
			return nil, ErrTimeout
		}
	} else {
		resp = <-ch
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Result, nil
}

func (c *Client) onResponse(resp *Response) error {
	c.lock.Lock()
	ch, ok := c.reqMap[resp.Id]
	delete(c.reqMap, resp.Id)
	c.lock.Unlock()

	if !ok { // TODO error
		return nil
	}

	ch <- resp

	return nil
}

func (c *Client) call(fn reflect.Value, method string, inArgs []reflect.Value) []reflect.Value {
	params := make([]interface{}, len(inArgs))
	for i := 0; i < len(inArgs); i++ {
		params[i] = inArgs[i].Interface()
	}

	out, err := c.CallRemote(method, params)

	return c.returnCall(fn, out, err)
}

func (c *Client) returnCall(fn reflect.Value, out interface{}, err error) []reflect.Value {
	var outNum = fn.Type().NumOut()
	var outs = make([]reflect.Value, 0, outNum)

	if err != nil { // return err
		return c.returnCallError(fn, err)
	}

	outv := reflect.ValueOf(out)
	if outv.Kind() == reflect.Array { // array
		for i := 0; i < outv.Len(); i++ {
			outs = append(outs, outv.Index(i))
		}
	} else {
		outs = append(outs, outv)
	}
	outs = append(outs, reflect.Zero(fn.Type().Out(outNum-1))) // zero value for last error

	if len(outs) != outNum {
		return c.returnCallError(fn, fmt.Errorf("invalid out len, %v != %v, %#v", len(outs), outNum, out))
	}

	return outs
}

func (c *Client) returnCallError(fn reflect.Value, err error) []reflect.Value {
	var outNum = fn.Type().NumOut()
	var outs = make([]reflect.Value, outNum)

	for i := 0; i < outNum-1; i++ {
		outs[i] = reflect.Zero(fn.Type().Out(i))
	}
	outs[outNum-1] = reflect.ValueOf(&err).Elem()
	return outs
}
