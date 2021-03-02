package anserpc

/*
  The anserpc library is free software: you can redistribute it
  and/or modify it under the terms of the Apache License.
*/

import (
	"sync"
)

type Anser struct {
	opts *options
	wg   sync.WaitGroup

	sr        *serviceRegistry
	rpcServer *httpServer
	rpcMu     sync.Mutex
	rpcErr    error
}

func New(ops ...Option) *Anser {
	opts := defaultOpt()
	for _, o := range ops {
		o.apply(opts)
	}

	a := &Anser{
		opts: opts,
		sr:   newServiceRegistry(),
	}

	newSafeLogger(a.opts.log)
	return a
}

func (a *Anser) Register(group, service, version string, public bool, receiver interface{}) {
	a.RegisterAPI(&API{
		Group:    group,
		Service:  service,
		Version:  version,
		Public:   public,
		Receiver: receiver,
	})
}

func (a *Anser) RegisterWithGroup(name string) *groupRegister {
	return newGroupRegister(name, a.sr)
}

func (a *Anser) RegisterService(name, version string, public bool, receiver interface{}) {
	a.Register("", name, version, public, receiver)
}

func (a *Anser) RegisterAPI(apis ...*API) {
	for _, api := range apis {
		a.sr.registerWithAPI(api)
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

	a.rpcServer = newHttpServer(a.opts.http, a.sr)
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

	if a.sr != nil {
		_xlog.Info("Anserpc registered services\n" + a.sr.modules())
	}

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
