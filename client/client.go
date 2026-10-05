/*
Package client provides a Go client for anserpc servers.

It speaks the same extended JSON-RPC 2.0 wire format as the server
(github.com/chao77977/anserpc) over three transports: HTTP, WebSocket and
IPC (unix socket). It supports single and batch calls and decodes server
errors into a typed Error carrying code / message / data.

Basic usage:

	c, err := client.DialHTTP("http://127.0.0.1:56789")
	if err != nil {
		// handle
	}
	defer c.Close()

	var ip string
	err = c.Call(context.Background(), &ip, client.Request{
		Group:   "system",
		Service: "network",
		Version: "1.0",
		Method:  "IP",
	})
*/
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
)

const jsonRPCVersion = "2.0"

// Request describes a single RPC call. Group and Version are optional and
// mirror the server's addressing model (group / service / service version /
// method). Params are encoded as a positional JSON array.
type Request struct {
	Group   string
	Service string
	Version string
	Method  string
	Params  []interface{}
}

// transport is the pluggable mechanism that performs a round-trip of one or
// more wire messages. HTTP opens a request per round-trip; WebSocket and IPC
// hold a persistent connection.
type transport interface {
	roundTrip(ctx context.Context, reqs []*wireRequest) ([]*wireResponse, error)
	close() error
}

// Client is a transport-agnostic anserpc client. It is safe for concurrent
// use as long as the underlying transport is (HTTP is; the streaming
// transports serialize round-trips with a mutex).
type Client struct {
	t   transport
	seq uint64
}

func newClient(t transport) *Client {
	return &Client{t: t}
}

// Close releases the underlying transport resources.
func (c *Client) Close() error {
	return c.t.close()
}

func (c *Client) nextID() uint64 {
	return atomic.AddUint64(&c.seq, 1)
}

// Call performs a single RPC. If result is non-nil, the server's result is
// JSON-decoded into it. A server-side error is returned as *Error.
func (c *Client) Call(ctx context.Context, result interface{}, req Request) error {
	wreq, err := c.makeRequest(req)
	if err != nil {
		return err
	}

	resps, err := c.t.roundTrip(ctx, []*wireRequest{wreq})
	if err != nil {
		return err
	}
	if len(resps) != 1 {
		return fmt.Errorf("anserpc client: expected 1 response, got %d", len(resps))
	}

	return decodeResult(resps[0], result)
}

// BatchElem is one element of a batch call: the request plus a destination
// for its result and a slot for its error.
type BatchElem struct {
	Request Request
	// Result, if non-nil, receives the JSON-decoded result for this element.
	Result interface{}
	// Error is set to the per-element error (often *Error) after the batch
	// returns. It is independent of the error returned by BatchCall itself.
	Error error
}

// BatchCall sends all requests as a single JSON-RPC batch and fills each
// element's Result/Error. The returned error is non-nil only for
// transport-level failures; per-call errors live in elems[i].Error.
func (c *Client) BatchCall(ctx context.Context, elems []BatchElem) error {
	if len(elems) == 0 {
		return nil
	}

	reqs := make([]*wireRequest, len(elems))
	byID := make(map[string]int, len(elems))
	for i := range elems {
		wreq, err := c.makeRequest(elems[i].Request)
		if err != nil {
			return err
		}
		reqs[i] = wreq
		byID[string(wreq.ID)] = i
	}

	resps, err := c.t.roundTrip(ctx, reqs)
	if err != nil {
		return err
	}

	// Match responses to requests by id; servers may reorder a batch.
	for _, resp := range resps {
		idx, ok := byID[string(resp.ID)]
		if !ok {
			continue
		}
		elems[idx].Error = decodeResult(resp, elems[idx].Result)
	}
	return nil
}

func (c *Client) makeRequest(req Request) (*wireRequest, error) {
	if req.Service == "" || req.Method == "" {
		return nil, fmt.Errorf("anserpc client: service and method are required")
	}

	var params json.RawMessage
	if len(req.Params) > 0 {
		b, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("anserpc client: encode params: %w", err)
		}
		params = b
	}

	id, _ := json.Marshal(c.nextID())
	return &wireRequest{
		Version:        jsonRPCVersion,
		Group:          req.Group,
		Service:        req.Service,
		ServiceVersion: req.Version,
		Method:         req.Method,
		Params:         params,
		ID:             id,
	}, nil
}

// wireRequest / wireResponse mirror the server's jsonMessage layout so the
// encoding is identical on both ends.
type wireRequest struct {
	Version        string          `json:"jsonrpc,omitempty"`
	Group          string          `json:"group,omitempty"`
	Service        string          `json:"service,omitempty"`
	ServiceVersion string          `json:"service_version,omitempty"`
	Method         string          `json:"method,omitempty"`
	Params         json.RawMessage `json:"params,omitempty"`
	ID             json.RawMessage `json:"id,omitempty"`
}

type wireResponse struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

func decodeResult(resp *wireResponse, result interface{}) error {
	if resp.Error != nil {
		return resp.Error
	}
	if result == nil || len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("anserpc client: decode result: %w", err)
	}
	return nil
}
