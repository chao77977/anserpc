# anserpc — Design Document

This document describes the architecture and internal design of `anserpc`, a
lightweight JSON-RPC 2.0 server library for Go. It is intended for contributors
and users who want to understand how the library works beyond the public API.

## 1. Goals and Scope

- Implement the [JSON-RPC 2.0](https://www.jsonrpc.org/specification) request
  model with a small, pragmatic set of extensions.
- Expose ordinary Go structs as RPC services with no code generation.
- Support multiple transports (HTTP, WebSocket, IPC) behind one request/response
  pipeline.
- Keep the dependency surface and configuration surface small.

Non-goals: this library does not implement JSON-RPC notifications/subscriptions
as a first-class feature; it does not provide authentication/authorization
beyond transport-level checks (vhost allowlist, denied methods, content-type
validation). A companion Go client ships in the `client` subpackage (see §4.10).

## 2. Protocol Model

`anserpc` follows JSON-RPC 2.0 but extends the request object with addressing
fields so one endpoint can host many logically grouped services.

Request object (`jsonMessage` in `codec.go`):

| Field | JSON key | Purpose |
| --- | --- | --- |
| Version | `jsonrpc` | Must equal `"2.0"`. |
| Group | `group` | Logical namespace (optional). |
| Service | `service` | Service name (required). |
| ServiceVersion | `service_version` | Service version (optional). |
| Method | `method` | Method name (required). |
| Params | `params` | Positional argument array. |
| ID | `id` | Correlation id, echoed back. |
| Result | `result` | Present on success response. |
| Error | `error` | Present on error response (`code`/`message`/`data`). |

Validation rules (`jsonMessage.doValidate`):

- `jsonrpc` must be exactly `"2.0"` → otherwise `-32001`.
- `service` and `method` must be non-empty → otherwise `-32002`.

Addressing is **case-insensitive**: group, service, and method names are
normalized to lower case via `util.FormatName`. A service's identity (its
"fingerprint") is `name` optionally suffixed with `_version`.

## 3. High-Level Architecture

```
                         ┌───────────────────────────┐
   HTTP / WS client ───► │        httpServer         │
                         │  (net/http + middleware)  │
                         └─────────────┬─────────────┘
                                       │  serviceCodec (jsonCodec / webSocketCodec)
   Unix socket client ──► ┌────────────┴────────────┐
                         │        ipcServer         │
                         └────────────┬─────────────┘
                                      │
                               ┌──────▼───────┐
                               │   doHandle   │   (handler.go)
                               │  read → run  │
                               └──────┬───────┘
                                      │ lookup
                              ┌───────▼────────┐
                              │ serviceRegistry│   (service.go)
                              │ group→service→ │
                              │   callback     │
                              └───────┬────────┘
                                      │ reflect.Call
                              ┌───────▼────────┐
                              │ user receiver  │
                              │   (struct)     │
                              └────────────────┘
```

The `Anser` type (`anser.go`) is the application façade. It owns the options,
the service registry, and the lifecycle of the HTTP and IPC servers.

## 4. Component Breakdown

### 4.1 Application façade — `anser.go`

`Anser` holds:

- `opts *options` — resolved configuration.
- `sr *serviceRegistry` — the shared service registry.
- `rs *httpServer`, `is *ipcServer` — transport servers, each guarded by its own mutex.
- `wg sync.WaitGroup`, `nRunning uint64` — lifecycle bookkeeping.

`New(...Option)` applies options over defaults and initializes the registry and
a process-wide safe logger.

`Run()` orchestrates startup:

1. `interruptHandle()` installs a SIGINT/SIGTERM handler (unless disabled) that
   calls `Close()`.
2. If `rpc` is configured, enable the HTTP server.
3. If `ipc` is configured, enable the IPC server.
4. Log status, then block on `wg.Wait()` until all servers exit.

`Close()` stops both servers and waits for their goroutines to drain. Server
enable/disable methods are idempotent and guarded by per-server mutexes so the
state transitions are concurrency-safe.

