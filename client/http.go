package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// httpTransport performs one HTTP request per round-trip. It is safe for
// concurrent use via the standard http.Client.
type httpTransport struct {
	url    string
	client *http.Client
}

// Option configures a Client at dial time.
type Option func(*options)

type options struct {
	httpClient *http.Client
}

// WithHTTPClient sets a custom *http.Client for the HTTP transport.
func WithHTTPClient(hc *http.Client) Option {
	return func(o *options) { o.httpClient = hc }
}

func applyOptions(opts []Option) *options {
	o := &options{httpClient: http.DefaultClient}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// DialHTTP returns a Client that talks to an anserpc HTTP endpoint, e.g.
// "http://127.0.0.1:56789".
func DialHTTP(url string, opts ...Option) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("anserpc client: empty URL")
	}
	o := applyOptions(opts)
	return newClient(&httpTransport{url: url, client: o.httpClient}), nil
}

func (t *httpTransport) roundTrip(ctx context.Context, reqs []*wireRequest) ([]*wireResponse, error) {
	body, isBatch, err := encodeRequests(reqs)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anserpc client: http status %d: %s", resp.StatusCode, string(data))
	}

	return decodeResponses(data, isBatch, len(reqs))
}

func (t *httpTransport) close() error { return nil }

// encodeRequests marshals a slice of requests as a single object (len==1) or a
// JSON array (batch), matching the server's batch-detection logic.
func encodeRequests(reqs []*wireRequest) (body []byte, isBatch bool, err error) {
	if len(reqs) == 1 {
		b, err := json.Marshal(reqs[0])
		return b, false, err
	}
	b, err := json.Marshal(reqs)
	return b, true, err
}

// decodeResponses parses the server reply. A batch reply is a JSON array; a
// single reply is a lone object.
func decodeResponses(data []byte, isBatch bool, n int) ([]*wireResponse, error) {
	if isBatch {
		var resps []*wireResponse
		if err := json.Unmarshal(data, &resps); err != nil {
			return nil, fmt.Errorf("anserpc client: decode batch response: %w", err)
		}
		return resps, nil
	}

	var resp wireResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("anserpc client: decode response: %w", err)
	}
	return []*wireResponse{&resp}, nil
}
