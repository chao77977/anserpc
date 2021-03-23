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

func doHandle(ctx context.Context, jCodec serviceCodec, sr *serviceRegistry) {
	msgs, isBatch, err := jCodec.readBatch()
	if err != nil {
		_xlog.Debug("Read message error", "err", err)
		jCodec.writeTo(ctx, makeJSONErrorMessage(_errInvalidRequest))
		return
	}

	msgHdl := newHandler(sr, ctx)
	defer msgHdl.close()

	if !isBatch {
		jCodec.writeTo(ctx, msgHdl.handleMsg(msgs[0]))
	} else {
		jCodec.writeTo(ctx, msgHdl.handleMsgs(msgs))
	}
}

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
	if h.msgsC != nil {
		for _, msgC := range h.msgsC {
			select {
			case <-msgC:
			default:
			}
		}
	}
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
			_xlog.Info("Method completed", "message", msg)
			if retMsg.hasErr() {
				_failureReqeustCounter.Inc(1)
			} else {
				_successRequestCounter.Inc(1)
			}
			return retMsg
		case <-timer.C:
			_failureReqeustCounter.Inc(1)
			break LOOP
		}
	}

	_xlog.Debug("Method run timeout", "message", msg)
	return msg.errResponse(_errHandleTimeout)
}

func (h *handler) handle(msg *jsonMessage, msgC chan<- *jsonMessage) {
	if err := msg.doValidate(); err != nil {
		_xlog.Debug("Message validation failure", "message", msg)
		msgC <- msg.errResponse(err)
		return
	}

	cb := h.sr.callback(msg.Group, msg.Service, msg.ServiceVersion, msg.Method)
	if cb == nil {
		_xlog.Debug("Method callback not found or not available",
			"message", msg)
		msgC <- msg.errResponse(_errMethodNotFound)
		return
	}

	args, err := msg.retrieveArgs(cb.argTypes)
	if err != nil {
		_xlog.Debug("Invalid message params", "message", msg, "err", err)
		msgC <- msg.errResponse(err)
		return
	}

	go func(c chan<- *jsonMessage) {
		_xlog.Info("Method starting", "message", msg)
		r, err := h.call(cb, msg.String(), args)
		if err != nil {
			c <- msg.errResponse(err)
			return
		}

		c <- msg.response(r)
	}(msgC)
}

func (h *handler) call(cb *callback, msg string, args []reflect.Value) (result interface{}, err error) {
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
			_xlog.Debug("Method crashed", "message", msg,
				"err", r, "stack", buf)
			err = _errMethodCrashed
		}
	}()

	_requestCounter.Inc(1)

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
