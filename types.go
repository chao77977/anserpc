package anserpc

import (
	"context"
	"io"
	"time"

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
	io.Reader
	WriteCloserAndDeadline
}

type CloserAndDeadline interface {
	io.Closer
	SetWriteDeadline(time.Time) error
}

type WriteCloserAndDeadline interface {
	io.Writer
	CloserAndDeadline
}

type serviceCodec interface {
	readBatch() ([]*jsonMessage, bool, error)
	writeTo(context.Context, interface{}) error
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
