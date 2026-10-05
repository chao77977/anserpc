package client_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	anserpc "github.com/chao77977/anserpc"
	"github.com/chao77977/anserpc/client"
)

// calcSvc is the server-side service exercised by the integration tests.
type calcSvc struct{}

func (s *calcSvc) Add(a, b int) (int, error) { return a + b, nil }
func (s *calcSvc) Echo(msg string) (string, error) {
	return msg, nil
}
func (s *calcSvc) Boom() error { return &richErr{} }

// richErr implements the server's ResultError interface.
type richErr struct{}

func (e *richErr) Error() string          { return "boom" }
func (e *richErr) ErrorCode() int         { return -42 }
func (e *richErr) ErrorMessage() string   { return "boom" }
func (e *richErr) ErrorData() interface{} { return map[string]int{"n": 7} }

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitTCP(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err == nil {
			c.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server %s not ready", addr)
}

func startHTTPServer(t *testing.T) string {
	t.Helper()
	port := freePort(t)
	app := anserpc.New(
		anserpc.WithRPCEndpoint("127.0.0.1", port),
		anserpc.WithDisableInterruptHandler(),
	)
	app.RegisterService("calc", "1.0", true, &calcSvc{})
	go app.Run()
	t.Cleanup(app.Close)
	addr := net.JoinHostPort("127.0.0.1", itoa(port))
	waitTCP(t, addr)
	return addr
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestIntegration_HTTP_Call(t *testing.T) {
	addr := startHTTPServer(t)
	c, err := client.DialHTTP("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var sum int
	err = c.Call(context.Background(), &sum, client.Request{
		Service: "calc", Version: "1.0", Method: "Add",
		Params: []interface{}{3, 4},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if sum != 7 {
		t.Fatalf("sum = %d, want 7", sum)
	}
}

func TestIntegration_HTTP_Error(t *testing.T) {
	addr := startHTTPServer(t)
	c, _ := client.DialHTTP("http://" + addr)
	defer c.Close()

	err := c.Call(context.Background(), nil, client.Request{
		Service: "calc", Version: "1.0", Method: "Boom",
	})
	if err == nil {
		t.Fatal("expected server error")
	}
	e, ok := client.AsError(err)
	if !ok {
		t.Fatalf("expected *client.Error, got %T", err)
	}
	if e.ErrorCode() != -42 || e.ErrorMessage() != "boom" {
		t.Fatalf("unexpected error: %+v", e)
	}
	var data struct {
		N int `json:"n"`
	}
	has, derr := e.ErrorData(&data)
	if derr != nil || !has || data.N != 7 {
		t.Fatalf("error data wrong: has=%v err=%v data=%+v", has, derr, data)
	}
}

func TestIntegration_HTTP_Batch(t *testing.T) {
	addr := startHTTPServer(t)
	c, _ := client.DialHTTP("http://" + addr)
	defer c.Close()

	var r1, r2 int
	elems := []client.BatchElem{
		{Request: client.Request{Service: "calc", Version: "1.0", Method: "Add", Params: []interface{}{1, 1}}, Result: &r1},
		{Request: client.Request{Service: "calc", Version: "1.0", Method: "Add", Params: []interface{}{10, 10}}, Result: &r2},
	}
	if err := c.BatchCall(context.Background(), elems); err != nil {
		t.Fatalf("BatchCall: %v", err)
	}
	if r1 != 2 || r2 != 20 {
		t.Fatalf("batch results wrong: r1=%d r2=%d", r1, r2)
	}
	for i, e := range elems {
		if e.Error != nil {
			t.Errorf("elem %d unexpected error: %v", i, e.Error)
		}
	}
}

func TestIntegration_WebSocket_Call(t *testing.T) {
	addr := startHTTPServer(t)
	c, err := client.DialWebSocket("ws://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var out string
	err = c.Call(context.Background(), &out, client.Request{
		Service: "calc", Version: "1.0", Method: "Echo",
		Params: []interface{}{"hi"},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out != "hi" {
		t.Fatalf("out = %q, want hi", out)
	}
}

func TestIntegration_IPC_Call(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "anser.sock")
	app := anserpc.New(
		anserpc.WithIPCEndpoint(sock),
		anserpc.WithDisableInterruptHandler(),
	)
	app.RegisterService("calc", "1.0", true, &calcSvc{})
	go app.Run()
	t.Cleanup(app.Close)

	// wait for the socket file
	deadline := time.Now().Add(5 * time.Second)
	var c *client.Client
	var err error
	for time.Now().Before(deadline) {
		c, err = client.DialIPC(sock)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if c == nil {
		t.Fatalf("DialIPC: %v", err)
	}
	defer c.Close()

	var sum int
	err = c.Call(context.Background(), &sum, client.Request{
		Service: "calc", Version: "1.0", Method: "Add",
		Params: []interface{}{8, 9},
	})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if sum != 17 {
		t.Fatalf("sum = %d, want 17", sum)
	}
}

func TestIntegration_ContextCancel(t *testing.T) {
	addr := startHTTPServer(t)
	c, _ := client.DialHTTP("http://" + addr)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	err := c.Call(ctx, nil, client.Request{
		Service: "calc", Version: "1.0", Method: "Add", Params: []interface{}{1, 2},
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
