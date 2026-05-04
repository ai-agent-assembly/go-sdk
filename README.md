# go-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/agent-assembly/go-sdk.svg)](https://pkg.go.dev/github.com/agent-assembly/go-sdk)
[![Live Docs](https://img.shields.io/badge/docs-live-blue)](https://ai-agent-assembly.github.io/go-sdk/)
[![Go Test Matrix](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/go-test.yml/badge.svg)](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/go-test.yml)
[![Lint](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/lint.yml/badge.svg)](https://github.com/AI-agent-assembly/go-sdk/actions/workflows/lint.yml)
[![Codecov](https://codecov.io/gh/AI-agent-assembly/go-sdk/graph/badge.svg)](https://codecov.io/gh/AI-agent-assembly/go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/agent-assembly/go-sdk)](https://goreportcard.com/report/github.com/agent-assembly/go-sdk)
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

> **Note** — `go get` and pkg.go.dev indexing are blocked today pending a module-path rename. The `go.mod` declares `github.com/agent-assembly/go-sdk` while the canonical GitHub URL is `github.com/AI-agent-assembly/go-sdk`. Until that rename ticket lands, clone the repo and use a `replace` directive in your consumer's `go.mod`.

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

    "github.com/agent-assembly/go-sdk/assembly"
)

func main() {
    a, err := assembly.Init(context.Background(),
        assembly.WithGatewayURL("https://your-gateway.com"),
        assembly.WithAPIKey("xxx"),
        assembly.WithFailClosed(false),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer a.Close()
}
```

## Development

- `make fmt`
- `make lint`
- `make test`

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
