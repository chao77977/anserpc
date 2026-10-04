package anserpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// handlerSvc exercises success, error, and panic paths.
type handlerSvc struct{}

func (s *handlerSvc) Echo(n int) (int, error) { return n, nil }
func (s *handlerSvc) Fail() error             { return errors.New("boom") }
func (s *handlerSvc) Panic() (string, error)  { panic("kaboom") }

func regWith(svc interface{}) *serviceRegistry {
	sr := newServiceRegistry()
	sr.registerWithAPI(&API{Service: "h", Version: "1.0", Public: true, Receiver: svc})
	return sr
}

func dispatch(t *testing.T, sr *serviceRegistry, msg *jsonMessage) *jsonMessage {
	t.Helper()
	h := newHandler(sr, context.Background())
	defer h.close()
	return h.handleMsg(msg)
}

func TestHandler_Success(t *testing.T) {
	sr := regWith(&handlerSvc{})
	msg := &jsonMessage{
		Version: "2.0", ID: json.RawMessage(`1`),
		Service: "h", ServiceVersion: "1.0", Method: "Echo",
		Params: json.RawMessage(`[7]`),
	}
	resp := dispatch(t, sr, msg)
	if resp.hasErr() {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if string(resp.Result) != "7" {
		t.Fatalf("Result = %s, want 7", resp.Result)
	}
}

func TestHandler_MethodError(t *testing.T) {
	sr := regWith(&handlerSvc{})
	msg := &jsonMessage{
		Version: "2.0", ID: json.RawMessage(`1`),
		Service: "h", ServiceVersion: "1.0", Method: "Fail",
	}
	resp := dispatch(t, sr, msg)
	if !resp.hasErr() {
		t.Fatal("expected error response")
	}
	if resp.Error.Message != "boom" {
		t.Fatalf("Error.Message = %q, want boom", resp.Error.Message)
	}
}

func TestHandler_PanicRecovered(t *testing.T) {
	sr := regWith(&handlerSvc{})
	msg := &jsonMessage{
		Version: "2.0", ID: json.RawMessage(`1`),
		Service: "h", ServiceVersion: "1.0", Method: "Panic",
	}
	resp := dispatch(t, sr, msg)
	if !resp.hasErr() {
		t.Fatal("panic should produce an error response, not crash")
	}
	if resp.Error.Code != _errMethodCrashed.ErrorCode() {
		t.Fatalf("Error.Code = %d, want %d", resp.Error.Code, _errMethodCrashed.ErrorCode())
	}
}

func TestHandler_ValidationError(t *testing.T) {
	sr := regWith(&handlerSvc{})
	// Wrong jsonrpc version fails validation before dispatch.
	msg := &jsonMessage{Version: "1.0", Service: "h", Method: "Echo"}
	resp := dispatch(t, sr, msg)
	if !resp.hasErr() {
		t.Fatal("expected validation error")
	}
}

func TestHandler_MethodNotFound(t *testing.T) {
	sr := regWith(&handlerSvc{})
	msg := &jsonMessage{
		Version: "2.0", ID: json.RawMessage(`1`),
		Service: "h", ServiceVersion: "1.0", Method: "DoesNotExist",
	}
	resp := dispatch(t, sr, msg)
	if !resp.hasErr() || resp.Error.Code != _errMethodNotFound.ErrorCode() {
		t.Fatalf("expected method-not-found, got %+v", resp.Error)
	}
}

func TestHandler_Batch(t *testing.T) {
	sr := regWith(&handlerSvc{})
	msgs := []*jsonMessage{
		{Version: "2.0", ID: json.RawMessage(`1`), Service: "h", ServiceVersion: "1.0", Method: "Echo", Params: json.RawMessage(`[1]`)},
		{Version: "2.0", ID: json.RawMessage(`2`), Service: "h", ServiceVersion: "1.0", Method: "Echo", Params: json.RawMessage(`[2]`)},
	}
	h := newHandler(sr, context.Background())
	defer h.close()
	resps := h.handleMsgs(msgs)
	if len(resps) != 2 {
		t.Fatalf("got %d responses, want 2", len(resps))
	}
	if string(resps[0].Result) != "1" || string(resps[1].Result) != "2" {
		t.Fatalf("batch results out of order: %s, %s", resps[0].Result, resps[1].Result)
	}
}

func TestDoHandle_EndToEnd(t *testing.T) {
	sr := regWith(&handlerSvc{})
	input := `{"jsonrpc":"2.0","id":5,"service":"h","service_version":"1.0","method":"Echo","params":[99]}`
	c, out := newMemCodec(input)
	doHandle(context.Background(), c, sr)
	if !jsonContains(out.String(), `"result":99`) {
		t.Fatalf("doHandle output = %s, want result 99", out.String())
	}
}

func jsonContains(s, sub string) bool {
	return contains(s, sub)
}
