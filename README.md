# anserpc - A Lightweight JSON2.0 RPC Lib
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
// service network
type network struct{}

func (n *network) Ping() error {
	return errors.New("unknown host")
}

func (n *network) IP() (string, error) {
	return "10.0.0.2", nil
}

func (n *network) Restart() {}

// service storage
type storage struct{}

func (s *storage) Add() error { return &myErr{} }

type myErr struct{}

func (e *myErr) Error() string          { return e.ErrorMessage() }
func (e *myErr) ErrorCode() int         { return -1 }
func (e *myErr) ErrorMessage() string   { return "error message" }
func (e *myErr) ErrorData() interface{} { return struct{}{} }
```
### New Application
Supported options as the following,
* anserpc.WithRPCEndpoint(host string, port int)
* anserpc.WithIPCEndpoint(path string)
* anserpc.WithLogFileOpt(path string, filterLvl logLvl)
* anserpc.WithHTTPVhostOpt(vhosts ...string)
* anserpc.WithHTTPDeniedMethodOpt(methods ...string)
* anserpc.WithDisableInterruptHandler()

### Register Services
Compared to standard RPC2.0 defination, we are introducing "group", "service", "service version" and "service is public" to register services. The same service name can be in different group. A service can have different versions.
* group: "system"
* service: "network" and "storage"
* service version: "1.0"
* service is public: true

If you really don't like "group" and "service version", you can keep them as null string.
```
app.Register("", "network", "", true, &network{})
app.Register("", "storage", "", false, &storage{})
```
But if you like, you can also use specified function to register service with group.
```
grp := app.RegisterWithGroup("system")
grp.Register("network", "1.0", true, &network{})
grp.Register("storage", "1.0", true, &storage{})
```
The following is methods from service "network" and "storage".
Service's method
* "network"'s methods
  * Ping
  * IP
  * Restart
* "storage" 's methods
  * Add

The return value of method has three types.
* no return value
* only one return value, must be 'error'
* two return values, the first must be result and the second must be 'error'

If you want to return error code, message and data, you can implement the following interface.
```
type ResultError interface {
	Error() string
	ErrorCode() int
	ErrorMessage() string
	ErrorData() interface{}
}
```

### Start Application
The following is output when appliaction starts.
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
INFO[03-04|21:02:15] Websocket: enabled
INFO[03-04|21:02:15] Server(s) shutdown on interrupt(CTRL+C)
INFO[03-04|21:02:15] Application started
```

#### Build-in Services
Method: Hello
```
curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"service": "built-in", "method": "Hello"}' http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"result":"olleh"}
```

Method: Metrics
```
 curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"service": "built-in", "method": "Metrics"}' http://127.0.0.1:56789

 {"jsonrpc":"2.0","id":10001,"result":"{\"anser/failure\":{\"count\":1},\"anser/requests\":{\"count\":2},\"anser/success\":{\"count\":1}}"}
```


#### Registered Services
```
curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"group": "system", "service": "network", "method": "Ping"}' http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"error":{"code":-32000,"message":"unknown host"}}
```

```
curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"group": "system", "service": "network", "method": "IP"}'  http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"result":"10.0.0.2"}
```

```
curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"group": "system", "service": "storage", "method": "Add"}'  http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"error":{"code":-1,"message":"error message","data":{}}}
```

```
curl -H "Content-Type: application/json" -X GET --data '{"jsonrpc": "2.0", "id":10001,"group": "system", "service": "storage", "method": "NotFound"}'  http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"error":{"code":-32601,"message":"method not found"}}
```

## Quick Sample: IPC
Anserpc can run both RPC on HTTP and IPC servers.
```
app := anserpc.New(
    anserpc.WithRPCEndpoint("0.0.0.0", 56789),
    anserpc.WithIPCEndpoint("/var/run/anser.sock"),
)
```

The following is output when appliaction starts.
```
INFO[03-06|12:06:11] Application register service(s):
INFO[03-06|12:06:11] built-in_1.0(public) -> Hello
INFO[03-06|12:06:11] system: network_1.0(public) -> IP
INFO[03-06|12:06:11] system: network_1.0(public) -> Ping
INFO[03-06|12:06:11] system: network_1.0(public) -> Restart
INFO[03-06|12:06:11] system: storage_1.0(public) -> Add
INFO[03-06|12:06:11] Application: running using 2 server(s)
INFO[03-06|12:06:11] HTTP: addr is [::]:56789
INFO[03-06|12:06:11] HTTP: virtual host is localhost
INFO[03-06|12:06:11] HTTP: denied method(s): DELETE/PUT
INFO[03-06|12:06:11] Websocket: enabled
INFO[03-06|12:06:11] IPC: path is /var/run/anser.sock
INFO[03-06|12:06:11] Server(s) shutdown on interrupt(CTRL+C)
INFO[03-06|12:06:11] Application started
```
