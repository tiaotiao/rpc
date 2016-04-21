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
	cliRpc, svrRpc := newTestRpc(t)

	err = svrRpc.Server.Register(&svrHandler{})
	if err != nil {
		t.Error(err.Error())
	}

	var cli = cliCaller{}
	err = cliRpc.Client.MakeClient(&cli)
	if err != nil {
		t.Error(err.Error())
	}

	if cli.Echo == nil {
		t.Fatal("func is nil")
	}

	// echo
	str := "abcde"
	ret, err := cli.Echo(str)
	if err != nil {
		t.Fatal(err.Error())
	}
	if ret != str {
		t.Fatal("not match", str, ret)
	}

	// deliver
	err = cli.Deliver(fooType{"foo", 12.34})
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO more tests
}

// -----------------------------------------------

type cliCaller struct {
	Echo    func(i string) (string, error)
	Deliver func(f fooType) error
}

// -----------------------------------------------

type svrHandler struct {
}

func (h *svrHandler) Echo(s string) (string, error) {
	return s, nil
}

func (h *svrHandler) Deliver(f fooType) error {
	return nil
}

/////////////////////////////////////////////////////////////////

func TestRpcCallRemote(t *testing.T) {
	var err error
	cliRpc, svrRpc := newTestRpc(t)

	// register func
	err = svrRpc.Server.RegisterFunc("addFunc", func(a, b int) (int, error) {
		return a + b, nil
	})
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

	var pr interface{}

	if expect != nil {
		p := reflect.New(reflect.TypeOf(expect))
		pr = p.Interface()
	}

	err := r.Client.CallRemote(method, params, pr)

	if !reflect.DeepEqual(err, expErr) {
		return fmt.Errorf("expErr not match: %#v, %#v", expErr, err)
	}

	if expect != nil {
		out := reflect.ValueOf(pr).Elem().Interface()

		if !reflect.DeepEqual(expect, out) {
			return fmt.Errorf("result not match: %#v, %#v", out, expect)
		}
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
