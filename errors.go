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

	_errProtoVersion = StatusError{
		code: 32104,
		err:  "invalid version",
	}

	_errProtoServiceOrMethodNotFound = StatusError{
		code: 32105,
		err:  "service or method not found",
	}

	_errServiceOrVersion = StatusError{
		code: 32201,
		err:  "service name or version not found",
	}

	_errMethodNotFound = StatusError{
		code: 32202,
		err:  "method not found",
	}

	_errResultErrorNotFound = StatusError{
		code: 32203,
		err:  "error of return result not found",
	}

	_errNumOfResult = StatusError{
		code: 32204,
		err:  "too many return results",
	}

	_errMethodCrashed = StatusError{
		code: 32301,
		err:  "method running crash",
	}

	_errInvalidParams = StatusError{
		code: 32302,
		err:  "invalid params message",
	}

	_errTooManyParams = StatusError{
		code: 32303,
		err:  "too many params",
	}

	_errMissingValueParams = StatusError{
		code: 32304,
		err:  "missing value for params",
	}

	_errHandleTimeout = StatusError{
		code: 32401,
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
