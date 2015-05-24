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
			var msg string
			err = cli.ParseValue(params, &msg)
			if err != nil {
				return nil, err
			}
			return msg, nil

		case "sum":
			var a []int
			err = cli.ParseValue(params, &a)
			if err != nil {
				return nil, err
			}

			var x int
			for _, n := range a {
				x += n
			}
			return x, nil

		case "struct":
			var e *Error
			err = cli.ParseValue(params, &e)
			if err != nil {
				return nil, err
			}

			return e.Code, nil

		default:
			return nil, ErrMethodNotFound
		}
	}

	// handler
	svr.SetHandler(handler)

	// echo
	var s string = ""
	err = cli.CallRemote("echo", "hello world", &s)
	if err != nil {
		t.Fatal(err.Error())
	}
	if s != "hello world" {
		t.Fatal("echo not match", s)
	}

	// sum
	var x int
	err = cli.CallRemote("sum", []int{10, 20}, &x)
	if err != nil {
		t.Fatal(err.Error())
	}
	if x != 30 {
		t.Fatal("sum error", x)
	}

	// struct
	var e int
	err = cli.CallRemote("struct", NewError(1001, "error"), &e)
	if err != nil {
		t.Fatal(err.Error())
	}
	if e != 1001 {
		t.Fatal("struct error", e)
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
