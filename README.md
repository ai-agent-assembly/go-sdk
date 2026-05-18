# go-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/AI-agent-assembly/go-sdk.svg)](https://pkg.go.dev/github.com/AI-agent-assembly/go-sdk)
[![Live Docs](https://img.shields.io/badge/docs-live-blue)](https://ai-agent-assembly.github.io/go-sdk/)
[![Go Test Matrix](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/go-test.yml/badge.svg)](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/go-test.yml)
[![Lint](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/lint.yml/badge.svg)](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/lint.yml)
[![Codecov](https://codecov.io/gh/AI-agent-assembly/go-sdk/graph/badge.svg)](https://codecov.io/gh/AI-agent-assembly/go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/AI-agent-assembly/go-sdk)](https://goreportcard.com/report/github.com/AI-agent-assembly/go-sdk)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Go SDK for [AI Agent Assembly](https://github.com/AI-agent-assembly) — runtime governance for AI agent tool calls.

The SDK initialises in a few lines, propagates agent identity through `context.Context`, wraps your agent's tool slice with policy enforcement, and forwards every check + result to the AAASM gateway over gRPC or HTTP.

## Prerequisites

- **Go ≥ 1.24** — the floor declared in `go.mod`.
- An AAASM gateway URL and API key (operator-issued).
- *(Optional)* a C compiler — only needed if you build with `-tags aa_ffi_go` to enable the native FFI transport. The default transport is pure-Go and runs cleanly with `CGO_ENABLED=0`.

## Installation

```bash
go get github.com/AI-agent-assembly/go-sdk
```

## Layout

```text
assembly/
  init.go
  runtime.go
  options.go
  defaults.go
  validation.go
  governance_client.go
  policy_model.go
  governance_errors.go
  tool_wrapper.go
  wrap_tools.go
  sidecar.go
  interceptor.go
examples/minimal/
```

## Quick Start

```go
import (
    "context"
    "log"

    "github.com/AI-agent-assembly/go-sdk/assembly"
)

ctx := assembly.WithAgentID(context.Background(), "my-agent")
a, err := assembly.Init(ctx, assembly.WithGatewayURL(url), assembly.WithAPIKey(key))
if err != nil {
    log.Fatal(err)
}
defer a.Close()
```

`WithAgentID` attaches the calling agent's identity to `ctx`; the SDK forwards it (and any `WithTraceID` / `WithRunID` values) to the gateway on every `Check` and `RecordResult`. See [Context Propagation](#context-propagation) below for the full set of context helpers.

## Documentation

- **Live site** — [ai-agent-assembly.github.io/go-sdk](https://ai-agent-assembly.github.io/go-sdk/) (Hugo, Hextra theme; built and deployed from `master`).
- **API reference** — [pkg.go.dev/github.com/AI-agent-assembly/go-sdk](https://pkg.go.dev/github.com/AI-agent-assembly/go-sdk) (auto-generated from godoc; preview locally with `godoc -http=:6060`).
- **Architecture** — [docs/architecture.md](docs/architecture.md) and [docs/api-reference.md](docs/api-reference.md).
- **Contributing** — [CONTRIBUTING.md](CONTRIBUTING.md).

## Development

```bash
make fmt              # gofmt -w on all .go files
make lint             # golangci-lint run ./...
make test             # go test ./...

go vet ./...
go test ./assembly                            # one package
go test ./assembly -run TestRegisterAgent     # one test
go test -count=1 -race ./...                  # race detector
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full contributor workflow, including the optional CGo native FFI build (`-tags aa_ffi_go`) and the memory regression harness.

## Context Propagation

- `assembly.WithAgentID` / `assembly.AgentIDFromContext` propagate and read agent identity.
- `assembly.WithTraceID` / `assembly.TraceIDFromContext` propagate explicit trace IDs and fallback to OpenTelemetry span context trace ID when unset.
- `assembly.WithRunID` / `assembly.RunIDFromContext` propagate run identity.
- `assembly.EnsureRunID` guarantees a stable run ID within the same context tree.

## Gateway Context Handling

- `GatewayClient.Check` fails fast when `ctx` is already cancelled.
- If `ctx` has no deadline, `GatewayClient.Check` applies SDK timeout defaults (`500ms` unless overridden by `WithTimeout`).
- The final effective context (values + timeout/deadline) is passed to the gateway transport check call.

## FFI Transport

- CGo bridge module lives in `internal/ffi/cgo_bridge.go` with `#cgo LDFLAGS: -laa_ffi_go`.
- Native FFI path is enabled with build tags: `-tags aa_ffi_go` (and `CGO_ENABLED=1`).
- Pure-Go UDS fallback is selected automatically when `aa_ffi_go` tag is not set.
- `CGO_ENABLED=0` is explicitly supported via fallback transport and CI matrix coverage.
- Optional memory harness test (1M sends): `AAASM_MEMORY_HARNESS=1 go test ./internal/ffi -run TestMemoryRegressionHarness`.

## SonarQube CI Setup

Configure these repository settings for the `SonarQube` workflow:

- Secret: `SONAR_TOKEN`
- Variable: `SONAR_HOST_URL` (for SonarCloud use `https://sonarcloud.io`)
- Variable: `SONAR_PROJECT_KEY`
- Variable: `SONAR_ORGANIZATION` (required for SonarCloud, optional for self-hosted SonarQube)
