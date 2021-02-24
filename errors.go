package anserpc

import (
	"fmt"
)

const (
	_defErrCode = 32100
)

var (
	_errInternal = StatusError{
		code: 32101,
		err:  "internal error",
	}

	_errInvalidMessage = StatusError{
		code: 32102,
		err:  "invalid message",
	}

	_errJSONContent = StatusError{
		code: 32103,
		err:  "invalid JSON content",
	}

	_errServiceNameOrVersion = StatusError{
		code: 32104,
		err:  "service name or version not found",
	}

	_errMethodNotFound = StatusError{
		code: 32105,
		err:  "method not found",
	}

	_errResultErrorNotFound = StatusError{
		code: 32106,
		err:  "error of return result not found",
	}

	_errNumOfResult = StatusError{
		code: 32107,
		err:  "too many return results",
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
