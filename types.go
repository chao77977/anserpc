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
	ErrorCode() int
	ErrorMessage() string
}

type ResultDataError interface {
	ErrorData() interface{}
}

type ResultError interface {
	ResultCodeError
	ResultDataError
}
