package anserpc

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// intSvc is the service used across integration tests.
type intSvc struct{}

func (s *intSvc) Add(a, b int) (int, error) { return a + b, nil }
func (s *intSvc) Boom() error               { return &richErr{} }

// freePort returns an available localhost TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitHTTP(t *testing.T, addr string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready", addr)
}

func TestIntegration_HTTP(t *testing.T) {
	port := freePort(t)
	app := New(WithRPCEndpoint("127.0.0.1", port), WithDisableInterruptHandler())
	app.RegisterService("calc", "1.0", true, &intSvc{})
	if err := app.enableRPCServer(); err != nil {
		t.Fatalf("enableRPCServer: %v", err)
	}
	defer app.Close()

	addr := app.rs.listenAddr()
	waitHTTP(t, addr)
	url := "http://" + addr

	// success
	body := `{"jsonrpc":"2.0","id":1,"service":"calc","service_version":"1.0","method":"Add","params":[2,3]}`
	resp := httpPost(t, url, body)
	if !contains(resp, `"result":5`) {
		t.Fatalf("Add result wrong: %s", resp)
	}

	// rich error
	body = `{"jsonrpc":"2.0","id":2,"service":"calc","service_version":"1.0","method":"Boom"}`
	resp = httpPost(t, url, body)
	if !contains(resp, `"code":-1`) || !contains(resp, `"message":"rich"`) {
		t.Fatalf("Boom error mapping wrong: %s", resp)
	}

	// built-in Hello
	body = `{"jsonrpc":"2.0","id":3,"service":"built-in","method":"Hello"}`
	resp = httpPost(t, url, body)
	if !contains(resp, `"result":"olleh"`) {
		t.Fatalf("Hello wrong: %s", resp)
	}
}

func TestIntegration_HTTP_InvalidContentType(t *testing.T) {
	port := freePort(t)
	app := New(WithRPCEndpoint("127.0.0.1", port), WithDisableInterruptHandler())
	app.RegisterService("calc", "1.0", true, &intSvc{})
	if err := app.enableRPCServer(); err != nil {
		t.Fatalf("enableRPCServer: %v", err)
	}
	defer app.Close()
	addr := app.rs.listenAddr()
	waitHTTP(t, addr)

	req, _ := http.NewRequest(http.MethodPost, "http://"+addr,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"service":"calc","method":"Add","params":[1,2]}`))
	req.Header.Set("Content-Type", "text/plain")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", r.StatusCode)
	}
}

func TestIntegration_WebSocket(t *testing.T) {
	port := freePort(t)
	app := New(WithRPCEndpoint("127.0.0.1", port), WithDisableInterruptHandler())
	app.RegisterService("calc", "1.0", true, &intSvc{})
	if err := app.enableRPCServer(); err != nil {
		t.Fatalf("enableRPCServer: %v", err)
	}
	defer app.Close()
	addr := app.rs.listenAddr()
	waitHTTP(t, addr)

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	req := map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "service": "calc",
		"service_version": "1.0", "method": "Add", "params": []int{10, 20},
	}
	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("ws write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var out jsonMessage
	if err := conn.ReadJSON(&out); err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if string(out.Result) != "30" {
		t.Fatalf("ws Add result = %s, want 30", out.Result)
	}
}

func TestIntegration_IPC(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "anser.sock")
	app := New(WithIPCEndpoint(sock), WithDisableInterruptHandler())
	app.RegisterService("calc", "1.0", true, &intSvc{})
	if err := app.enableIPCServer(); err != nil {
		t.Fatalf("enableIPCServer: %v", err)
	}
	defer app.Close()

	// wait for socket
	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ {
		if _, serr := os.Stat(sock); serr == nil {
			conn, err = net.Dial("unix", sock)
			if err == nil {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if conn == nil {
		t.Fatalf("could not connect to IPC socket: %v", err)
	}
	defer conn.Close()

	req := `{"jsonrpc":"2.0","id":1,"service":"calc","service_version":"1.0","method":"Add","params":[4,5]}` + "\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("ipc write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	dec := json.NewDecoder(conn)
	var out jsonMessage
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("ipc read: %v", err)
	}
	if string(out.Result) != "9" {
		t.Fatalf("ipc Add result = %s, want 9", out.Result)
	}
}

func httpPost(t *testing.T, url, body string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http post: %v", err)
	}
	defer r.Body.Close()
	buf := &bytes.Buffer{}
	buf.ReadFrom(r.Body)
	return buf.String()
}
