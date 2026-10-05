# Contributing to anserpc

Thanks for your interest in contributing! This document explains how to build,
test, and submit changes.

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By
participating, you are expected to uphold it. Please report unacceptable
behavior as described there.

## Getting Started

anserpc is a standard Go module. You need Go 1.17 or newer.

```sh
git clone https://github.com/chao77977/anserpc.git
cd anserpc
go build ./...
go test ./...
```

## Development Workflow

1. Fork the repository and create a topic branch from `main`
   (e.g. `fix/ipc-deadlock`, `feat/configurable-timeout`).
2. Make your change, including tests.
3. Run the full local check suite (below) and make sure it passes.
4. Open a pull request against `main` using the PR template.

### Local checks

These mirror what CI runs. Please run them before pushing:

```sh
# formatting (CI fails on unformatted files)
gofmt -s -l .

# build and static analysis
go build ./...
go vet ./...

# tests with the race detector
go test -race ./...

# linter (same config CI uses)
golangci-lint run ./...
```

Install `golangci-lint` from <https://golangci-lint.run/usage/install/>.
CI pins version `v1.64.8`.

## Coding Guidelines

- Format all code with `gofmt -s`. Imports should be grouped (goimports).
- Keep public API changes minimal and documented; this library is pre-1.0 and
  the API may still evolve, but avoid gratuitous breakage.
- Add or update tests for any behavior change. The project targets meaningful
  coverage; do not regress it.
- Match the existing style and naming conventions in the file you are editing.
- Prefer small, focused commits with clear messages. We loosely follow
  [Conventional Commits](https://www.conventionalcommits.org/)
  (`fix:`, `feat:`, `test:`, `docs:`, `ci:`, `refactor:`).

## Pull Requests

- Reference the issue your PR addresses (e.g. `Closes #42`).
- Describe what changed, why, and how you verified it.
- Keep PRs scoped to a single logical change where possible.
- All CI checks must pass. The `govulncheck` job is informational and may
  report Go standard-library advisories tied to the runner's Go version; it
  does not block merges.

## Reporting Bugs and Requesting Features

Use the issue templates under
[.github/ISSUE_TEMPLATE](.github/ISSUE_TEMPLATE). For security issues, do
**not** open a public issue — follow [SECURITY.md](SECURITY.md) instead.

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE), the same license as the project.
