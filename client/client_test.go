package client

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMakeRequest(t *testing.T) {
	c := newClient(nil)
	wreq, err := c.makeRequest(Request{
		Group: "system", Service: "network", Version: "1.0",
		Method: "Add", Params: []interface{}{1, 2},
	})
	if err != nil {
		t.Fatalf("makeRequest: %v", err)
	}
	if wreq.Version != "2.0" {
		t.Errorf("version = %q, want 2.0", wreq.Version)
	}
	if wreq.Group != "system" || wreq.Service != "network" ||
		wreq.ServiceVersion != "1.0" || wreq.Method != "Add" {
		t.Errorf("addressing fields wrong: %+v", wreq)
	}
	if string(wreq.Params) != "[1,2]" {
		t.Errorf("params = %s, want [1,2]", wreq.Params)
	}
	if len(wreq.ID) == 0 {
		t.Error("id should be set")
	}
}

func TestMakeRequest_RequiresServiceAndMethod(t *testing.T) {
	c := newClient(nil)
	if _, err := c.makeRequest(Request{Method: "M"}); err == nil {
		t.Error("missing service should error")
	}
	if _, err := c.makeRequest(Request{Service: "S"}); err == nil {
		t.Error("missing method should error")
	}
}

func TestMakeRequest_NoParams(t *testing.T) {
	c := newClient(nil)
	wreq, err := c.makeRequest(Request{Service: "s", Method: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if wreq.Params != nil {
		t.Errorf("params should be nil when none given, got %s", wreq.Params)
	}
}

func TestNextID_Monotonic(t *testing.T) {
	c := newClient(nil)
	a := c.nextID()
	b := c.nextID()
	if b <= a {
		t.Errorf("ids not monotonic: %d then %d", a, b)
	}
}

func TestDecodeResult_Success(t *testing.T) {
	resp := &wireResponse{Result: json.RawMessage(`"hello"`)}
	var out string
	if err := decodeResult(resp, &out); err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("out = %q, want hello", out)
	}
}

func TestDecodeResult_Error(t *testing.T) {
	resp := &wireResponse{Error: &Error{Code: -1, Message: "boom"}}
	err := decodeResult(resp, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := AsError(err)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if e.ErrorCode() != -1 || e.ErrorMessage() != "boom" {
		t.Errorf("unexpected error: %+v", e)
	}
}

func TestDecodeResult_NilResultDest(t *testing.T) {
	resp := &wireResponse{Result: json.RawMessage(`123`)}
	if err := decodeResult(resp, nil); err != nil {
		t.Errorf("nil dest should be a no-op, got %v", err)
	}
}

func TestError_Interface(t *testing.T) {
	var err error = &Error{Code: -32601, Message: "method not found"}
	if err.Error() == "" {
		t.Error("Error() should be non-empty")
	}
	// errors.As interop via AsError
	e, ok := AsError(err)
	if !ok || e.Code != -32601 {
		t.Errorf("AsError failed: %v %v", e, ok)
	}
	// a non-*Error does not match
	if _, ok := AsError(errors.New("plain")); ok {
		t.Error("plain error should not match AsError")
	}
}

func TestError_Data(t *testing.T) {
	e := &Error{Code: -1, Message: "m", Data: json.RawMessage(`{"n":5}`)}
	var payload struct {
		N int `json:"n"`
	}
	has, err := e.ErrorData(&payload)
	if err != nil {
		t.Fatal(err)
	}
	if !has || payload.N != 5 {
		t.Errorf("ErrorData wrong: has=%v payload=%+v", has, payload)
	}

	// no data
	e2 := &Error{Code: -1}
	has, err = e2.ErrorData(&payload)
	if err != nil || has {
		t.Errorf("expected no data, got has=%v err=%v", has, err)
	}
}

func TestEncodeDecodeRoundTrip_Single(t *testing.T) {
	reqs := []*wireRequest{{Version: "2.0", Service: "s", Method: "m", ID: json.RawMessage(`1`)}}
	body, isBatch, err := encodeRequests(reqs)
	if err != nil {
		t.Fatal(err)
	}
	if isBatch {
		t.Error("single request should not be batch")
	}
	// a single object, not an array
	if body[0] != '{' {
		t.Errorf("single encoding should be an object, got %s", body)
	}
}

func TestEncodeRequests_Batch(t *testing.T) {
	reqs := []*wireRequest{
		{Version: "2.0", Service: "a", Method: "m", ID: json.RawMessage(`1`)},
		{Version: "2.0", Service: "b", Method: "n", ID: json.RawMessage(`2`)},
	}
	body, isBatch, err := encodeRequests(reqs)
	if err != nil {
		t.Fatal(err)
	}
	if !isBatch {
		t.Error("multiple requests should be batch")
	}
	if body[0] != '[' {
		t.Errorf("batch encoding should be an array, got %s", body)
	}
}

func TestBatchCall_Empty(t *testing.T) {
	c := newClient(nil)
	if err := c.BatchCall(context.Background(), nil); err != nil {
		t.Errorf("empty batch should be a no-op, got %v", err)
	}
}
