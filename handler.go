package anserpc

import (
	"context"
)

type handler struct {
	sr  *serviceRegistry
	msg chan *jsonMessage
	ctx context.Context
}

func newHandler(sr *serviceRegistry, ctx context.Context) *handler {
	return &handler{
		sr:  sr,
		msg: make(chan *jsonMessage),
		ctx: ctx,
	}
}

func (h *handler) close() {}

func (h *handler) handleMsg(msg *jsonMessage) *jsonMessage {
	// TODO
	return nil
}
