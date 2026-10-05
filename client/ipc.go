package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// streamTransport is a persistent, connection-oriented transport (IPC and the
// WebSocket transport share this round-trip discipline): round-trips are
// serialized with a mutex because a single connection carries all calls.
type ipcTransport struct {
	mu   sync.Mutex
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

// DialIPC connects to an anserpc IPC endpoint (a unix domain socket path).
func DialIPC(path string, opts ...Option) (*Client, error) {
	if path == "" {
		return nil, fmt.Errorf("anserpc client: empty IPC path")
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("anserpc client: dial ipc: %w", err)
	}
	dec := json.NewDecoder(conn)
	dec.UseNumber()
	return newClient(&ipcTransport{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  dec,
	}), nil
}

func (t *ipcTransport) roundTrip(ctx context.Context, reqs []*wireRequest) ([]*wireResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		t.conn.SetDeadline(deadline)
		defer t.conn.SetDeadline(time.Time{})
	}

	// Honor an already-cancelled context before doing I/O.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	isBatch := len(reqs) > 1
	if isBatch {
		if err := t.enc.Encode(reqs); err != nil {
			return nil, fmt.Errorf("anserpc client: write ipc batch: %w", err)
		}
	} else {
		if err := t.enc.Encode(reqs[0]); err != nil {
			return nil, fmt.Errorf("anserpc client: write ipc: %w", err)
		}
	}

	return t.readResponses(isBatch)
}

// readResponses reads one JSON value from the stream and parses it as either a
// single response or a batch array.
func (t *ipcTransport) readResponses(isBatch bool) ([]*wireResponse, error) {
	var raw json.RawMessage
	if err := t.dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("anserpc client: read ipc: %w", err)
	}
	return decodeResponses(raw, isBatch, 0)
}

func (t *ipcTransport) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.Close()
}
