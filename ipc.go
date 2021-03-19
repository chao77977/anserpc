package anserpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/chao77977/anserpc/util"
)

const (
	_maxPathLength = 128
)

type ipcServerConn struct {
	io.Reader
	WriteCloserAndDeadline
}

type ipcServer struct {
	sr       *serviceRegistry
	mu       sync.Mutex
	listener net.Listener
	endpoint ipcEndpoint
	err      chan error
}

func newIPCServer(sr *serviceRegistry) *ipcServer {
	return &ipcServer{
		sr:  sr,
		err: make(chan error),
	}
}

func (i *ipcServer) isRunning() bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.listener != nil
}

func (i *ipcServer) setPath(endpoint ipcEndpoint) error {
	if len(endpoint) > _maxPathLength {
		return fmt.Errorf("IPC endpoint is longer that %d characters",
			_maxPathLength)
	}

	if err := util.MakeFilePath(string(endpoint)); err != nil {
		return err
	}

	os.Remove(string(endpoint))
	i.endpoint = endpoint

	return nil
}

func (i *ipcServer) start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.endpoint == "" || i.listener != nil {
		// already running or not configured
		return nil
	}

	listener, err := net.Listen("unix", string(i.endpoint))
	if err != nil {
		return err
	}

	os.Chmod(string(i.endpoint), 0600)
	i.listener = listener

	go i.serve()
	return nil
}

func (i *ipcServer) serve() {
	for {
		conn, err := i.listener.Accept()
		if err != nil {
			if util.IsTemporaryError(err) {
				continue
			}

			i.err <- err
			return
		}

		i.serveIPC(conn)
	}
}

func (i *ipcServer) wait() {
	if err := <-i.err; err != nil {
		_xlog.Debug("IPC server is stopped", "err", err)
	}
}

func (i *ipcServer) stop() {
	i.mu.Lock()
	defer i.mu.Lock()
	i.doStop()
}

func (i *ipcServer) doStop() {
	if i.listener == nil {
		return
	}

	i.listener.Close()
	i.endpoint = (ipcEndpoint)("")
	i.listener = nil
}

func (i *ipcServer) serveIPC(conn net.Conn) {
	ctx := context.WithValue(context.Background(),
		"anser-local", conn.LocalAddr())

	localConn := &ipcServerConn{
		Reader:                 io.LimitReader(conn, _maxReqContentLength),
		WriteCloserAndDeadline: conn,
	}

	jcodec := newCodec(localConn)
	defer jcodec.close()

	doHandle(ctx, jcodec, i.sr)
}
