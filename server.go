package rpc

import (
	"fmt"
	"reflect"
	"sync"
)

type Server struct {
	codec Codec

	funcs map[string]reflect.Value
	lock  sync.RWMutex
}

func newServerWithCodec(codec Codec) *Server {
	s := new(Server)
	s.codec = codec
	s.funcs = make(map[string]reflect.Value)
	return s
}

// Register all objects.Funcs to rpc
func (s *Server) Register(object interface{}) (err error) {
	t := reflect.TypeOf(object)
	v := reflect.ValueOf(object)

	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("'object' must be a point of struct")
	}

	var count int
	for i := 0; i < t.NumMethod(); i++ {
		mt := t.Method(i)
		mv := v.Method(i)

		// find a public method
		if mt.PkgPath != "" || !mv.CanInterface() {
			continue
		}

		// register
		err := s.RegisterFunc(mt.Name, mv.Interface())
		if err != nil {
			return err
		}

		count += 1
	}

	if count <= 0 {
		return fmt.Errorf("Register object has no method")
	}

	return nil
}

// Register a function to rpc
func (s *Server) RegisterFunc(method string, f interface{}) (err error) {
	if method == "" {
		return fmt.Errorf("method name is empty")
	}
	if f == nil {
		return fmt.Errorf("func is nil")
	}

	v := reflect.ValueOf(f)
	t := reflect.TypeOf(f)

	// f must be a Func
	if t.Kind() != reflect.Func {
		return fmt.Errorf("'%v' is not a function", method)
	}

	// f must return error as last param
	nOut := t.NumOut()
	if nOut == 0 || t.Out(nOut-1).Kind() != reflect.Interface {
		err = fmt.Errorf("must return error as the last output param")
		return
	}
	_, b := t.Out(nOut - 1).MethodByName("Error")
	if !b {
		err = fmt.Errorf("must return error as the last output param")
		return
	}

	// register type
	err = codecRegisterFuncTypes(s.codec, f)
	if err != nil {
		return err
	}

	// register
	s.lock.Lock()
	if _, ok := s.funcs[method]; ok {
		err = fmt.Errorf("method has been registered")
		s.lock.Unlock()
		return
	}
	s.funcs[method] = v
	s.lock.Unlock()
	return
}

func (s *Server) onRequest(req *Request) (err error) {
	result, err := s.handle(req)
	e, ok := err.(*Error)
	if !ok {
		if err != nil {
			e = NewError(CodeFunctionError, err.Error())
		} else {
			e = nil
		}
	}
	return s.codec.WriteResponse(req.Id, result, e) // encode and write
}

func (s *Server) handle(req *Request) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("Panic: %v", r)
			}
		}
	}()

	method := req.Method

	s.lock.RLock()
	f, ok := s.funcs[method]
	s.lock.RUnlock()

	if !ok {
		return nil, ErrMethodNotFound
	}

	inValues, err := s.buildInValues(f, req)
	if err != nil {
		return nil, ErrInvalidParams
	}

	outs := f.Call(inValues)

	return s.returnResult(outs)
}

func (s *Server) buildInValues(fv reflect.Value, req *Request) (inValues []reflect.Value, err error) {
	f := fv.Type()
	numIn := f.NumIn()
	if numIn != req.Len() {
		return nil, fmt.Errorf("params len=%v error! need %v", req.Len(), numIn)
	}

	inValues = make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		param := req.params[i]

		if param == nil {
			inValues[i] = reflect.Zero(f.In(i))
		} else {
			pv := reflect.New(f.In(i))
			v := pv.Interface()
			err = req.Param(i, v)
			if err != nil {
				return nil, err
			}

			inValues[i] = reflect.ValueOf(pv.Elem().Interface())
		}
	}

	return inValues, nil
}

func (s *Server) returnResult(outs []reflect.Value) (result interface{}, err error) {
	if len(outs) <= 0 {
		return nil, fmt.Errorf("len(outs) <= 0")
	}

	lastErr := outs[len(outs)-1].Interface()
	if lastErr != nil {
		if e, ok := lastErr.(error); !ok {
			return nil, fmt.Errorf("last output arg must be error type")
		} else {
			return nil, e
		}
	}

	outs = outs[:len(outs)-1]

	if len(outs) == 0 {
		return nil, nil
	}
	if len(outs) == 1 {
		return outs[0].Interface(), nil
	}
	results := make([]interface{}, len(outs))
	for i := 0; i < len(outs); i++ {
		results[i] = outs[i].Interface()
	}
	return results, nil
}
