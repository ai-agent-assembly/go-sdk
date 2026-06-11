---
title: go-sdk
toc: false
---

# go-sdk · AI Agent Assembly

Go SDK for **AI Agent Assembly** — runtime governance for AI agent tool calls, written in idiomatic Go.

Initialise the runtime, wrap your agent's tools, and every tool call is checked against the gateway policy before it runs and recorded after.

## Quick start

```go
import (
    "context"
    "log"

    "github.com/ai-agent-assembly/go-sdk/assembly"
)

func main() {
    a, err := assembly.Init(context.Background(),
        assembly.WithGatewayURL("https://gateway.example.com"),
        assembly.WithAPIKey("..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer a.Close()
}
```

[Get started →](getting-started/)

## What you get

- **Functional options** — extend configuration via `WithGatewayURL`, `WithAPIKey`, `WithFailClosed`, etc., without breaking call sites.
- **Context propagation** — `AgentID`, `TraceID`, and `RunID` flow through `context.Context`; OpenTelemetry-aware.
- **Tool wrapping** — `WrapTools` adds policy `Check` and `RecordResult` to any tool slice.
- **HTTP & gRPC interceptors** — capture parent agent metadata from incoming requests.
- **Pure-Go by default** — works in container images with `CGO_ENABLED=0`.
- **Native FFI opt-in** — enable the CGo bridge with `-tags aa_ffi_go` for in-process calls.

## Where to next

- [Getting started](getting-started/) — install, configure, and run a first governed call.
- [Configuration](configuration/) — every `Init` option, defaults, enforcement modes, and context helpers.
- [Architecture](architecture/) — module layout, FFI bridge, interceptor flow, context propagation.
- [API reference](api-reference/) — godoc on pkg.go.dev plus local preview instructions.
- [Troubleshooting](troubleshooting/) — typed errors, timeouts, and build/transport gotchas.
- [Compatibility](compatibility/) — gateway protocol pin, wire contract, and toolchain floor.
- [Release process](release-process/) — versioning and how tags become releases.
- [Guides](guides/) — context propagation, FFI modes, error handling.
