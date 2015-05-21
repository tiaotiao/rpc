package rpc

import (
	"fmt"
	"reflect"
	"sync"
)

type Server struct {
	codec Codec

	funcs map[string]*reflect.Value
	lock  sync.RWMutex
}

func newServerWithCodec(codec Codec) *Server {
	s := new(Server)
	s.codec = codec
	s.funcs = make(map[string]*reflect.Value)
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

	// register type
	err = OnRegisterFunc(s.codec, method, f)
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
	s.funcs[method] = &v
	s.lock.Unlock()
	return
}

func (s *Server) Handle(method string, params []interface{}) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("Panic: %v", r)
			}
		}
	}()

	// find method
	s.lock.RLock()
	f, ok := s.funcs[method]
	s.lock.RUnlock()
	if !ok {
		return nil, ErrMethodNotFound
	}

	// make input values
	inValues := make([]reflect.Value, len(params))
	for i := 0; i < len(params); i++ {
		if params[i] == nil {
			inValues[i] = reflect.Zero(f.Type().In(i))
		} else {
			inValues[i] = reflect.ValueOf(params[i])
		}
	}

	// check params
	err = s.checkParams(f.Type(), inValues)
	if err != nil {
		return nil, ErrInvalidParams
	}

	// reflect call func
	outs := f.Call(inValues)

	return s.returnResult(outs)
}

func (s *Server) returnResult(outs []reflect.Value) (result interface{}, err error) {
	if len(outs) <= 0 {
		return nil, fmt.Errorf("len(outs) <= 0")
	}

	// check the last error param
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

	// multi outputs (nerver happen for now)
	results := make([]interface{}, len(outs))
	for i := 0; i < len(outs); i++ {
		results[i] = outs[i].Interface()
	}
	return results, nil
}

func (s *Server) onRequest(req *Request, err error) error {
	var result interface{}
	if err == nil {
		result, err = s.Handle(req.Method, req.Params)
	}

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

func (s *Server) checkParams(f reflect.Type, inValues []reflect.Value) error {
	numIn := f.NumIn()
	if numIn != len(inValues) {
		return fmt.Errorf("params len=%v error! need %v", inValues, numIn)
	}

	for i := 0; i < numIn; i++ {
		p := f.In(i)
		if !inValues[i].Type().AssignableTo(p) {
			return fmt.Errorf("param %v (%v) type error! need %v", i, inValues[i].Type().Name(), p.Name())
		}
	}
	return nil
}
