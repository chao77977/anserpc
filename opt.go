package anserpc

const (
	_defRPCHost = "127.0.0.1"
	_defRPCPort = 56789
	_defIPCPath = "/var/run/anser.rpc"
)

type Option interface {
	apply(opts *options)
}

type options struct {
	rpc *rpcEndpoint
	ipc ipcEndpoint
	log *logOpt
}

func defaultOpt() *options {
	return &options{
		log: withDefaultLogOpt(),
	}
}

type rpcEndpoint struct {
	host string
	port int
}

func (r *rpcEndpoint) apply(opts *options) {
	opts.rpc = r
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
