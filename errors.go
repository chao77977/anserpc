package anserpc

import (
	"fmt"
)

var (
	ErrMethodNotFound = StatusError{
		code: 32601,
		err:  "method not found",
	}

	ErrJSONContent = StatusError{
		code: 32602,
		err:  "invalid JSON content",
	}

	ErrRequest = StatusError{
		code: 32603,
		err:  "invalid request",
	}
)

type StatusError struct {
	code int
	err  string
}

func (s StatusError) Error() string {
	return fmt.Sprintf("err %d: %s", s.code, s.err)
}
