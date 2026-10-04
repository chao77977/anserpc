package anserpc

import (
	"context"
	"testing"
)

// testSvc is a sample receiver exercising the supported method shapes.
type testSvc struct{}

func (s *testSvc) NoReturn()                         {}
func (s *testSvc) ErrOnly() error                    { return nil }
func (s *testSvc) ResultAndErr() (string, error)     { return "ok", nil }
func (s *testSvc) WithCtx(ctx context.Context) error { return nil }
func (s *testSvc) WithCtxArg(ctx context.Context, n int) (int, error) {
	return n + 1, nil
}
func (s *testSvc) unexported() {} // must be ignored

func newReg() *serviceRegistry {
	return newServiceRegistry()
}

func TestRegistry_BuiltInRegistered(t *testing.T) {
	sr := newReg()
	// The built-in service is public and should resolve.
	if cb := sr.callback("", "built-in", "1.0", "Hello"); cb == nil {
		t.Fatal("built-in Hello should be resolvable")
	}
}

func TestRegistry_RegisterAndResolve(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{
		Group:    "system",
		Service:  "demo",
		Version:  "1.0",
		Public:   true,
		Receiver: &testSvc{},
	})

	// All exported methods resolve.
	for _, m := range []string{"NoReturn", "ErrOnly", "ResultAndErr", "WithCtx", "WithCtxArg"} {
		if cb := sr.callback("system", "demo", "1.0", m); cb == nil {
			t.Errorf("method %q should resolve", m)
		}
	}

	// Unexported method is not registered.
	if cb := sr.callback("system", "demo", "1.0", "unexported"); cb != nil {
		t.Error("unexported method should not resolve")
	}
}

func TestRegistry_CaseInsensitive(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{
		Group: "System", Service: "Demo", Version: "1.0",
		Public: true, Receiver: &testSvc{},
	})

	// Group, service and method all match case-insensitively.
	if cb := sr.callback("SYSTEM", "DEMO", "1.0", "noreturn"); cb == nil {
		t.Fatal("lookup should be case-insensitive")
	}
}

func TestRegistry_PrivateServiceNotResolvable(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{
		Group: "g", Service: "secret", Version: "1.0",
		Public: false, Receiver: &testSvc{},
	})

	if cb := sr.callback("g", "secret", "1.0", "NoReturn"); cb != nil {
		t.Fatal("private service must not be resolvable via callback")
	}
}

func TestRegistry_VersionedServices(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{Service: "svc", Version: "1.0", Public: true, Receiver: &testSvc{}})
	sr.registerWithAPI(&API{Service: "svc", Version: "2.0", Public: true, Receiver: &testSvc{}})

	if cb := sr.callback("", "svc", "1.0", "ErrOnly"); cb == nil {
		t.Error("v1.0 should resolve")
	}
	if cb := sr.callback("", "svc", "2.0", "ErrOnly"); cb == nil {
		t.Error("v2.0 should resolve")
	}
}

func TestRegistry_UnknownReturnsNil(t *testing.T) {
	sr := newReg()
	if cb := sr.callback("", "nope", "", "X"); cb != nil {
		t.Error("unknown service should return nil callback")
	}
}

func TestRegistry_Modules(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{
		Group: "system", Service: "demo", Version: "1.0",
		Public: true, Receiver: &testSvc{},
	})
	mods := sr.modules()
	if len(mods) == 0 {
		t.Fatal("modules() should list registered methods")
	}
	// Must include the built-in and the demo service somewhere.
	var hasDemo bool
	for _, m := range mods {
		if contains(m, "demo") {
			hasDemo = true
		}
	}
	if !hasDemo {
		t.Errorf("modules() missing demo service: %v", mods)
	}
}

func TestMakeCallback_Signatures(t *testing.T) {
	sr := newReg()
	sr.registerWithAPI(&API{Service: "svc", Public: true, Receiver: &testSvc{}})

	cb := sr.callback("", "svc", "", "WithCtxArg")
	if cb == nil {
		t.Fatal("WithCtxArg should resolve")
	}
	if !cb.hasCtx {
		t.Error("WithCtxArg should be detected as having a context param")
	}
	if len(cb.argTypes) != 1 {
		t.Errorf("WithCtxArg argTypes = %d, want 1", len(cb.argTypes))
	}
	if cb.returnType != 1 {
		t.Errorf("WithCtxArg returnType = %d, want 1 (result+error)", cb.returnType)
	}

	cbNo := sr.callback("", "svc", "", "NoReturn")
	if cbNo.returnType != -1 {
		t.Errorf("NoReturn returnType = %d, want -1", cbNo.returnType)
	}
	cbErr := sr.callback("", "svc", "", "ErrOnly")
	if cbErr.returnType != 0 {
		t.Errorf("ErrOnly returnType = %d, want 0", cbErr.returnType)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
