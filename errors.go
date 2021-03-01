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

	_errMethodCrashed = StatusError{
		code: 32108,
		err:  "method running crash",
	}

	_errInvalidParams = StatusError{
		code: 32109,
		err:  "invalid params message",
	}

	_errTooManyParams = StatusError{
		code: 32110,
		err:  "too many params",
	}

	_errMissingValueParams = StatusError{
		code: 32111,
		err:  "missing value for params",
	}

	_errHandleTimeout = StatusError{
		code: 32112,
		err:  "handling message timeout",
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
