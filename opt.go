package anserpc

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/chao77977/anserpc/util"
)

const (
	_defRPCHost = "127.0.0.1"
	_defRPCPort = 56789
	_defIPCPath = "/var/run/anser.rpc"

	_defAppJson    = "application/json"
	_defAppJsonRpc = "application/json-rpc"
	_defAppJsonReq = "application/jsonrequest"
)

type Option interface {
	apply(opts *options)
}

type options struct {
	rpc    *rpcEndpoint
	ipc    ipcEndpoint
	log    *logOpt
	http   *httpOpt
	intrpt *interruptOpt
}

func defaultOpt() *options {
	return &options{
		log:  withDefaultLogOpt(),
		http: withDefaultHTTPOpt(),
	}
}

type rpcEndpoint struct {
	host string
	port int
}

func (r *rpcEndpoint) apply(opts *options) {
	opts.rpc = r
}

func (r *rpcEndpoint) String() string {
	return fmt.Sprintf("%s:%d", r.host, r.port)
}

func WithRPCEndpoint(host string, port int) Option {
	return &rpcEndpoint{
		host: host,
		port: port,
	}
}

func WithDefaultRPCEndpoint() Option {
	return WithRPCEndpoint(_defRPCHost, _defRPCPort)
}

type ipcEndpoint string

func (i ipcEndpoint) apply(opts *options) {
	opts.ipc = i
}

func (i ipcEndpoint) String() string {
	return string(i)
}

func WithIPCEndpoint(path string) Option {
	return ipcEndpoint(path)
}

func WithDefaultIPCEndpoint() Option {
	return WithIPCEndpoint(_defIPCPath)
}

type logOpt struct {
	path      string
	filterLvl logLvl
	silent    bool
	logger    Logger
}

func (l *logOpt) apply(opts *options) {
	opts.log = l
}

func WithLogFileOpt(path string, filterLvl logLvl) Option {
	return &logOpt{
		path:      path,
		filterLvl: filterLvl,
		silent:    true,
	}
}

func withDefaultLogOpt() *logOpt {
	return &logOpt{
		filterLvl: LvlDebug,
		silent:    false,
	}
}

func WithDefaultLogOpt() Option {
	return withDefaultLogOpt()
}

func WithLoggerOpt(logger Logger) Option {
	return &logOpt{
		logger: logger,
	}
}

func withDefaultHTTPOpt() *httpOpt {
	vhosts := util.WithLowerStringSet([]string{
		"localhost",
	})

	deniedMethods := util.WithLowerStringSet([]string{
		http.MethodDelete,
		http.MethodPut,
	})

	allowedContentTypes := util.WithLowerStringSet([]string{
		_defAppJson,
		_defAppJsonRpc,
		_defAppJsonReq,
	})

	return &httpOpt{
		vhosts:              vhosts,
		deniedMethods:       deniedMethods,
		allowedContentTypes: allowedContentTypes,
	}
}

func WithHTTPVhostOpt(vhosts ...string) Option {
	opt := &httpOpt{
		vhosts: util.NewStringSet(),
	}

	for _, host := range vhosts {
		if host == "" {
			continue
		}

		opt.vhosts.Add(strings.ToLower(host))
	}

	return opt
}

func WithHTTPDeniedMethodOpt(methods ...string) Option {
	opt := &httpOpt{
		deniedMethods: util.NewStringSet(),
	}

	for _, method := range methods {
		if method == "" {
			continue
		}

		opt.deniedMethods.Add(strings.ToLower(method))
	}

	return opt
}

type interruptOpt struct {
	disableInterruptHandler bool
}

func (i *interruptOpt) apply(opts *options) {
	opts.intrpt = i
}

func WithDisableInterruptHandler() Option {
	return &interruptOpt{
		disableInterruptHandler: true,
	}
}
