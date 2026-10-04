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

// ctxKey is an unexported type for context keys defined in this package,
// avoiding collisions with keys defined elsewhere (staticcheck SA1029).
type ctxKey string

const (
	// ctxRemoteAddr carries the client/peer address of the active connection.
	ctxRemoteAddr ctxKey = "anser-remote-addr"
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
	close()
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
