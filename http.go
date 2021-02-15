package anserpc

import (
	"fmt"
	"net"
	"net/http"
	"sync"
)

type httpServer struct {
	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	err      chan error
	endpoint *rpcEndpoint
}

func newHttpServer() *httpServer {
	return &httpServer{
		err: make(chan error),
	}
}

func (h *httpServer) setListenAddr(endpoint *rpcEndpoint) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil && (endpoint.port != h.endpoint.port ||
		endpoint.host != h.endpoint.host) {
		return fmt.Errorf("HTTP server is already running on %s", h.endpoint)
	}

	h.endpoint = endpoint
	return nil
}

func (h *httpServer) listenAddr() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.listener != nil {
		return h.listener.Addr().String()
	}

	return h.endpoint.String()
}

func (h *httpServer) start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.endpoint == nil || h.listener != nil {
		// already running or not configured
		return nil
	}

	listener, err := net.Listen("tcp", h.endpoint.String())
	if err != nil {
		// TODO: cleanup
		return err
	}

	h.listener = listener
	h.server = &http.Server{Handler: h}

	go h.serve()
	_xlog.Info("HTTP Server is running", "on", h.endpoint.String())

	return nil
}

func (h *httpServer) serve() {
	h.err <- h.server.Serve(h.listener)
}

func (h *httpServer) wait() error {
	return <-h.err
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//TODO:
	fmt.Println("running...")
}
