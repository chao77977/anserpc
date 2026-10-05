package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsTransport holds a persistent WebSocket connection. Round-trips are
// serialized with a mutex since one connection carries all calls.
type wsTransport struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

// DialWebSocket connects to an anserpc WebSocket endpoint, e.g.
// "ws://127.0.0.1:56789".
func DialWebSocket(url string, opts ...Option) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("anserpc client: empty WebSocket URL")
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("anserpc client: dial websocket: %w", err)
	}
	return newClient(&wsTransport{conn: conn}), nil
}

func (t *wsTransport) roundTrip(ctx context.Context, reqs []*wireRequest) ([]*wireResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		t.conn.SetWriteDeadline(deadline)
		t.conn.SetReadDeadline(deadline)
		defer func() {
			t.conn.SetWriteDeadline(time.Time{})
			t.conn.SetReadDeadline(time.Time{})
		}()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	isBatch := len(reqs) > 1
	var payload interface{} = reqs[0]
	if isBatch {
		payload = reqs
	}
	if err := t.conn.WriteJSON(payload); err != nil {
		return nil, fmt.Errorf("anserpc client: write websocket: %w", err)
	}

	_, data, err := t.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("anserpc client: read websocket: %w", err)
	}
	return decodeResponses(data, isBatch, len(reqs))
}

func (t *wsTransport) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Best-effort graceful close.
	_ = t.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	return t.conn.Close()
}
