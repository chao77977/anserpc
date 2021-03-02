package anserpc

import (
	"fmt"
)

const (
	_defErrCode = -32000
)

var (
	// the JSON sent is not a valid Request object
	_errInvalidRequest = StatusError{
		code: -32600,
		err:  "invalid request",
	}

	// the method does not exist / is not available
	_errMethodNotFound = StatusError{
		code: -32601,
		err:  "method not found",
	}

	// tnvalid method parameter(s)
	_errInvalidParams = StatusError{
		code: -32602,
		err:  "invalid params",
	}

	// internal JSON-RPC error
	_errInternal = StatusError{
		code: -32603,
		err:  "internal error",
	}

	// invalid JSON was received by the server
	_errJSONContent = StatusError{
		code: -32700,
		err:  "parse error",
	}

	_errProtoVersion = StatusError{
		code: -32001,
		err:  "invalid version",
	}

	_errProtoServiceOrMethodNotFound = StatusError{
		code: -32002,
		err:  "service or method not found",
	}

	_errServiceOrVersion = StatusError{
		code: -32003,
		err:  "service name or version not found",
	}

	_errResultErrorNotFound = StatusError{
		code: -32004,
		err:  "error of return result not found",
	}

	_errNumOfResult = StatusError{
		code: -32005,
		err:  "too many return results",
	}

	_errMethodCrashed = StatusError{
		code: -32006,
		err:  "method running crash",
	}

	_errTooManyParams = StatusError{
		code: -32007,
		err:  "too many params",
	}

	_errMissingValueParams = StatusError{
		code: -32008,
		err:  "missing value for params",
	}

	_errHandleTimeout = StatusError{
		code: -32009,
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
