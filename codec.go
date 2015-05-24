package rpc

import ()

type Codec interface {
	WriteRequest(*Request) (err error)
	WriteResponse(*Response) (err error)

	Read() (req *Request, resp *Response, err error)
	ParseValue(val interface{}, dst interface{}) error

	Close() error
}

// func OnRegisterFunc(c Codec, method string, f interface{}) error {
// 	t := reflect.TypeOf(f)

// 	// must be a Func
// 	if t.Kind() != reflect.Func {
// 		return fmt.Errorf("'%v' is not a function", method)
// 	}

// 	// must return the last param as an error
// 	nOut := t.NumOut()
// 	if nOut == 0 || t.Out(nOut-1).Kind() != reflect.Interface {
// 		return fmt.Errorf("the last output param must be an error")
// 	}
// 	_, b := t.Out(nOut - 1).MethodByName("Error")
// 	if !b {
// 		return fmt.Errorf("the last output param must be an error")
// 	}

// 	// get inputs
// 	var in []interface{}
// 	for i := 0; i < t.NumIn(); i++ {
// 		it := t.In(i)
// 		switch it.Kind() {
// 		case reflect.Chan, reflect.Func, reflect.UnsafePointer:
// 			return fmt.Errorf("input param [%v] %v type not support", i, it.Kind().String())
// 		}

// 		v := reflect.New(it).Elem()
// 		in = append(in, v.Interface())
// 	}

// 	// get outputs
// 	var out interface{}
// 	numOut := t.NumOut() - 1 // ignore the last output param which is known as an error type
// 	if numOut > 1 {
// 		return fmt.Errorf("too many output params")
// 	}
// 	if numOut == 1 {
// 		it := t.Out(0)
// 		v := reflect.New(it).Elem()
// 		out = v.Interface()
// 	}

// 	// register to codec
// 	return c.OnRegister(method, in, out)
// }
