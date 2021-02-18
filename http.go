package anserpc

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/chao77977/anserpc/util"
)

const (
	_maxReqContentLength = 1024 * 1024 * 5
)

type httpOpt struct {
	vhosts              util.StringSet
	deniedMethods       util.StringSet
	allowedContentTypes util.StringSet
}

func (h *httpOpt) apply(opts *options) {
	opts.http.vhosts.Merge(h.vhosts)
	opts.http.deniedMethods.Merge(h.deniedMethods)
	opts.http.allowedContentTypes.Merge(h.allowedContentTypes)
}

type validateHandler struct {
	deniedMethods       util.StringSet
	allowedContentTypes util.StringSet
	next                http.Handler
}

func (v *validateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check method
	if v.deniedMethods.Contains(strings.ToLower(r.Method)) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check content length
	if r.ContentLength > _maxReqContentLength {
		http.Error(w, "content lentgth too large",
			http.StatusRequestEntityTooLarge)
		return
	}

	// allow OPTIONS
	if r.Method == http.MethodOptions {
		v.next.ServeHTTP(w, r)
		return
	}

	// check content type
	if m, _, err := mime.ParseMediaType(r.Header.Get("content-type")); err == nil {
		if v.allowedContentTypes.Contains(m) {
			v.next.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, "invalid content type", http.StatusUnsupportedMediaType)
}

func newValidateHandler(opt *httpOpt, next http.Handler) http.Handler {
	return &validateHandler{
		deniedMethods:       opt.deniedMethods,
		allowedContentTypes: opt.allowedContentTypes,
		next:                next,
	}
}

type virtualHostHandler struct {
	vhosts util.StringSet
	next   http.Handler
}

func (v *virtualHostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host == "" {
		v.next.ServeHTTP(w, r)
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	if ip := net.ParseIP(host); ip != nil {
		v.next.ServeHTTP(w, r)
		return
	}

	// validate the host
	if v.vhosts.Contains("*") || v.vhosts.Contains(host) {
		v.next.ServeHTTP(w, r)
		return
	}

	http.Error(w, "host access denied", http.StatusForbidden)
}

func newVirtualHostHandler(opt *httpOpt, next http.Handler) http.Handler {
	return &virtualHostHandler{
		vhosts: opt.vhosts,
		next:   next,
	}
}

type httpServer struct {
	opt      *httpOpt
	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	err      chan error
	endpoint *rpcEndpoint
	head     http.Handler
}

func newHttpServer(opt *httpOpt) *httpServer {
	server := &httpServer{
		opt: opt,
		err: make(chan error),
	}

	server.head = newValidateHandler(opt, server)
	server.head = newVirtualHostHandler(opt, server.head)
	return server
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
		return err
	}

	h.listener = listener
	h.server = &http.Server{Handler: h.head}

	go h.serve()
	_xlog.Info("HTTP Server is running on " + h.endpoint.String())

	return nil
}

func (h *httpServer) serve() {
	h.err <- h.server.Serve(h.listener)
}

func (h *httpServer) wait() error {
	return <-h.err
}

func (h *httpServer) stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.doStop()
}

func (h *httpServer) doStop() {
	if h.listener == nil {
		return
	}

	h.server.Shutdown(context.Background())
	h.listener.Close()

	h.endpoint = (*rpcEndpoint)(nil)
	h.server, h.listener = nil, nil
}

func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//TODO:
	fmt.Println("running...")
}
