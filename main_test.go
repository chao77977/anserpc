package anserpc

import (
	"os"
	"testing"
)

// quietLogger discards all log output so tests stay clean.
type quietLogger struct{}

func (quietLogger) Crit(string, ...interface{})  {}
func (quietLogger) Warn(string, ...interface{})  {}
func (quietLogger) Error(string, ...interface{}) {}
func (quietLogger) Info(string, ...interface{})  {}
func (quietLogger) Debug(string, ...interface{}) {}

// TestMain initializes the package-global logger before any test runs.
// In production New() does this; unit tests that exercise internals
// directly must initialize it too, otherwise _xlog is nil and logging
// calls panic.
func TestMain(m *testing.M) {
	newSafeLogger(&logOpt{logger: quietLogger{}})
	os.Exit(m.Run())
}
