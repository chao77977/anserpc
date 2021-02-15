package anserpc

type Anser struct {
	opts      *options
	rpcServer *httpServer
}

func New(ops ...Option) *Anser {
	opts := defaultOpt()
	for _, o := range ops {
		o.apply(opts)
	}

	return &Anser{
		opts: opts,
	}
}
