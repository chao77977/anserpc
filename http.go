package anserpc

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chao77977/anserpc/util"
)

const (
	_maxReqContentLength = 1024 * 1024 * 5
)

var (
	_httpMethods = util.WithStringSet([]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
	})
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
	// permit empty request for health-checking
	if r.Method == http.MethodGet && r.ContentLength == 0 &&
		r.URL.RawQuery == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// check method
	if v.deniedMethods.Contains(r.Method) {
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

type gzipWriteHandler struct {
	rw   *gzipResponseWriter
	next http.Handler
}

type gzipResponseWriter struct {
	io.WriteCloser
	http.ResponseWriter
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.WriteCloser.Write(b)
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipWriteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		g.next.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")

	gw := gwPool.Get().(*gzip.Writer)
	defer gwPool.Put(gw)
	gw.Reset(w)

	g.rw = &gzipResponseWriter{
		WriteCloser:    gw,
		ResponseWriter: w,
	}

	defer g.rw.WriteCloser.Close()

	g.next.ServeHTTP(g.rw, r)
}

func newGzipWriteHandler(next http.Handler) http.Handler {
	return &gzipWriteHandler{
		next: next,
	}
}

var gwPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

type httpServerConn struct {
	io.Reader
	io.Writer
}

func (h *httpServerConn) Close() error { return nil }

func (h *httpServerConn) SetWriteDeadline(time.Time) error { return nil }

type httpServer struct {
	sr       *serviceRegistry
	opt      *httpOpt
	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	err      chan error
	endpoint *rpcEndpoint
	head     http.Handler
}

func newHttpServer(opt *httpOpt, sr *serviceRegistry) *httpServer {
	server := &httpServer{
		sr:  sr,
		opt: opt,
		err: make(chan error),
	}

	server.head = newValidateHandler(opt, server)
	server.head = newVirtualHostHandler(opt, server.head)
	server.head = newGzipWriteHandler(server.head)
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

func (h *httpServer) isRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.listener != nil
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
	return nil
}

func (h *httpServer) serve() {
	h.err <- h.server.Serve(h.listener)
}

func (h *httpServer) wait() {
	if err := <-h.err; err != nil {
		_xlog.Debug("HTTP server is stopped", "err", err)
	}
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
	w.Header().Set("content-type", _defAppJson)

	ctx := r.Context()
	ctx = context.WithValue(ctx, "anser-remote", r.RemoteAddr)

	conn := &httpServerConn{
		Reader: io.LimitReader(r.Body, _maxReqContentLength),
		Writer: w,
	}

	jcodec := newCodec(conn)
	defer jcodec.close()

	if err := h.serveRequest(ctx, jcodec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *httpServer) serveRequest(ctx context.Context, jCodec *jsonCodec) error {
	msgs, isBatch, err := jCodec.readBatch()
	if err != nil {
		jCodec.writeTo(ctx, makeJSONErrorMessage(err))
		return nil
	}

	msgHdl := newHandler(h.sr, ctx)
	defer msgHdl.close()

	if !isBatch {
		jCodec.writeTo(ctx, msgHdl.handleMsg(msgs[0]))
		return nil
	}

	jCodec.writeTo(ctx, msgHdl.handleMsgs(msgs))
	return nil
}
