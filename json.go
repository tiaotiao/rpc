package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

//////////////////////////////////////////////////////////////////

type JsonCodec struct {
	conn io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder

	inVals map[string][]interface{} // map[method][]paramTypes
	inLock sync.RWMutex

	outVals map[string]interface{} // map[method]resultType
	outLock sync.RWMutex

	reqIds  map[int64]string // map[reqId]method
	reqLock sync.RWMutex
}

func NewJsonCodec(conn io.ReadWriteCloser) *JsonCodec {
	c := new(JsonCodec)
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	c.inVals = make(map[string][]interface{})
	c.outVals = make(map[string]interface{})
	return c
}

func (c *JsonCodec) WriteRequest(id int64, method string, params []interface{}) (err error) {
	d := rpcdata{Id: id, Method: method, Params: params}
	err = c.enc.Encode(&d) // encode and write
	if err != nil {
		return err
	}
	// record the request id
	c.reqLock.Lock()
	c.reqIds[id] = method
	c.reqLock.Unlock()
	return err
}

func (c *JsonCodec) WriteResponse(id int64, result interface{}, e *Error) (err error) {
	d := rpcdata{Id: id, Result: result, Error: e}
	err = c.enc.Encode(&d) // encode and write
	return err
}

func (c *JsonCodec) Read() (req *Request, resp *Response, err error) {
	var r = struct {
		Id        int64             `json:"id"`
		Method    string            `json:"method"`
		RawParams []json.RawMessage `json:"params"`
		RawResult json.RawMessage   `json:"result"`
		Error     *Error            `json:"error"`
	}{}
	err = c.dec.Decode(&r) // read and decode
	if err != nil {
		return
	}

	if r.Method != "" { // it's a request
		c.inLock.RLock()
		inVals, ok := c.inVals[r.Method]
		c.inLock.RUnlock()
		if !ok {
			// method is not found, bad request
			err = fmt.Errorf("method is not found. %#v", r)
			return
		}

		if len(r.RawParams) != len(inVals) {
			err = fmt.Errorf("num of params not match %v != %v. %#v", len(r.RawParams), len(inVals), r)
			return
		}

		// unmarshal params
		params := make([]interface{}, len(inVals))
		for i, raw := range r.RawParams {
			in := inVals[i]
			err = json.Unmarshal(raw, &in)
			if err != nil {
				// json error
				return
			}
			params[i] = in
		}

		req = &Request{Id: r.Id, Method: r.Method, Params: params}

	} else { // it's a response
		c.reqLock.Lock()
		method, ok := c.reqIds[r.Id]
		delete(c.reqIds, r.Id)
		c.reqLock.Unlock()
		if !ok {
			// id is not found, bad response
			err = fmt.Errorf("id is not found, bad response. %#v", r)
			return
		}

		c.outLock.RLock()
		result, ok := c.outVals[method]
		c.outLock.RUnlock()
		if !ok {
			return fmt.Errorf("method not found. %v, %#v", method, r)
		}

		// unmarshal result
		err = json.Unmarshal(r.RawResult, &result)
		if err != nil {
			return
		}

		resp = &Response{Id: r.Id, Result: result, Error: r.Error}
	}

	return
}

func (c *JsonCodec) OnRegister(method string, in []interface{}, out interface{}) error {
	// TODO validate

	c.inLock.Lock()
	c.inVals[method] = in
	c.inLock.Unlock()

	c.outLock.Lock()
	c.outVals[method] = out
	c.outLock.Unlock()

	return nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

type jsonProto struct {
	Id        int64           `json:"id"`
	Method    string          `json:"method,omitempty"`
	RawParams json.RawMessage `json:"params,omitempty"`
	Params    []interface{}   `json:"-"`
	RawResult json.RawMessage `json:"result,omitempty"`
	Result    interface{}     `json:"-"`
	Error     *Error          `json:"error,omitempty"`
}
