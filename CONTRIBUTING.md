# Contributing to go-sdk

Thanks for taking the time to contribute. This guide describes how to set up
a development environment, run the test suite, and submit a pull request.

## Development Environment

### Required

- **Go ≥ 1.24** — install via the official distribution or `brew install go`.
- **golangci-lint** — `brew install golangci-lint` (or see the
  [project's install guide](https://golangci-lint.run/welcome/install/)).
- **make** — for the `Makefile` targets (`fmt`, `lint`, `test`).

### Optional

- **A C compiler + Rust toolchain** — required only when building or testing
  the native FFI transport (`-tags aa_ffi_go`). The default pure-Go transport
  works without either. The Rust side ships the `aa_ffi_go` shared library that
  the CGo bridge links against; see `internal/ffi/cgo_bridge.go` for the
  `#cgo LDFLAGS` line and the build-tag-gated source layout.
- **Hugo (extended) ≥ 0.146** — required only if you want to preview the docs
  site locally (`cd website && hugo server`).

### First-time setup

```bash
git clone https://github.com/AI-agent-assembly/go-sdk.git
cd go-sdk
go mod download
make test
```

If the test run is green, your environment is ready.

## Running Tests and Checks

## Idiomatic Go Conventions

## Pull Request Checklist
