package anserpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"testing"
	"time"
)

// memConn is an in-memory Conn: reads from a buffer, writes to another.
type memConn struct {
	in  *bytes.Buffer
	out *bytes.Buffer
}

func (m *memConn) Read(p []byte) (int, error)       { return m.in.Read(p) }
func (m *memConn) Write(p []byte) (int, error)      { return m.out.Write(p) }
func (m *memConn) Close() error                     { return nil }
func (m *memConn) SetWriteDeadline(time.Time) error { return nil }

func newMemCodec(input string) (*jsonCodec, *bytes.Buffer) {
	out := &bytes.Buffer{}
	conn := &memConn{in: bytes.NewBufferString(input), out: out}
	return newCodec(conn), out
}

func TestCodec_ReadSingle(t *testing.T) {
	c, _ := newMemCodec(`{"jsonrpc":"2.0","id":1,"service":"s","method":"m"}`)
	msgs, isBatch, err := c.readBatch()
	if err != nil {
		t.Fatalf("readBatch error: %v", err)
	}
	if isBatch {
		t.Error("single object should not be a batch")
	}
	if len(msgs) != 1 || msgs[0].Service != "s" || msgs[0].Method != "m" {
		t.Fatalf("unexpected msgs: %+v", msgs)
	}
}

func TestCodec_ReadBatch(t *testing.T) {
	c, _ := newMemCodec(`[{"jsonrpc":"2.0","id":1,"service":"a","method":"m"},
	                      {"jsonrpc":"2.0","id":2,"service":"b","method":"n"}]`)
	msgs, isBatch, err := c.readBatch()
	if err != nil {
		t.Fatalf("readBatch error: %v", err)
	}
	if !isBatch {
		t.Error("array should be a batch")
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d msgs, want 2", len(msgs))
	}
}

func TestCodec_ReadInvalidJSON(t *testing.T) {
	c, _ := newMemCodec(`{not json`)
	_, _, err := c.readBatch()
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestCodec_WriteTo(t *testing.T) {
	c, out := newMemCodec("")
	msg := &jsonMessage{Version: "2.0", Result: json.RawMessage(`"hi"`)}
	if err := c.writeTo(context.Background(), msg); err != nil {
		t.Fatalf("writeTo error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte(`"hi"`)) {
		t.Fatalf("output missing result: %s", out.String())
	}
}

func TestDoValidate(t *testing.T) {
	cases := []struct {
		name string
		msg  *jsonMessage
		ok   bool
	}{
		{"valid", &jsonMessage{Version: "2.0", Service: "s", Method: "m"}, true},
		{"bad version", &jsonMessage{Version: "1.0", Service: "s", Method: "m"}, false},
		{"no service", &jsonMessage{Version: "2.0", Method: "m"}, false},
		{"no method", &jsonMessage{Version: "2.0", Service: "s"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.doValidate()
			if tc.ok && err != nil {
				t.Errorf("expected valid, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestRetrieveArgs(t *testing.T) {
	intType := reflect.TypeOf(0)
	strType := reflect.TypeOf("")

	t.Run("positional", func(t *testing.T) {
		m := &jsonMessage{Params: json.RawMessage(`[42,"x"]`)}
		args, err := m.retrieveArgs([]reflect.Type{intType, strType})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(args) != 2 || args[0].Int() != 42 || args[1].String() != "x" {
			t.Fatalf("unexpected args: %+v", args)
		}
	})

	t.Run("empty params with no types", func(t *testing.T) {
		m := &jsonMessage{}
		args, err := m.retrieveArgs(nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(args) != 0 {
			t.Fatalf("want 0 args, got %d", len(args))
		}
	})

	t.Run("too many params", func(t *testing.T) {
		m := &jsonMessage{Params: json.RawMessage(`[1,2,3]`)}
		_, err := m.retrieveArgs([]reflect.Type{intType})
		if err == nil {
			t.Fatal("expected error for too many params")
		}
	})

	t.Run("invalid params not array", func(t *testing.T) {
		m := &jsonMessage{Params: json.RawMessage(`{"a":1}`)}
		_, err := m.retrieveArgs([]reflect.Type{intType})
		if err == nil {
			t.Fatal("expected error for non-array params")
		}
	})
}

func TestMakeJSONErrorMessage(t *testing.T) {
	t.Run("plain error uses default code", func(t *testing.T) {
		m := makeJSONErrorMessage(io.EOF)
		if m.Error == nil || m.Error.Code != _defErrCode {
			t.Fatalf("want default code %d, got %+v", _defErrCode, m.Error)
		}
	})

	t.Run("StatusError carries code and message", func(t *testing.T) {
		m := makeJSONErrorMessage(_errMethodNotFound)
		if m.Error.Code != -32601 || m.Error.Message != "method not found" {
			t.Fatalf("unexpected error mapping: %+v", m.Error)
		}
	})

	t.Run("ResultError carries code/message/data", func(t *testing.T) {
		m := makeJSONErrorMessage(&richErr{})
		if m.Error.Code != -1 || m.Error.Message != "rich" || m.Error.Data == nil {
			t.Fatalf("unexpected rich error mapping: %+v", m.Error)
		}
	})
}

type richErr struct{}

func (e *richErr) Error() string          { return "rich" }
func (e *richErr) ErrorCode() int         { return -1 }
func (e *richErr) ErrorMessage() string   { return "rich" }
func (e *richErr) ErrorData() interface{} { return map[string]int{"n": 1} }

func TestCodecSet(t *testing.T) {
	cs := newCodecSet()
	c, _ := newMemCodec("")
	cs.add(c)
	var count int
	cs.each(func(sc serviceCodec) bool { count++; return false })
	if count != 1 {
		t.Fatalf("codecSet each visited %d, want 1", count)
	}
	cs.remove(c)
	count = 0
	cs.each(func(sc serviceCodec) bool { count++; return false })
	if count != 0 {
		t.Fatalf("after remove, each visited %d, want 0", count)
	}
}
