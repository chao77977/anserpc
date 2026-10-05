package anserpc

import (
	"encoding/json"
	"testing"
)

func TestOptions_Defaults(t *testing.T) {
	o := defaultOpt()
	if o.log == nil || o.http == nil {
		t.Fatal("defaults should set log and http")
	}
	if !o.http.WebsocketAllowed {
		t.Error("websocket should be allowed by default")
	}
	// default HTTP opt seeds localhost vhost and DELETE/PUT denied methods
	if !o.http.vhosts.Contains("localhost") {
		t.Error("default vhost localhost missing")
	}
	if !o.http.deniedMethods.Contains("DELETE") || !o.http.deniedMethods.Contains("PUT") {
		t.Error("default denied methods missing")
	}
}

func TestOptions_Apply(t *testing.T) {
	o := defaultOpt()

	WithRPCEndpoint("1.2.3.4", 9000).apply(o)
	if o.rpc == nil || o.rpc.host != "1.2.3.4" || o.rpc.port != 9000 {
		t.Fatalf("rpc endpoint not applied: %+v", o.rpc)
	}
	if got := o.rpc.String(); got != "1.2.3.4:9000" {
		t.Errorf("rpc String = %q", got)
	}

	WithIPCEndpoint("/tmp/x.sock").apply(o)
	if string(o.ipc) != "/tmp/x.sock" {
		t.Errorf("ipc = %q", o.ipc)
	}
	if o.ipc.String() != "/tmp/x.sock" {
		t.Errorf("ipc String = %q", o.ipc.String())
	}

	WithDisableInterruptHandler().apply(o)
	if o.intrpt == nil || !o.intrpt.disableInterruptHandler {
		t.Error("interrupt handler disable not applied")
	}

	WithLoggerOpt(quietLogger{}).apply(o)
	if o.log.logger == nil {
		t.Error("logger opt not applied")
	}
}

func TestOptions_DefaultEndpoints(t *testing.T) {
	o := defaultOpt()
	WithDefaultRPCEndpoint().apply(o)
	if o.rpc.host != _defRPCHost || o.rpc.port != _defRPCPort {
		t.Errorf("default rpc endpoint wrong: %+v", o.rpc)
	}
	WithDefaultIPCEndpoint().apply(o)
	if string(o.ipc) != _defIPCPath {
		t.Errorf("default ipc path wrong: %q", o.ipc)
	}
}

func TestOptions_HTTPVhostAndDeniedMethods(t *testing.T) {
	o := defaultOpt()
	WithHTTPVhostOpt("example.com", "").apply(o)
	if !o.http.vhosts.Contains("example.com") {
		t.Error("vhost not applied")
	}
	// empty vhost is skipped (no panic, not added)
	WithHTTPDeniedMethodOpt("PATCH", "BOGUS").apply(o)
	if !o.http.deniedMethods.Contains("PATCH") {
		t.Error("PATCH should be denied")
	}
	if o.http.deniedMethods.Contains("BOGUS") {
		t.Error("unknown method BOGUS should be ignored")
	}
}

func TestOptions_WithLogFileOpt(t *testing.T) {
	o := defaultOpt()
	WithLogFileOpt("/tmp/anser.log", LvlInfo).apply(o)
	if o.log.path != "/tmp/anser.log" || o.log.filterLvl != LvlInfo || !o.log.silent {
		t.Errorf("log file opt wrong: %+v", o.log)
	}
}

func TestAnser_New_AllowFlags(t *testing.T) {
	a := New(
		WithRPCEndpoint("127.0.0.1", 1),
		WithIPCEndpoint("/tmp/anser-test.sock"),
		WithDisableInterruptHandler(),
	)
	if !a.rpcAllowed() {
		t.Error("rpcAllowed should be true")
	}
	if !a.ipcAllowed() {
		t.Error("ipcAllowed should be true")
	}
	// servers not started yet
	if a.statusRPCServer() == _statRunning {
		t.Error("RPC server should not be running before Run/enable")
	}
	if a.statusIPCServer() == _statRunning {
		t.Error("IPC server should not be running before Run/enable")
	}
}

func TestAnser_RegisterWithGroup(t *testing.T) {
	a := New(WithDisableInterruptHandler())
	grp := a.RegisterWithGroup("mygroup")
	grp.Register("svc", "1.0", true, &handlerSvc{})

	if cb := a.sr.callback("mygroup", "svc", "1.0", "Echo"); cb == nil {
		t.Fatal("service registered via group should resolve")
	}
}

func TestAnser_RegisterService(t *testing.T) {
	a := New(WithDisableInterruptHandler())
	a.RegisterService("standalone", "2.0", true, &handlerSvc{})
	if cb := a.sr.callback("", "standalone", "2.0", "Echo"); cb == nil {
		t.Fatal("RegisterService should resolve with empty group")
	}
}

func TestBuiltIn_HelloAndMetrics(t *testing.T) {
	s := builtInService{}
	hi, err := s.Hello()
	if err != nil || hi != "olleh" {
		t.Fatalf("Hello = %q, %v", hi, err)
	}
	m, err := s.Metrics()
	if err != nil {
		t.Fatalf("Metrics error: %v", err)
	}
	// Metrics returns a JSON string.
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(m), &out); err != nil {
		t.Fatalf("Metrics output not valid JSON: %v (%s)", err, m)
	}
}

func TestJSONError_Accessors(t *testing.T) {
	e := &jsonError{Code: -7, Message: "oops", Data: map[string]int{"n": 1}}
	if e.ErrorCode() != -7 {
		t.Errorf("ErrorCode = %d", e.ErrorCode())
	}
	if e.Error() != "oops" || e.ErrorMessage() != "oops" {
		t.Errorf("message accessors wrong: %q / %q", e.Error(), e.ErrorMessage())
	}
	if e.ErrorData() == nil {
		t.Error("ErrorData should be non-nil")
	}
}

func TestMakeJSONErrorMessage_NilAndPlain(t *testing.T) {
	// nil error still yields a default error object.
	m := makeJSONErrorMessage(nil)
	if m.Error == nil || m.Error.Code != _defErrCode {
		t.Fatalf("nil error mapping wrong: %+v", m.Error)
	}
}
