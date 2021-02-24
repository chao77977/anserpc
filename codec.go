package anserpc

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
)

const (
	_defJsonRpcVersion = "2.0"
)

type jsonMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	Group   string          `json:"group,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
}

func (m *jsonMessage) doValidate() error {
	// TODO

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
	} else if e, ok := err.(ResultCodeError); ok {
		msg.Error.Code = e.ErrorCode()
		msg.Error.Message = e.ErrorMessage()
	} else if e, ok := err.(ResultDataError); ok {
		msg.Error.Data = e.ErrorData()
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
