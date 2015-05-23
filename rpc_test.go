package rpc

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestRpc(t *testing.T) {
	var err error

	cli, svr := newTestRpc(t)

	if cli == nil || svr == nil {
		t.Fatal("rpc is nil", cli, svr)
	}

	handler := func(method string, params interface{}) (result interface{}, err error) {
		switch method {

		case "echo":
			return params, nil

		case "sum":
			var x int
			a := params.([]int)
			for _, n := range a {
				x += n
			}
			return x, nil

		case "struct":
			e := params.(*Error)
			return e.Code, nil

		default:
			return nil, ErrMethodNotFound
		}
	}

	// handler
	svr.SetHandler(handler)

	// register methods
	register(cli, svr, "echo", "string", "string", t)
	register(cli, svr, "sum", []int{0, 0}, 0, t)
	register(cli, svr, "struct", (*Error)(nil), 0, t)

	var result interface{}

	// echo
	result, err = cli.CallRemote("echo", "hello world")
	if err != nil {
		t.Fatal(err.Error())
	}
	s, ok := result.(string)
	if !ok {
		t.Fatal("result is not string", result)
	}
	if s != "hello world" {
		t.Fatal("echo not match", s)
	}

	// sum
	result, err = cli.CallRemote("sum", []int{10, 20})
	if err != nil {
		t.Fatal(err.Error())
	}
	x, ok := result.(int)
	if !ok {
		t.Fatal("result type error", result)
	}
	if x != 30 {
		t.Fatal("sum error", x)
	}

	// struct
	result, err = cli.CallRemote("struct", NewError(1001, "error"))
	if err != nil {
		t.Fatal(err.Error())
	}
	e, ok := result.(int)
	if !ok {
		t.Fatal("result type error", result)
	}
	if e != 1001 {
		t.Fatal("struct error", e)
	}
}

func register(cli, svr *Rpc, method string, in, out interface{}, t *testing.T) {
	var err error
	err = cli.RegisterMethod(method, in, out)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = svr.RegisterMethod(method, in, out)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func newTestRpc(t *testing.T) (*Rpc, *Rpc) {
	cliConn, svrConn, err := newTestConn()
	if err != nil {
		t.Fatal(err.Error())
	}

	cliRpc := NewRpc(cliConn)
	svrRpc := NewRpc(svrConn)

	go cliRpc.Run()
	go svrRpc.Run()

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
