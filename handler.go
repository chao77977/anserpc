package anserpc

import (
	"context"
	"reflect"
	"runtime"
)

type handler struct {
	sr   *serviceRegistry
	msgs []chan *jsonMessage
	ctx  context.Context
}

func newHandler(sr *serviceRegistry, ctx context.Context) *handler {
	return &handler{
		sr:  sr,
		ctx: ctx,
	}
}

func (h *handler) close() {}

func (h *handler) handleMsg(msg *jsonMessage) *jsonMessage {
	if err := msg.doValidate(); err != nil {
		return msg.errResponse(err)
	}

	cb := h.sr.callback(msg.Group, msg.Service, "", msg.Method)
	if cb == nil {
		_xlog.Error("Callback method not found or not available",
			"group", msg.Group, "service", msg.Service, "method", msg.Method)
		return msg.errResponse(_errMethodNotFound)
	}

	args, err := msg.retrieveArgs(cb.argTypes)
	if err != nil {
		return msg.errResponse(err)
	}

	msgC := make(chan *jsonMessage)
	go func(c chan<- *jsonMessage) {
		r, err := h.call(cb, msg.Method, args)
		if err != nil {
			c <- msg.errResponse(err)
		}

		c <- msg.response(r)
	}(msgC)

	// TODO: timeout
	return <-msgC
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
			_xlog.Error("method run crash", "method", method,
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
