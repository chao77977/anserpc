package anserpc

import (
	"errors"
)

func main() {
	app := New(
		WithRPCEndpoint("0.0.0.0", 56789),
		WithIPCEndpoint("/var/run/anser.sock"),
	)

	// register service
	app.Register("system", "network", "1.0", true, &network{})
	app.Register("system", "storage", "1.0", false, &storage{})

	// application starts
	app.Run()
}

// services and their methods
// service network
type network struct{}

func (n *network) Ping() error {
	return errors.New("unknown host")
}

func (n *network) IP() (string, error) {
	return "10.0.0.2", nil
}

func (n *network) Restart() {}

// service storage
type storage struct{}

func (s *storage) Add() error { return &myErr{} }

type myErr struct{}

func (e *myErr) Error() string          { return e.ErrorMessage() }
func (e *myErr) ErrorCode() int         { return -1 }
func (e *myErr) ErrorMessage() string   { return "error message" }
func (e *myErr) ErrorData() interface{} { return struct{}{} }
