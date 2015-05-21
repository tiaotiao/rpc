package rpc

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRpc(t *testing.T) {
	var err error

	// init rpc

	cliRpc, svrRpc := newTestRpc(t)

	err = svrRpc.Server.Register(&svrHandler{})
	if err != nil {
		t.Fatal(err.Error())
	}

	var cli = cliCaller{}
	err = cliRpc.Client.MakeProto(&cli)
	if err != nil {
		t.Fatal(err.Error())
	}

	////////////////////////////////////////

	// echo
	str := "abcde"
	ret, err := cli.Echo(str)
	if err != nil {
		t.Fatal(err.Error())
	}
	if ret != str {
		t.Fatal("not match", str, ret)
	}

	// basic
	ok, err := cli.Basic(1001, "basic", true, 1.2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !ok {
		t.Fatal("basic err")
	}

	// bad request
	err = cli.BadRequest("my bad")
	if err != nil {
		if err.Error() != "bad request" {
			t.Fatal("bad request", err.Error())
		}
	}

	// no output
	err = cli.NoOutput(10, 20)
	if err != nil {
		t.Fatal("no output error:", err.Error())
	}

	// struct
	s, err := cli.Struct(st{1, "do something"})
	if err != nil {
		t.Fatal("struct", err.Error())
	}
	if s == nil || s.Id != 2 || s.Msg != "ok" {
		t.Fatal("struct", s)
	}

}

type svrHandler struct {
}

func (h *svrHandler) Echo(s string) (string, error) {
	return s, nil
}

func (h *svrHandler) Basic(id int64, name string, b bool, f float64) (bool, error) {
	return true, nil
}

func (h *svrHandler) BadRequest(name string) error {
	return fmt.Errorf("bad request")
}

func (h *svrHandler) NoOutput(ids []int64) error {
	return nil
}

func (h *svrHandler) Struct(s st) (*st, error) {
	s.Id += 1
	s.Msg = "ok"
	return &s, nil
}

type st struct {
	Id  int64
	Msg string
}

type cliCaller struct {
	Echo       func(i string) (string, error)
	NoOutput   func(...int64) error
	Basic      func(id int64, name string, b bool, f float64) (bool, error)
	BadRequest func(string) error
	Struct     func(s st) (*st, error)
}

/////////////////////////////////////////////////////////////////

func TestRpcCallRemote(t *testing.T) {
	var err error
	cliRpc, svrRpc := newTestRpc(t)

	// register func
	fn := func(a, b int) (int, error) {
		return a + b, nil
	}
	var cn func(a, b int) (int, error)

	err = cliRpc.Client.MakeFunc("addFunc", &cn)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = svrRpc.Server.RegisterFunc("addFunc", fn)
	if err != nil {
		t.Fatal(err.Error())
	}

	// call
	err = callAndCheck(cliRpc, "addFunc", []interface{}{10, 20}, 30, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// call invalid params
	err = callAndCheck(cliRpc, "addFunc", []interface{}{"ab", 20}, nil, ErrInvalidParams)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = callAndCheck(cliRpc, "addFunc", []interface{}{10, 20, 30}, nil, ErrInvalidParams)
	if err != nil {
		t.Fatal(err.Error())
	}

	// method not found
	err = callAndCheck(cliRpc, "subFunc", []interface{}{30, 20}, nil, ErrMethodNotFound)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func callAndCheck(r *Rpc, method string, params []interface{}, expect interface{}, expErr error) error {
	out, err := r.Client.CallRemote(method, params)
	if !reflect.DeepEqual(err, expErr) {
		return fmt.Errorf("expErr not match: %#v, %#v", expErr, err)
	}
	if !reflect.DeepEqual(expect, out) {
		return fmt.Errorf("result not match: %#v, %#v", out, expect)
	}
	return nil
}

func newTestRpc(t *testing.T) (*Rpc, *Rpc) {
	cliConn, svrConn, err := newTestConn()
	if err != nil {
		t.Fatal(err.Error())
	}

	cliRpc := NewRpc(cliConn)
	svrRpc := NewRpc(svrConn)

	return cliRpc, svrRpc
}

func newTestConn() (net.Conn, net.Conn, error) {
	var addr = "127.0.0.1:9462"

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	var cliConn net.Conn
	var svrConn net.Conn
	var sc net.Conn
	var lock sync.RWMutex

	go func() {
		c, _ := lis.Accept()
		lis.Close()

		lock.Lock()
		sc = c
		lock.Unlock()
	}()

	cliConn, err = net.DialTimeout("tcp", addr, time.Second)

	<-time.After(time.Millisecond * 10)

	lock.RLock()
	svrConn = sc
	lock.RUnlock()

	if err != nil {
		return nil, nil, err
	}

	return cliConn, svrConn, nil
}
