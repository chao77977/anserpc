package anserpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"
)

const (
	_defJsonRpcVersion = "2.0"
)

type jsonMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	Group   string          `json:"group,omitempty"`
	Service string          `json:"service,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
}

func (m *jsonMessage) doValidate() error {
	if m.Version != _defJsonRpcVersion {
		return _errProtoVersion
	}

	if m.Service == "" || m.Method == "" {
		return _errProtoServiceOrMethodNotFound
	}

	return nil
}

func (m *jsonMessage) hasErr() bool {
	return m.Error != nil
}

func (m *jsonMessage) errResponse(err error) *jsonMessage {
	resp := makeJSONErrorMessage(err)
	resp.ID = m.ID
	return resp
}

func (m *jsonMessage) response(result interface{}) *jsonMessage {
	b, err := json.Marshal(result)
	if err != nil {
		_xlog.Error("parse json error", "err", err)
		return m.errResponse(_errJSONContent)
	}

	return &jsonMessage{
		Version: m.Version,
		ID:      m.ID,
		Result:  b,
	}
}

func (m *jsonMessage) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return _errJSONContent.Error()
	}

	return string(b)
}

func zeroArgs(args []reflect.Value, types []reflect.Type) ([]reflect.Value, error) {
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, _errMissingValueParams
		}

		args = append(args, reflect.Zero(types[i]))
	}

	return args, nil
}

func (m *jsonMessage) retrieveArgs(types []reflect.Type) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, len(types))
	dec := json.NewDecoder(bytes.NewReader(m.Params))
	tok, err := dec.Token()
	if err == io.EOF || tok == nil && err == nil {
		return zeroArgs(args, types)
	}

	if err != nil {
		return nil, _errInvalidParams
	}

	if tok != nil && tok != json.Delim('[') {
		return nil, _errInvalidParams
	}

	for i := 0; dec.More(); i++ {
		if i > len(types) {
			return nil, _errTooManyParams
		}

		v := reflect.New(types[i])
		if err := dec.Decode(v.Interface()); err != nil {
			return nil, _errInvalidParams
		}

		if v.IsNil() && types[i].Kind() != reflect.Ptr {
			return nil, _errMissingValueParams
		}

		args = append(args, v.Elem())
	}

	_, err = dec.Token()
	if err != nil {
		return nil, _errInvalidParams
	}

	return zeroArgs(args, types)
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (j *jsonError) ErrorCode() int {
	return j.Code
}

func (j *jsonError) Error() string {
	return j.ErrorMessage()
}

func (j *jsonError) ErrorMessage() string {
	return j.Message
}

func (j *jsonError) ErrorData() interface{} {
	return j.Data
}

func makeJSONErrorMessage(err error) *jsonMessage {
	msg := &jsonMessage{
		Version: _defJsonRpcVersion,
		Error: &jsonError{
			Code: _defErrCode,
		},
	}

	if err == nil {
		return msg
	}

	if e, ok := err.(ResultError); ok {
		msg.Error.Code = e.ErrorCode()
		msg.Error.Message = e.ErrorMessage()
		msg.Error.Data = e.ErrorData()
	} else if e, ok := err.(ResultDataError); ok {
		msg.Error.Code = e.ErrorCode()
		msg.Error.Message = e.Error()
		msg.Error.Data = e.ErrorData()
	} else if e, ok := err.(ResultCodeError); ok {
		msg.Error.Code = e.ErrorCode()
		msg.Error.Message = e.Error()
	} else {
		msg.Error.Message = err.Error()
	}

	return msg
}

type jsonCodec struct {
	mu        sync.Mutex
	closeOnce sync.Once
	closeC    chan struct{}
	encode    func(x interface{}) error
	decode    func(x interface{}) error
	conn      Conn
}

func (j *jsonCodec) readBatch() ([]*jsonMessage, bool, error) {
	var rawMsg json.RawMessage
	if err := j.decode(&rawMsg); err != nil {
		_xlog.Error("parse error", "err", err)
		return nil, false, _errInvalidMessage
	}

	isBatch := false
	// skip insignificant whitespace and six structural characters
	// https://www.ietf.org/rfc/rfc4627.txt
	for _, c := range rawMsg {
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}

		if c == '[' {
			isBatch = true
		}

		break
	}

	var msgs []*jsonMessage
	if !isBatch {
		var msg jsonMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			_xlog.Error("parse json error", "err", err)
			return nil, isBatch, _errJSONContent
		}

		msgs = append(msgs, &msg)
	} else {
		dec := json.NewDecoder(bytes.NewReader(rawMsg))
		dec.Token()
		for dec.More() {
			var msg jsonMessage
			if err := dec.Decode(&msg); err != nil {
				_xlog.Error("parse json error", "err", err)
				return nil, isBatch, _errJSONContent
			}

			msgs = append(msgs, &msg)
		}
	}

	return msgs, isBatch, nil
}

func (j *jsonCodec) writeTo(ctx context.Context, x interface{}) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// ctx: more settings
	return j.encode(x)
}

func (j *jsonCodec) close() {
	j.closeOnce.Do(func() {
		close(j.closeC)
		j.conn.Close()
	})
}

func newCodec(conn Conn) *jsonCodec {
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	dec.UseNumber()

	return &jsonCodec{
		closeC: make(chan struct{}),
		encode: enc.Encode,
		decode: dec.Decode,
		conn:   conn,
	}
}
