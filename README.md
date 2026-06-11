# go-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/ai-agent-assembly/go-sdk.svg)](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk)
[![Live Docs](https://img.shields.io/badge/docs-live-blue)](https://ai-agent-assembly.github.io/go-sdk/)
[![Go Test Matrix](https://github.com/ai-agent-assembly/go-sdk/actions/workflows/go-test.yml/badge.svg)](https://github.com/ai-agent-assembly/go-sdk/actions/workflows/go-test.yml)
[![Lint](https://github.com/ai-agent-assembly/go-sdk/actions/workflows/lint.yml/badge.svg)](https://github.com/ai-agent-assembly/go-sdk/actions/workflows/lint.yml)
[![Codecov](https://codecov.io/gh/ai-agent-assembly/go-sdk/graph/badge.svg)](https://codecov.io/gh/ai-agent-assembly/go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/ai-agent-assembly/go-sdk)](https://goreportcard.com/report/github.com/ai-agent-assembly/go-sdk)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Go SDK for [AI Agent Assembly](https://github.com/ai-agent-assembly) — runtime governance for AI agent tool calls.

The SDK initialises in a few lines, propagates agent identity through `context.Context`, wraps your agent's tool slice with policy enforcement, and forwards every check + result to the AAASM gateway over gRPC or HTTP.

## Project status

`go-sdk` is **pre-release**. Published tags are on the `v0.0.1-alpha` line (latest [`v0.0.1-alpha.3`](https://github.com/ai-agent-assembly/go-sdk/releases)); the [`VERSION`](VERSION) file pins the gateway **protocol version** the SDK is built against (currently `0.0.0`). The public `assembly` package API may still change between alpha tags — pin an exact tag in your `go.mod` and review the [release notes](https://github.com/ai-agent-assembly/go-sdk/releases) before upgrading. See [Core-runtime compatibility](docs/compatibility.md) for the version/protocol contract.

Anything outside the `assembly/` package (`internal/`, `examples/`) is not part of the public API and may change without notice.

## Prerequisites

- **Go ≥ 1.26** — the floor declared in `go.mod`.
- For production: an operator-issued gateway URL and API key. For local development you can skip both — `Init` discovers a gateway on `http://localhost:7391` (see [Configuration](docs/configuration.md#gateway-and-credential-resolution)).
- *(Optional)* a C compiler — only needed if you build with `-tags aa_ffi_go` to enable the native FFI transport. The default transport is pure-Go and runs cleanly with `CGO_ENABLED=0`.

## Installation

```bash
go get github.com/ai-agent-assembly/go-sdk
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

    "github.com/ai-agent-assembly/go-sdk/assembly"
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
- **API reference** — [pkg.go.dev/github.com/ai-agent-assembly/go-sdk](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk) (auto-generated from godoc; preview locally with `godoc -http=:6060`).
- **Architecture** — [docs/architecture.md](docs/architecture.md) and [docs/api-reference.md](docs/api-reference.md).
- **Contributing** — [CONTRIBUTING.md](CONTRIBUTING.md).

## AI Agent Assembly ecosystem

- **Organization** — [github.com/ai-agent-assembly](https://github.com/ai-agent-assembly): org profile and the full production-repo map.
- **Core runtime** — [ai-agent-assembly/agent-assembly](https://github.com/ai-agent-assembly/agent-assembly): the gateway, policy engine, proxy, and eBPF layers this SDK talks to.
- **Protocol spec** — the gateway wire protocol this SDK is pinned to lives in the core monorepo at [docs/src/protocol](https://github.com/ai-agent-assembly/agent-assembly/tree/master/docs/src/protocol).
- **Canonical docs** — the org-wide documentation site at [ai-agent-assembly.github.io/agent-assembly-docs](https://ai-agent-assembly.github.io/agent-assembly-docs/).
- **Release notes** — [github.com/ai-agent-assembly/go-sdk/releases](https://github.com/ai-agent-assembly/go-sdk/releases).

## Support & Security

- **Questions / bugs / feature requests** — open an issue at [github.com/ai-agent-assembly/go-sdk/issues](https://github.com/ai-agent-assembly/go-sdk/issues).
- **Security vulnerabilities** — do **not** file a public issue; report privately via the [security policy](https://github.com/ai-agent-assembly/go-sdk/security/policy).
- **Troubleshooting** — common errors and fixes are in [docs/troubleshooting.md](docs/troubleshooting.md).

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
