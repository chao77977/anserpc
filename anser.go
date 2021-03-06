package anserpc

/*
  The anserpc library is free software: you can redistribute it
  and/or modify it under the terms of the Apache License.
*/

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/chao77977/anserpc/util"
)

const (
	_statRunning serverStatus = iota + 1
	_statStopped
)

type Anser struct {
	opts     *options
	wg       sync.WaitGroup
	mu       sync.Mutex
	nRunning uint64

	sr   *serviceRegistry
	rs   *httpServer
	rsMu sync.Mutex
	is   *ipcServer
	isMu sync.Mutex
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

func (a *Anser) ipcAllowed() bool {
	return a.opts.ipc != ""
}

func (a *Anser) interruptHandle() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.opts.intrpt == nil || a.opts.intrpt != nil &&
		!a.opts.intrpt.disableInterruptHandler {
		util.RegisterOnInterrupt(a.Close)
	}
}

func (a *Anser) startToWait(wp waitProc) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if wp == nil {
			return
		}

		wp.wait()
	}()
}

func (a *Anser) statusIPCServer() serverStatus {
	a.isMu.Lock()
	defer a.isMu.Unlock()

	if a.is != nil && a.is.isRunning() {
		return _statRunning
	}

	return _statStopped
}

func (a *Anser) enableIPCServer() error {
	a.isMu.Lock()
	defer a.isMu.Unlock()

	a.is = newIPCServer(a.sr)
	if err := a.is.setPath(a.opts.ipc); err != nil {
		return err
	}

	if err := a.is.start(); err != nil {
		return err
	}

	a.startToWait(a.is)
	atomic.AddUint64(&a.nRunning, 1)

	return nil
}

func (a *Anser) disableIPCServer() {
	a.isMu.Lock()
	defer a.isMu.Unlock()

	if a.is == nil {
		return
	}

	a.is.stop()
}

func (a *Anser) statusRPCServer() serverStatus {
	a.rsMu.Lock()
	defer a.rsMu.Unlock()

	if a.rs != nil && a.rs.isRunning() {
		return _statRunning
	}

	return _statStopped
}

func (a *Anser) enableRPCServer() error {
	a.rsMu.Lock()
	defer a.rsMu.Unlock()

	a.rs = newHttpServer(a.opts.http, a.sr)
	if err := a.rs.setListenAddr(a.opts.rpc); err != nil {
		return err
	}

	if err := a.rs.start(); err != nil {
		return err
	}

	a.startToWait(a.rs)
	atomic.AddUint64(&a.nRunning, 1)

	return nil
}

func (a *Anser) disableRPCServer() {
	a.rsMu.Lock()
	defer a.rsMu.Unlock()

	if a.rs == nil {
		return
	}

	a.rs.stop()
}

func (a *Anser) Run() {
	a.interruptHandle()
	if a.rpcAllowed() && a.statusRPCServer() != _statRunning {
		if err := a.enableRPCServer(); err != nil {
			_xlog.Debug("Failed to enable RPC server", "err", err)
			a.disableRPCServer()
		}
	}

	if a.ipcAllowed() && a.statusIPCServer() != _statRunning {
		if err := a.enableIPCServer(); err != nil {
			_xlog.Debug("Failed to enable IPC server", "err", err)
			a.disableIPCServer()
		}
	}

	a.status()
	a.wg.Wait()
	_xlog.Info("Application is down")
}

func (a *Anser) status() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sr != nil && a.sr.modules() != nil {
		_xlog.Info("Application register service(s):")
		for _, module := range a.sr.modules() {
			_xlog.Info(module)
		}
	}

	_xlog.Info(Fmt("Application: running using %d server(s)",
		atomic.LoadUint64(&a.nRunning)))

	if a.statusRPCServer() == _statRunning {
		_xlog.Info("HTTP: addr is " + a.rs.listenAddr())
	}

	if a.opts.http != nil {
		for _, host := range a.opts.http.vhosts.List() {
			_xlog.Info("HTTP: virtual host is " + host)
		}

		methods := a.opts.http.deniedMethods.List()
		if len(methods) != 0 {
			_xlog.Info("HTTP: denied method(s): " +
				strings.ToUpper(strings.Join(methods, "/")))
		}
	}

	if a.statusIPCServer() == _statRunning {
		_xlog.Info("IPC: path is " + string(a.is.endpoint))
	}

	if a.opts.intrpt == nil || !a.opts.intrpt.disableInterruptHandler {
		_xlog.Info("Server(s) shutdown on interrupt(CTRL+C)")
	}

	_xlog.Info("Application started")
}

func (a *Anser) Close() {
	a.disableRPCServer()
	a.disableIPCServer()
	a.wg.Wait()
}