### 4.2 Options — `opt.go`

Configuration uses the functional-options pattern. Each option implements
`Option` with an `apply(*options)` method. `options` aggregates:

- `rpc *rpcEndpoint` — host/port; nil disables HTTP.
- `ipc ipcEndpoint` — socket path; empty disables IPC.
- `log *logOpt` — logging configuration.
- `http *httpOpt` — HTTP middleware configuration (see `http.go`).
- `intrpt *interruptOpt` — interrupt-handler toggle.

HTTP options merge into the defaults (vhosts, denied methods, allowed content
types accumulate rather than replace), which is why `withDefaultHTTPOpt`
pre-seeds `localhost`, `DELETE`/`PUT`, and the three JSON content types.

### 4.3 Service Registry — `service.go`

The registry is a two-level map: `group name → *group → []*service`.

- Each `group` keeps its services in a slice **sorted by fingerprint** and uses
  binary search (`sort.Search`) for insert/lookup. This keeps lookups
  `O(log n)` and allows versioned services of the same name to coexist.
- `makeService` uses reflection to build a `*service` whose `callbacks` map
  method name → `*callback`.
- `makeCallbacks` iterates the receiver's exported methods; `makeCallback`
  inspects each method's signature:
  - Detects an optional leading `context.Context` parameter (`hasCtx`).
  - Records the remaining input types as `argTypes`.
  - Classifies the return shape into `returnType`:
    `-1` none, `0` error-only, `1` result+error. Invalid shapes are rejected at
    registration time (`-32004`, `-32005`).

`serviceRegistry.callback(group, service, version, method)` resolves a request
to a concrete `*callback`, returning `nil` when the service is missing, not
public, or the method is unknown — which the handler maps to "method not found".

Built-in services (`api.go`) are registered automatically: `Hello` (health
check) and `Metrics` (dumps the go-metrics registry as JSON).

### 4.4 Codec Layer — `codec.go` and `types.go`

The `serviceCodec` interface decouples the request pipeline from the transport:

```go
type serviceCodec interface {
    readBatch() ([]*jsonMessage, bool, error)
    writeTo(context.Context, interface{}) error
    close()
}
```

`jsonCodec` is the base implementation around an `encode`/`decode` pair and a
`CloserAndDeadline` connection. `readBatch` peeks the first significant byte to
decide whether the payload is a single object or a batch array, then decodes
accordingly. `writeTo` is mutex-guarded and sets a write deadline (from the
context deadline, or a 10s default).

`codecSet` is a concurrency-safe set of live codecs so a server can close all
open connections during shutdown.

Argument decoding (`retrieveArgs`) decodes the positional `params` array into
`reflect.Value`s matching `argTypes`, enforcing arity (`-32007`) and
required/pointer semantics (`-32008`). `zeroArgs` fills trailing pointer
parameters with zero values when omitted.

Error construction (`makeJSONErrorMessage`) inspects the returned `error`
against the `ResultError` / `ResultDataError` / `ResultCodeError` interfaces to
extract code/message/data, falling back to the default code `-32000`.

### 4.5 Request Handler — `handler.go`

`doHandle(ctx, codec, registry)` is the single entry point used by every
transport:

1. `codec.readBatch()` to parse one or many messages.
2. Build a `handler` and dispatch each message, then write the response(s).

For each message `handler.handle`:

1. Validate the message shape.
2. Resolve the callback via the registry.
3. Decode arguments.
4. Run the call **in its own goroutine**, delivering the result on a buffered
   channel.

`handler.wait` blocks on that channel with a timeout timer
(`_defTimeout`, 3600s). On completion it increments success/failure counters; on
timeout it returns `-32009`. The actual invocation (`handler.call`) assembles
the argument list (receiver, optional context, decoded args), calls via
reflection, and uses `recover()` to convert panics into `-32006` instead of
crashing the server.

Batches are dispatched concurrently (one goroutine per message) and the results
are reassembled in request order.

### 4.6 Transports

