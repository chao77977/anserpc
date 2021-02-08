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
