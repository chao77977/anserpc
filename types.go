package anserpc

import (
	"io"
)

type waitProc interface {
	wait() error
	stop()
}

type Conn interface {
	io.ReadWriteCloser
}

type ResultCodeError interface {
	Error() string
	ErrorCode() int
}

type ResultMessageError interface {
	Error() string
	ErrorMessage() string
}

type ResultDataError interface {
	ResultCodeError
	ErrorData() interface{}
}

type ResultError interface {
	ResultCodeError
	ResultMessageError
	ResultDataError
}
