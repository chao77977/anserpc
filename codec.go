package anserpc

import (
	"encoding/json"
	"sync"
)

type jsonMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type jsonCodec struct {
	mu        sync.Mutex
	closeOnce sync.Once
	closeC    chan struct{}
	encode    func(v interface{}) error
	decode    func(v interface{}) error
	conn      Conn
}

func (j *jsonCodec) readBatch() ([]*jsonMessage, bool, error) {
	var rawMsg json.RawMessage
	if err := j.decode(&rawMsg); err != nil {
		return nil, false, err
	}

	// TODO
	return nil, false, nil
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
