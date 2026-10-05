# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Companion Go client SDK in the `client` subpackage, speaking the same
  JSON-RPC 2.0 wire format over HTTP, WebSocket and IPC. Supports single and
  batch calls, a typed `Error` (code/message/data), context cancellation, and
  a pluggable `*http.Client`.
- GitHub Actions CI: build, `go vet`, `go test -race`, gofmt check across a Go
  version matrix (1.17 / 1.21 / 1.23), plus golangci-lint and govulncheck.
- `.golangci.yml` linter configuration and `.gitignore`.
- Real unit and integration test suite covering the service registry, codec,
  handler, and all three transports. Statement coverage raised to ~79%.
- `docs/DESIGN.md` architecture document and `docs/ROADMAP.md`.

### Fixed
- `ipcServer.stop` deadlock caused by a deferred `Lock` instead of `Unlock`.
- `codec.retrieveArgs` off-by-one that caused an index-out-of-range panic when
  a request supplied exactly one more positional parameter than declared.

### Changed
- Added the missing `github.com/gorilla/websocket` dependency to `go.mod` so
  the WebSocket transport builds.
- Bumped `golang.org/x/sys` to resolve advisory GO-2022-0493.
- Replaced raw string context keys with an unexported key type (staticcheck
  SA1029).

## [0.1.0] - 2026-10-04

### Added
- Initial tagged release: JSON-RPC 2.0 server over HTTP, WebSocket and IPC.
- Reflection-based service registration (group / service / version / public).
- Built-in `Hello` and `Metrics` services.
- Configurable HTTP vhost allowlist, denied methods, gzip, and size caps.
- Graceful shutdown on SIGINT/SIGTERM.

[Unreleased]: https://github.com/chao77977/anserpc/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/chao77977/anserpc/releases/tag/v0.1.0
