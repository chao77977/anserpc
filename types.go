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