**HTTP — `http.go`.** `httpServer` wraps `net/http`. The handler chain is built
inside-out in `newHttpServer`:

```
websocket → gzip → virtualHost → validate → httpServer.ServeHTTP
```

- `validateHandler`: allows empty `GET` for health checks; rejects denied
  methods (405), oversized bodies (413, 5 MiB cap), and disallowed content
  types (415); passes `OPTIONS` through.
- `virtualHostHandler`: allows requests with no host or an IP host; otherwise
  requires the host to be in the vhost allowlist (or `*`), else 403.
- `gzipWriteHandler`: transparently gzips responses when the client advertises
  `Accept-Encoding: gzip`, reusing writers from a `sync.Pool`.
- `httpServer.ServeHTTP`: wraps the request body/response in an
  `httpServerConn`, builds a `jsonCodec`, registers it in the codec set, and
  calls `doHandle`. The remote address is attached to the context.

**WebSocket — `websocket.go`.** `websocketHandler` upgrades qualifying requests
with `gorilla/websocket`. It runs a dedicated reader goroutine feeding a channel
loop that dispatches each (batch of) message(s) through the same `handler`.
`webSocketCodec` embeds `jsonCodec` and adds a server-side ping keep-alive: a
timer pings every 60s and is reset whenever a message is successfully written.

> Dependency note: `gorilla/websocket` is imported here but is not declared in
> `go.mod`. This should be added for the module to build with the WebSocket
> transport.

**IPC — `ipc.go`.** `ipcServer` listens on a Unix domain socket. `setPath`
validates path length (≤128), ensures the parent directory exists, removes any
stale socket, and after `Listen` chmods the socket to `0600`. The accept loop
tolerates temporary errors and serves each connection via `doHandle`.

### 4.7 Logging — `log.go`

A process-wide logger `_xlog` is initialized once (`sync.Once`) via
`newSafeLogger`. It wraps `inconshreveable/log15` and supports either terminal
output (default) or JSON file output (`silent`), filtered by level. A custom
`Logger` can be injected with `WithLoggerOpt`.

### 4.8 Metrics — `metrics.go`

Three `rcrowley/go-metrics` counters track `anser/requests`, `anser/success`,
and `anser/failure`. They are incremented in the handler path and exposed via
the built-in `Metrics` method.

### 4.9 Utilities — `util/`

- `datastructure.go`: `StringSet` with case-normalizing constructors and `Merge`
  (used for vhosts, denied methods, content types).
- `path.go`: filesystem path existence and creation helpers.
- `string.go`: `FormatName` (lower-casing) and a lazy `Fmt` wrapper.
- `interrupt.go`: a singleton interrupt monitor that fans a single
  SIGINT/SIGTERM out to all registered callbacks.
- `net.go`: `IsTemporaryError` for the IPC accept loop.

### 4.10 Client — `client/`

