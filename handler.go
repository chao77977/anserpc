package anserpc

import (
	"context"
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
		return makeJSONErrorMessage(err)
	}

	cb := h.sr.callback(msg.Group, msg.Service, "", msg.Method)
	if cb == nil {
		_xlog.Error("Callback method not found or not available",
			"group", msg.Group, "service", msg.Service, "method", msg.Method)
		return makeJSONErrorMessage(_errMethodNotFound)
	}

	// TODO
	//msg := make(chan *jsonMessage)
	return nil
}

func (h *handler) run() {

}
