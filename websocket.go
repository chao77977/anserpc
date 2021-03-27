package anserpc

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	_pingInterval     = 60 * time.Second
	_pingWriteTimeout = 5 * time.Second
)

type websocketHandler struct {
	allowed  bool
	server   *httpServer
	next     http.Handler
	upgrader websocket.Upgrader
	readErr  chan error
	readMsg  chan readMessage
	closeC   chan struct{}
}

func (ws *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !ws.allowed || !websocket.IsWebSocketUpgrade(r) {
		ws.next.ServeHTTP(w, r)
		return
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		_xlog.Debug("WebSocket upgrade failure", "err", err)
		http.Error(w, "WebSocket upgrade failure",
			http.StatusInternalServerError)
		return
	}

	ctx := context.WithValue(context.Background(),
		"anser-websocket-remote", conn.RemoteAddr())

	jwc := newWebSocketCodec(conn)
	defer jwc.close()

	ws.server.codecs.add(jwc)
	defer ws.server.codecs.remove(jwc)

	ws.doHandle(ctx, jwc)
}

func (ws *websocketHandler) doHandle(ctx context.Context, jCodec serviceCodec) {
	defer func() {
		for {
			select {
			case <-ws.readErr:
			case <-ws.readMsg:
				return
			}
		}
	}()

	go ws.read(ctx, jCodec)

	for {
		select {
		case <-ws.closeC:
			return

		case err := <-ws.readErr:
			_xlog.Debug("Read message error", "err", err)
			return

		case r := <-ws.readMsg:
			msgHdl := newHandler(ws.server.sr, ctx)
			defer msgHdl.close()

			if !r.isBatch {
				jCodec.writeTo(ctx, msgHdl.handleMsg(r.msgs[0]))
			} else {
				jCodec.writeTo(ctx, msgHdl.handleMsgs(r.msgs))
			}
		}
	}
}

func (ws *websocketHandler) read(ctx context.Context, jCodec serviceCodec) {
	defer func() {
		if r := recover(); r != nil {
			_xlog.Debug("Reading on failed websocket connection")
			close(ws.closeC)
		}
	}()

	for {
		msgs, isBatch, err := jCodec.readBatch()
		if err != nil {
			if _, ok := err.(*json.SyntaxError); ok {
				jCodec.writeTo(ctx, makeJSONErrorMessage(_errInvalidRequest))
			}

			ws.readErr <- err
			return
		}

		ws.readMsg <- readMessage{msgs, isBatch}
	}
}

func newWebsocketHandler(opt *httpOpt, server *httpServer, next http.Handler) http.Handler {
	return &websocketHandler{
		allowed: opt.WebsocketAllowed,
		server:  server,
		next:    next,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		readErr: make(chan error),
		readMsg: make(chan readMessage),
		closeC:  make(chan struct{}),
	}
}

type webSocketCodec struct {
	*jsonCodec
	conn   *websocket.Conn
	wg     sync.WaitGroup
	resetC chan struct{}
}

func (w *webSocketCodec) writeTo(ctx context.Context, x interface{}) error {
	err := w.jsonCodec.writeTo(ctx, x)
	if err == nil {
		select {
		case w.resetC <- struct{}{}:
		default:
		}
	}

	return err
}

func (w *webSocketCodec) ping() {
	timer := time.NewTimer(_pingInterval)
	defer timer.Stop()
	defer w.wg.Done()

	for {
		select {
		case <-w.closeC:
			return
		case <-w.resetC:
			if !timer.Stop() {
				<-timer.C
			}

			timer.Reset(_pingInterval)

		case <-timer.C:
			// send ping ...
			w.jsonCodec.mu.Lock()
			w.conn.SetWriteDeadline(time.Now().Add(_pingWriteTimeout))
			w.conn.WriteMessage(websocket.PingMessage, nil)
			w.jsonCodec.mu.Unlock()
			timer.Reset(_pingInterval)
		}
	}
}

func (w *webSocketCodec) close() {
	w.jsonCodec.close()
	w.wg.Wait()

	select {
	case <-w.resetC:
	default:
	}
}

func newWebSocketCodec(conn *websocket.Conn) *webSocketCodec {
	conn.SetReadLimit(_maxReqContentLength)
	wsc := &webSocketCodec{
		jsonCodec: &jsonCodec{
			closeC: make(chan struct{}),
			encode: conn.WriteJSON,
			decode: conn.ReadJSON,
			conn:   conn,
		},
		conn:   conn,
		resetC: make(chan struct{}, 1),
	}

	go wsc.ping()
	wsc.wg.Add(1)

	return wsc
}

type readMessage struct {
	msgs    []*jsonMessage
	isBatch bool
}
