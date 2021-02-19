package anserpc

import (
	"fmt"
)

const (
	_defErrCode = 32100
)

var (
	_errMethodNotFound = StatusError{
		code: 32101,
		err:  "method not found",
	}

	_errInvalidMessage = StatusError{
		code: 32102,
		err:  "invalid message",
	}

	_errJSONContent = StatusError{
		code: 32103,
		err:  "invalid JSON content",
	}
)

type StatusError struct {
	code int
	err  string
}

func (s StatusError) ErrorCode() int {
	return s.code
}

func (s StatusError) ErrorMessage() string {
	return s.err
}

func (s StatusError) Error() string {
	return fmt.Sprintf("status error <%d>: %s", s.code, s.err)
}
