package anserpc

import (
	"context"
	"reflect"
	"runtime"
	"time"
)

const (
	_defTimeout = 3600 // 2 hours
)

type handler struct {
	sr    *serviceRegistry
	ctx   context.Context
	msgsC []chan *jsonMessage
}

func newHandler(sr *serviceRegistry, ctx context.Context) *handler {
	return &handler{
		sr:  sr,
		ctx: ctx,
	}
}

func (h *handler) close() {
	// TODO//
}

func (h *handler) handleMsgs(msgs []*jsonMessage) []*jsonMessage {
	l := len(msgs)
	h.msgsC = make([]chan *jsonMessage, 0, l)
	for _, msg := range msgs {
		msgC := make(chan *jsonMessage, 1)
		h.msgsC = append(h.msgsC, msgC)
		h.handle(msg, msgC)
	}

	retMsgs := make([]*jsonMessage, l)
	for i := 0; i < l; i++ {
		retMsgs[i] = h.wait(msgs[i], h.msgsC[i])
	}

	return retMsgs
}

func (h *handler) handleMsg(msg *jsonMessage) *jsonMessage {
	msgC := make(chan *jsonMessage, 1)
	h.msgsC = append(h.msgsC, msgC)
	h.handle(msg, msgC)
	return h.wait(msg, msgC)
}

func (h *handler) wait(msg *jsonMessage, msgC <-chan *jsonMessage) *jsonMessage {
	timer := time.NewTimer(_defTimeout * time.Second)
	defer timer.Stop()

LOOP:
	for {
		select {
		case retMsg := <-msgC:
			_xlog.Info("Method completed", "method", msg.Method)
			return retMsg
		case <-timer.C:
			break LOOP
		}
	}

	_xlog.Info("Method run timeout", "method", msg.Method)
	return msg.errResponse(_errHandleTimeout)
}

func (h *handler) handle(msg *jsonMessage, msgC chan<- *jsonMessage) {
	if err := msg.doValidate(); err != nil {
		msgC <- msg.errResponse(err)
		return
	}

	cb := h.sr.callback(msg.Group, msg.Service, "", msg.Method)
	if cb == nil {
		_xlog.Error("Callback method not found or not available",
			"group", msg.Group, "service", msg.Service, "method", msg.Method)
		msgC <- msg.errResponse(_errMethodNotFound)
		return
	}

	args, err := msg.retrieveArgs(cb.argTypes)
	if err != nil {
		msgC <- msg.errResponse(err)
		return
	}

	go func(c chan<- *jsonMessage) {
		_xlog.Info("Method start to run", "method", msg.Method)
		r, err := h.call(cb, msg.Method, args)
		if err != nil {
			c <- msg.errResponse(err)
		}

		c <- msg.response(r)
	}(msgC)
}

func (h *handler) call(cb *callback, method string, args []reflect.Value) (result interface{}, err error) {
	callArgs := make([]reflect.Value, 0, len(args)+2)

	if cb.rcvr.IsValid() {
		callArgs = append(callArgs, cb.rcvr)
	}

	if cb.hasCtx {
		callArgs = append(callArgs, reflect.ValueOf(h.ctx))
	}

	callArgs = append(callArgs, args...)

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf, false)]
			_xlog.Error("Method run crash", "method", method,
				"err", r, "stack", buf)
			err = _errMethodCrashed
		}
	}()

	r := cb.fn.Call(callArgs)
	if cb.returnType < 0 {
		return nil, nil
	} else if cb.returnType == 0 {
		if r[0].IsNil() {
			return nil, nil
		}

		return nil, r[0].Interface().(error)
	}

	if r[1].IsNil() {
		return r[0].Interface(), nil
	}

	return r[0].Interface(), r[1].Interface().(error)
}
