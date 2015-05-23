package rpc

import (
	"fmt"
	"sync"
)

type Methods struct {
	inputs map[string]interface{} // map[method]paramTypes
	inLock sync.RWMutex

	outputs map[string]interface{} // map[method]resultType
	outLock sync.RWMutex
}

func newMethods() *Methods {
	m := new(Methods)
	m.inputs = make(map[string]interface{})
	m.outputs = make(map[string]interface{})
	return m
}

func (m *Methods) Register(method string, in interface{}, out interface{}) error {
	if method == "" {
		return fmt.Errorf("method is empty")
	}
	if in == nil {
		return fmt.Errorf("input is nil")
	}
	if out == nil {
		return fmt.Errorf("output is nil")
	}

	m.inLock.Lock()
	m.outLock.Lock()

	defer m.inLock.Unlock()
	defer m.outLock.Unlock()

	if _, exist := m.inputs[method]; exist {
		return fmt.Errorf("method exists '%v'", method)
	}

	m.inputs[method] = in
	m.outputs[method] = out

	return nil
}

func (m *Methods) Unregister(method string) error {
	m.inLock.Lock()
	m.outLock.Lock()

	defer m.inLock.Unlock()
	defer m.outLock.Unlock()

	if _, exist := m.inputs[method]; !exist {
		return fmt.Errorf("method not exists '%v'", method)
	}

	delete(m.inputs, method)
	delete(m.outputs, method)

	return nil
}

func (m *Methods) In(method string) (interface{}, bool) {
	m.inLock.RLock()
	in, ok := m.inputs[method]
	m.inLock.RUnlock()
	return in, ok
}

func (m *Methods) Out(method string) (interface{}, bool) {
	m.outLock.RLock()
	out, ok := m.outputs[method]
	m.outLock.RUnlock()
	return out, ok
}

func (m *Methods) Names() []string {
	var methods []string
	m.inLock.RLock()
	for method, _ := range m.inputs {
		methods = append(methods, method)
	}
	m.inLock.RUnlock()
	return methods
}

// func (m *Methods) RegisterFunc(method string, fn interface{}) error {
// 	if method == "" {
// 		return fmt.Errorf("method is empty")
// 	}
// 	if fn == nil {
// 		return fmt.Errorf("func is nil")
// 	}

// 	// get params from fn
// 	in, out, err := m.checkFunc(fn)
// 	if err != nil {
// 		return fmt.Errorf("register '%v' %v", method, err.Error())
// 	}

// 	return m.Register(method, in, out)
// }

// // fn must like this: func(input) (output, error)
// func (m *Methods) checkFunc(fn interface{}) (in interface{}, out interface{}, err error) {
// 	t := reflect.TypeOf(fn)

// 	// must be a Func
// 	if t.Kind() != reflect.Func {
// 		err = fmt.Errorf("not a function type")
// 		return
// 	}

// 	// check input
// 	nIn := t.NumIn()
// 	if nIn > 1 {
// 		err = fmt.Errorf("too many input params")
// 		return
// 	}

// 	if nIn == 1 {
// 		it := t.In(0)
// 		v := reflect.New(it).Elem()
// 		in = v.Interface()
// 	}

// 	// check output
// 	nOut := t.NumOut()
// 	if nOut > 2 {
// 		err = fmt.Errorf("too many output params")
// 		return
// 	}
// 	if nOut == 0 {
// 		err = fmt.Errorf("the last output param must be an error")
// 		return
// 	}

// 	// check the last output param
// 	last := t.Out(nOut - 1)
// 	if last.Kind() != reflect.Interface {
// 		return fmt.Errorf("the last output param must be an error")
// 	}
// 	_, b := last.MethodByName("Error")
// 	if !b {
// 		return fmt.Errorf("the last output param must be an error")
// 	}

// 	if nOut == 2 {
// 		it := t.Out(0)
// 		v := reflect.New(it).Elem()
// 		out = v.Interface()
// 	}

// 	return
// }
