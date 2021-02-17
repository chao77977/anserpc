package anserpc

import (
	"sync"
)

type Anser struct {
	opts *options
	wg   sync.WaitGroup

	rpcServer *httpServer
	rpcMu     sync.Mutex
	rpcErr    error
}

func New(ops ...Option) *Anser {
	opts := defaultOpt()
	for _, o := range ops {
		o.apply(opts)
	}

	return &Anser{
		opts: opts,
	}
}

func (a *Anser) rpcAllowed() bool {
	return a.opts.rpc != nil
}

func (a *Anser) startToWait(wp waitProc) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if wp == nil {
			return
		}

		a.rpcErr = wp.wait()
	}()
}

func (a *Anser) enableRPCServer() error {
	a.rpcMu.Lock()
	defer a.rpcMu.Unlock()

	a.rpcServer = newHttpServer()
	a.rpcErr = a.rpcServer.setListenAddr(a.opts.rpc)
	if a.rpcErr != nil {
		return a.rpcErr
	}

	a.rpcErr = a.rpcServer.start()
	if a.rpcErr != nil {
		return a.rpcErr
	}

	a.startToWait(a.rpcServer)

	return nil
}

func (a *Anser) disableRPCServer() {
	a.rpcMu.Lock()
	defer a.rpcMu.Unlock()

	if a.rpcServer == nil {
		return
	}

	a.rpcServer.stop()
}

func (a *Anser) Run() {
	newSafeLogger(a.opts.log)

	if a.rpcAllowed() {
		if err := a.enableRPCServer(); err != nil {
			a.disableRPCServer()
		}
	}

	a.wg.Wait()

	if a.rpcErr != nil {
		_xlog.Error("RPC server is stopped", "err", a.rpcErr)
	}

	_xlog.Info("AnserRPC Service is down")
}

func (a *Anser) Close() {
	a.disableRPCServer()
	a.wg.Wait()
}
