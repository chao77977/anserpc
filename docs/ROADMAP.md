# Roadmap

This document tracks planned work. The active milestone defines the next
release's scope.

## Milestone: v0.2.0 — Project Quality & Hardening

**Goal:** Bring anserpc up to the standard of a high-quality, trustworthy
open-source Go project — strong test coverage, continuous integration,
contributor on-ramps, and the health files the community expects.

**Target:** the next minor release after `v0.1.0`.

### Scope

#### 1. Testing
- [ ] Add real unit tests (the current `example_test.go` is only a usage sample
      with no `Test*` functions).
- [ ] Cover the service registry: registration, versioning, public/private,
      case-insensitive lookup.
- [ ] Cover the codec: single vs. batch parsing, argument decoding, error
      mapping (`ResultError` family).
- [ ] Cover the handler: success/error/timeout/panic-recovery paths.
- [ ] Add integration tests for the HTTP, WebSocket and IPC transports.
- [ ] Target a meaningful coverage threshold (e.g. 70%+) and report it.

#### 2. Continuous Integration
- [ ] Add `.github/workflows/ci.yml`: build, `go vet`, `go test -race`,
      `gofmt`/`goimports` check across supported Go versions (1.17 … latest).
- [ ] Add `govulncheck` to CI to catch dependency vulnerabilities.
- [ ] Add a linter (`golangci-lint`) with a checked-in config.

#### 3. Community & Health Files
- [ ] `CONTRIBUTING.md` — how to build, test, and submit changes.
- [ ] `CODE_OF_CONDUCT.md` — Contributor Covenant.
- [ ] `SECURITY.md` — how to report vulnerabilities.
- [ ] `CHANGELOG.md` — Keep a Changelog format, starting with v0.1.0.
- [ ] `.github/ISSUE_TEMPLATE/` and `PULL_REQUEST_TEMPLATE.md`.

#### 4. Documentation Polish
- [ ] Add CI / coverage / Go Reference (pkg.go.dev) badges to the README.
- [ ] Add runnable `Example*` functions so examples appear on pkg.go.dev.
- [ ] Expand package-level doc comments (`doc.go`).

#### 5. Code Hardening (follow-ups from the v0.1.0 review)
- [ ] Make the handler timeout configurable instead of the fixed 3600s constant.
- [ ] Guard `codecSet.contains` or document it as internal-only.
- [ ] Review `group.load` clamp-to-last-element fallback for edge cases.
- [ ] Audit goroutine lifecycles in the WebSocket read/ping paths for leaks.

### Definition of Done
- CI is green on every supported Go version.
- `go test -race ./...` passes with the coverage target met.
- All health files are present and linked from the README.
- `govulncheck` reports no third-party module vulnerabilities.