A companion Go client lives in the `client` subpackage. It reuses the exact
wire format (`wireRequest`/`wireResponse` mirror the server's `jsonMessage`)
and is organized around a small `transport` interface:

```go
type transport interface {
    roundTrip(ctx context.Context, reqs []*wireRequest) ([]*wireResponse, error)
    close() error
}
```

- `Client` is transport-agnostic and exposes `Call` (single) and `BatchCall`
  (JSON-RPC batch). Request ids are assigned from an atomic counter; batch
  responses are matched back to requests by id, so server reordering is safe.
- `httpTransport` (`DialHTTP`) issues one HTTP POST per round-trip and is
  concurrency-safe via `net/http`. `WithHTTPClient` injects a custom client.
- `ipcTransport` (`DialIPC`) holds a persistent unix-socket connection and
  serializes round-trips with a mutex, using a `json.Encoder`/`Decoder` pair
  (decoder `UseNumber`, matching the server).
- `wsTransport` (`DialWebSocket`) holds a persistent gorilla/websocket
  connection, likewise mutex-serialized.
- `Error` mirrors the server error object (code/message/data) and implements
  the same accessor methods; `AsError` extracts it from a returned `error`,
  and `ErrorData` decodes the optional data payload.
- Context deadlines/cancellation are honored: the streaming transports set
  socket deadlines from the context and check for cancellation before I/O.

## 5. Request Lifecycle (End to End)

Taking an HTTP call to `system/network/IP` as an example:

1. Client POSTs/GETs a JSON-RPC body to `127.0.0.1:56789`.
2. `validateHandler` → `virtualHostHandler` → `gzipWriteHandler` pass the request
   through (not a WebSocket upgrade).
3. `httpServer.ServeHTTP` builds a `jsonCodec` over the request/response and calls
   `doHandle`.
4. `readBatch` decodes a single `jsonMessage`.
5. `handler.handle` validates it, resolves the `IP` callback for the lower-cased
   `system`/`network`/`1.0`, decodes empty params.
6. The callback runs in a goroutine; `IP()` returns `("10.0.0.2", nil)`.
7. `handler.wait` receives the result, bumps the success counter, and builds a
   response message.
8. `writeTo` encodes `{"jsonrpc":"2.0","id":...,"result":"10.0.0.2"}` back,
   gzip-compressed if the client asked for it.

## 6. Concurrency Model

- Per-server mutexes (`rsMu`, `isMu`) serialize lifecycle transitions.
- The registry uses a single mutex around its maps; service slices are
  copy-on-write in `group.add`, so lookups read a stable slice.
- Each in-flight RPC runs in its own goroutine; batches fan out and join in
  order.
- Codec writes are mutex-guarded; `codecSet` protects the live-connection set.
- Response delivery uses buffered channels (capacity 1) so a late-arriving
  result after timeout does not block its goroutine.

## 7. Security Considerations

- **Transport filtering:** HTTP enforces a virtual-host allowlist, a denied-method
  list, a content-type allowlist, and a 5 MiB request-size cap.
- **IPC permissions:** the Unix socket is created with `0600`.
- **Panic isolation:** user method panics are recovered and reported as error
  `-32006` rather than taking down the server.
- **No built-in auth:** authentication/authorization is the caller's
  responsibility; `public=false` only hides a service from dispatch, it is not an
  access-control mechanism per client.

## 8. Known Issues / Future Work

- **Missing `gorilla/websocket` in `go.mod`.** The module does not build the
  WebSocket transport out of the box until this dependency is added and
  `go mod tidy` is run.
- **`ipcServer.stop` double-locks.** `stop()` calls `i.mu.Lock()` twice
  (the deferred call should be `Unlock`), which will deadlock on shutdown. This
  is a bug worth fixing.
- **`group.load` fallback.** When a binary search overshoots, it clamps to the
  last element; callers must verify the returned service name matches (the
  registry's `callback` does this).
- **Timeout granularity.** The handler timeout is a fixed 3600s constant; making
  it configurable per-application would be a useful enhancement.
- **Notifications/subscriptions** are not implemented; only request/response.
  A companion Go client is provided (see §4.10).

## 9. File Map

| File | Responsibility |
| --- | --- |
| `anser.go` | `Anser` façade, lifecycle, server orchestration. |
| `opt.go` | Functional options and defaults. |
| `api.go` | `API` type and built-in services. |
| `service.go` | Service registry, reflection-based callback building. |
| `handler.go` | Request dispatch, invocation, timeout, panic recovery. |
| `codec.go` | `jsonMessage`, JSON codec, batch parsing, error mapping, codec set. |
| `http.go` | HTTP server and middleware chain. |
| `websocket.go` | WebSocket upgrade, codec, ping keep-alive. |
| `ipc.go` | Unix-socket server. |
| `types.go` | Shared interfaces (`serviceCodec`, `Conn`, result-error interfaces). |
| `errors.go` | Error codes and `StatusError`. |
| `log.go` | Logging setup. |
| `metrics.go` | Request counters. |
| `doc.go` | Package doc comment. |
| `util/` | Helpers: string sets, paths, names, interrupts, net errors. |
