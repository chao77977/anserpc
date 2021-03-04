# anserpc - A Light JSON2.0 RPC Lib


## Quick Sample
```
app := anserpc.New(
    anserpc.WithRPCEndpoint("0.0.0.0", 56789),
    anserpc.WithDisableInterruptHandler(),
)

app.Register("system", "network", "1.0", true, &network{})
app.Register("system", "storage", "1.0", false, &storage{})
app.Run()
```


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
