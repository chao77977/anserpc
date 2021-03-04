# anserpc - A Light JSON2.0 RPC Lib
Anser cygnoides (Chinese name 鸿雁), that is Swan Goose. Anserpc is matching
JSON-RPC 2.0 specification, providing a lib to develop server application easily.

## Install
```
$ go get github.com/chao77977/anserpc

```

## Quick Sample: RPC on HTTP
```
app := anserpc.New(
    anserpc.WithRPCEndpoint("0.0.0.0", 56789),
)

// register service
app.Register("system", "network", "1.0", true, &network{})
app.Register("system", "storage", "1.0", false, &storage{})

app.Run()
```
### New an application
Some options as the following,
* anserpc.WithRPCEndpoint(host string, port int)
* anserpc.WithIPCEndpoint(path string)
* anserpc.WithLogFileOpt(path string, filterLvl logLvl)
* anserpc.WithHTTPVhostOpt(vhosts ...string)
* anserpc.WithHTTPDeniedMethodOpt(methods ...string)
* anserpc.WithDisableInterruptHandler()

### Register services

```
type network struct{}

func (n *network) Ping() error {
	return nil
}

func (n *network) IP() (string, error) {
	return "10.0.0.2", nil
}

func (n *network) Restart() {}

type storage struct{}

func (s *storage) Add() error { return nil }
```

```
INFO[03-04|21:02:15] Application register service(s):
INFO[03-04|21:02:15] built-in_1.0(public) -> Hello
INFO[03-04|21:02:15] system: network_1.0(public) -> Restart
INFO[03-04|21:02:15] system: network_1.0(public) -> IP
INFO[03-04|21:02:15] system: network_1.0(public) -> Ping
INFO[03-04|21:02:15] system: storage_1.0 -> Add
INFO[03-04|21:02:15] Application: running using 1 server(s)
INFO[03-04|21:02:15] HTTP: addr is [::]:56789
INFO[03-04|21:02:15] HTTP: virtual host is localhost
INFO[03-04|21:02:15] HTTP: denied method(s): DELETE/PUT
INFO[03-04|21:02:15] Server(s) shutdown on interrupt(CTRL+C)
INFO[03-04|21:02:15] Application started
```
