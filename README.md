# anserpc - A Light JSON2.0 RPC Lib
Anser cygnoides (Chinese name 鸿雁), that is swan goose. Anserpc is matching
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

// application starts
app.Run()

// services and their methods
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
### New an application
Supported options as the following,
* anserpc.WithRPCEndpoint(host string, port int)
* anserpc.WithIPCEndpoint(path string)
* anserpc.WithLogFileOpt(path string, filterLvl logLvl)
* anserpc.WithHTTPVhostOpt(vhosts ...string)
* anserpc.WithHTTPDeniedMethodOpt(methods ...string)
* anserpc.WithDisableInterruptHandler()

### Register services
Compared to standard RPC2.0 defination, we are introducing "group", "service", "service version" and "service is public" to register services. The same service name can be in different group. A service can have different versions.
group: "system"
service: "network" and "storage"
service version: "1.0"
service is public: true

If you really don't like "group" and "service version", you can keep them as null string.
```
app.Register("", "network", "", true, &network{})
app.Register("", "storage", "", false, &storage{})
```
But if you like, you can also use specified function to register service with group.
```
grp := s.RegisterWithGroup("system")
grp.Register("network", "1.0", true, &network{})
grp.Register("storage", "1.0", true, &storage{})
```
The following is methods from serive "network" and "storage".

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
