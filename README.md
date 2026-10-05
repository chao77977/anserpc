# anserpc — A Lightweight JSON-RPC 2.0 Library for Go

> Anser cygnoides (Chinese name 鸿雁), the swan goose.

`anserpc` is a small, dependency-light library that implements the
[JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification). It lets you
expose plain Go structs as RPC services over multiple transports with minimal
boilerplate.

Supported transports:

- **HTTP** — JSON-RPC over HTTP
- **WebSocket** — JSON-RPC over a persistent WebSocket connection (with server-side ping keep-alive)
- **IPC** — JSON-RPC over a Unix domain socket

Key features:

- Register any struct as a service; exported methods are auto-discovered via reflection.
- Logical addressing by **group / service / service-version / method**.
- Per-service public/private visibility.
- Batch request support.
- Built-in services for health check (`Hello`) and runtime metrics (`Metrics`).
- Configurable HTTP virtual-host allowlist, denied methods, content-type checks, and gzip responses.
- Graceful shutdown on `SIGINT` / `SIGTERM` (CTRL+C).
- Pluggable logging (terminal or JSON file output, or your own `Logger`).

See [docs/DESIGN.md](docs/DESIGN.md) for the architecture and internals.

## Install

```sh
go get github.com/chao77977/anserpc
```

Requires Go 1.15+.

> **Note:** `websocket.go` imports `github.com/gorilla/websocket`, which is not
> currently listed in `go.mod`. If you build the WebSocket transport you may
> need to run `go get github.com/gorilla/websocket` and tidy the module.

## Quick Start: RPC over HTTP

```go
package main

import "errors"
import "github.com/chao77977/anserpc"

func main() {
    app := anserpc.New(
        anserpc.WithRPCEndpoint("0.0.0.0", 56789),
    )

    // register services
    app.Register("system", "network", "1.0", true, &network{})
    app.Register("system", "storage", "1.0", false, &storage{})

    // blocks until interrupted
    app.Run()
}

// service: network
type network struct{}

func (n *network) Ping() error          { return errors.New("unknown host") }
func (n *network) IP() (string, error)  { return "10.0.0.2", nil }
func (n *network) Restart()             {}

// service: storage
type storage struct{}

func (s *storage) Add() error { return &myErr{} }

// a rich error carrying code / message / data
type myErr struct{}

func (e *myErr) Error() string          { return e.ErrorMessage() }
func (e *myErr) ErrorCode() int         { return -1 }
func (e *myErr) ErrorMessage() string   { return "error message" }
func (e *myErr) ErrorData() interface{} { return struct{}{} }
```

## Configuration Options

Pass options to `anserpc.New(...)`:

| Option | Description |
| --- | --- |
| `WithRPCEndpoint(host string, port int)` | Enable the HTTP/WebSocket server on the given address. |
| `WithDefaultRPCEndpoint()` | Enable HTTP on the default `127.0.0.1:56789`. |
| `WithIPCEndpoint(path string)` | Enable the IPC (Unix socket) server at `path`. |
| `WithDefaultIPCEndpoint()` | Enable IPC on the default `/var/run/anser.rpc`. |
| `WithLogFileOpt(path string, filterLvl logLvl)` | Write logs as JSON to a file at the given level. |
| `WithDefaultLogOpt()` | Terminal logging at `LvlDebug` (the default). |
| `WithLoggerOpt(logger Logger)` | Use your own `Logger` implementation. |
| `WithHTTPVhostOpt(vhosts ...string)` | Add allowed virtual hosts (`"*"` allows any). |
| `WithHTTPDeniedMethodOpt(methods ...string)` | Deny specific HTTP methods. |
| `WithDisableInterruptHandler()` | Do not install the CTRL+C shutdown handler. |

Log levels: `LvlCrit`, `LvlError`, `LvlWarn`, `LvlInfo`, `LvlDebug`.

HTTP defaults: virtual host `localhost`; denied methods `DELETE` and `PUT`;
allowed content types `application/json`, `application/json-rpc`,
`application/jsonrequest`; WebSocket enabled; max request body 5 MiB.

## Registering Services

A service is addressed by four coordinates:

- **group** — logical namespace (e.g. `"system"`); the same service name can live in different groups.
- **service** — the service name (e.g. `"network"`).
- **service version** — allows multiple versions of one service (e.g. `"1.0"`).
- **public** — if `false`, the service is registered but not callable from clients.

Group and version are optional; pass empty strings to omit them.

```go
// without group / version
app.Register("", "network", "", true, &network{})
app.Register("", "storage", "", false, &storage{})
```

Equivalent registration APIs:

```go
// RegisterService: no group
app.RegisterService("network", "1.0", true, &network{})

// RegisterWithGroup: fluent group registration
grp := app.RegisterWithGroup("system")
grp.Register("network", "1.0", true, &network{})
grp.Register("storage", "1.0", true, &storage{})

// RegisterAPI: pass an *API struct directly
app.RegisterAPI(&anserpc.API{
    Group: "system", Service: "network", Version: "1.0",
    Public: true, Receiver: &network{},
})
```

