package anserpc

import (
	"io"

	"github.com/chao77977/anserpc/util"
)

var (
	Fmt = util.Fmt
)

type serverStatus int

type waitProc interface {
	wait()
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
