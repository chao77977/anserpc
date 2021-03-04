package util

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var _interrupter = &interrupter{
	callbacks: make([]func(), 0, 1),
}

func RegisterOnInterrupt(cb func()) {
	_interrupter.register(cb)
}

type interrupter struct {
	mu        sync.Mutex
	once      sync.Once
	callbacks []func()
}

func (i *interrupter) register(cb func()) {
	if cb == nil {
		return
	}

	i.monitor()

	i.mu.Lock()
	i.callbacks = append(i.callbacks, cb)
	i.mu.Unlock()
}

func (i *interrupter) monitor() {
	i.once.Do(func() {
		go func() {
			sigC := make(chan os.Signal, 1)
			signal.Notify(sigC,
				os.Interrupt,
				syscall.SIGINT,
				syscall.SIGTERM,
			)
			<-sigC
			i.callback()
		}()
	})
}

func (i *interrupter) callback() {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, cb := range i.callbacks {
		cb()
	}
}