> Group, service, and method names are matched **case-insensitively** (normalized to lower case).

### Method Signature Rules

Exported methods of the receiver are discovered automatically. The first
parameter may optionally be a `context.Context`. Return values must be one of:

- no return value
- a single `error`
- a result value **and** an `error` (in that order)

```go
func (s *svc) A()                          // ok: no return
func (s *svc) B() error                     // ok: error only
func (s *svc) C() (string, error)           // ok: result + error
func (s *svc) D(ctx context.Context) error  // ok: leading context
func (s *svc) E(ctx context.Context, n int) (int, error) // ok
```

### Rich Errors

To return a JSON-RPC `error` object with a custom code, message, and data,
implement one of the error interfaces (fullest shown):

```go
type ResultError interface {
    Error() string
    ErrorCode() int
    ErrorMessage() string
    ErrorData() interface{}
}
```

Partial interfaces `ResultCodeError` (code only) and `ResultDataError`
(code + data) are also recognized. A plain `error` maps to the default error
code `-32000` with its message.

## Startup Output

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

## Calling Services

### Built-in Services

Health check (`Hello`):

```sh
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"service":"built-in","method":"Hello"}' \
  http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"result":"olleh"}
```

Metrics (`Metrics`):

```sh
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"service":"built-in","method":"Metrics"}' \
  http://127.0.0.1:56789

{"jsonrpc":"2.0","id":10001,"result":"{\"anser/failure\":{\"count\":1},\"anser/requests\":{\"count\":2},\"anser/success\":{\"count\":1}}"}
```

### Registered Services

```sh
# returns a plain error
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"group":"system","service":"network","method":"Ping"}' \
  http://127.0.0.1:56789
{"jsonrpc":"2.0","id":10001,"error":{"code":-32000,"message":"unknown host"}}

# returns a result
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"group":"system","service":"network","method":"IP"}' \
  http://127.0.0.1:56789
{"jsonrpc":"2.0","id":10001,"result":"10.0.0.2"}

# returns a rich error (code/message/data)
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"group":"system","service":"storage","method":"Add"}' \
  http://127.0.0.1:56789
{"jsonrpc":"2.0","id":10001,"error":{"code":-1,"message":"error message","data":{}}}

# unknown method
curl -H "Content-Type: application/json" -X GET \
  --data '{"jsonrpc":"2.0","id":10001,"group":"system","service":"storage","method":"NotFound"}' \
  http://127.0.0.1:56789
{"jsonrpc":"2.0","id":10001,"error":{"code":-32601,"message":"method not found"}}
```

### Request / Response Format

Requests extend standard JSON-RPC 2.0 with `group`, `service`, and
`service_version` fields for addressing:

```json
{
  "jsonrpc": "2.0",
  "id": 10001,
  "group": "system",
  "service": "network",
  "service_version": "1.0",
  "method": "IP",
  "params": []
}
```

`params` must be a JSON array (positional). Pointer-typed parameters may be
omitted and default to the zero value.

## Client SDK

A Go client lives in the [`client`](client) subpackage. It speaks the same
wire format over HTTP, WebSocket, and IPC, and supports single and batch calls.

```go
import "github.com/chao77977/anserpc/client"

c, err := client.DialHTTP("http://127.0.0.1:56789")
if err != nil {
    // handle
}
defer c.Close()

// single call
var ip string
err = c.Call(context.Background(), &ip, client.Request{
    Group: "system", Service: "network", Version: "1.0", Method: "IP",
})

// typed error with code / message / data
if e, ok := client.AsError(err); ok {
    log.Printf("rpc error %d: %s", e.ErrorCode(), e.ErrorMessage())
}
```

Other transports:

```go
c, _ := client.DialWebSocket("ws://127.0.0.1:56789")
c, _ := client.DialIPC("/var/run/anser.sock")
```

Batch call:

```go
var a, b int
elems := []client.BatchElem{
    {Request: client.Request{Service: "calc", Method: "Add", Params: []interface{}{1, 2}}, Result: &a},
    {Request: client.Request{Service: "calc", Method: "Add", Params: []interface{}{3, 4}}, Result: &b},
}
_ = c.BatchCall(context.Background(), elems)
// each element's per-call error is in elems[i].Error
```

## Running Multiple Servers

HTTP and IPC can run simultaneously:

```go
app := anserpc.New(
    anserpc.WithRPCEndpoint("0.0.0.0", 56789),
    anserpc.WithIPCEndpoint("/var/run/anser.sock"),
)
```

## Error Codes

| Code | Meaning |
| --- | --- |
| `-32600` | invalid request |
| `-32601` | method not found |
| `-32602` | invalid params |
| `-32603` | internal error |
| `-32700` | parse error |
| `-32000` | default error code for plain errors |
| `-32001` | invalid version |
| `-32002` | service or method not found |
| `-32003` | service not found |
| `-32004` | error return value not found |
| `-32005` | too many return results |
| `-32006` | method running crash (recovered panic) |
| `-32007` | too many params |
| `-32008` | missing value for params |
| `-32009` | handling message timeout |

## License

`anserpc` source code is licensed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
