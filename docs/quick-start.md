---
title: Quick Start
weight: 1
---

# Quick Start

This walkthrough takes you from zero to a governed tool call in three steps:
install the SDK, initialise the runtime, and wrap your tools so every call is
checked against the AI Agent Assembly gateway. The whole thing is a single
`main` you can copy, paste, and run.

## Prerequisites

- **Go** ≥ 1.26 (the floor declared in `go.mod`).
- For **local development**: nothing else — `Init` auto-discovers a gateway on
  `http://localhost:7391`, and starts one for you if none is running (it shells
  out to `aasm start --mode local --foreground`, so the
  [`aasm` CLI](https://github.com/ai-agent-assembly/agent-assembly) must be on
  your `PATH`).
- For **production**: a gateway URL and, if your gateway requires auth, an API
  key. Both can come from options, environment variables, or a config file —
  see [Configuration](../configuration/).
- *(Optional)* a C compiler, only if you opt into the native FFI transport with
  `-tags aa_ffi_go`. The default transport is pure-Go and needs none.

## Step 1 — Install

```bash
go get github.com/ai-agent-assembly/go-sdk
```

## Step 2 — Initialise the runtime

`Init` returns an `*assembly.Assembly` — your runtime handle. Always `Close` it
when you're done so the connection (and any managed sidecar) is released.

```go
package main

import (
    "context"
    "log"

    "github.com/ai-agent-assembly/go-sdk/assembly"
)

func main() {
    // Stamp this agent's identity onto the context. The SDK forwards it to
    // the gateway on every check and record.
    ctx := assembly.WithAgentID(context.Background(), "my-agent")

    a, err := assembly.Init(ctx,
        assembly.WithGatewayURL("https://gateway.example.com"),
        assembly.WithAPIKey("..."), // optional — omit for local, unauthenticated dev
    )
    if err != nil {
        log.Fatalf("init assembly runtime: %v", err)
    }
    defer func() {
        if err := a.Close(); err != nil {
            log.Printf("close assembly runtime: %v", err)
        }
    }()

    log.Println("assembly runtime ready")
}
```

For **local development** you can drop both options entirely — `assembly.Init(ctx)`
resolves the gateway from the environment, then `~/.aasm/config.yaml`, then the
local default. See [Configuration](../configuration/#gateway-and-credential-resolution)
for the full resolution order.

## Step 3 — Wrap your tools

Your tools just need to satisfy the SDK's small `Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Call(ctx context.Context, input string) (string, error)
}
```

`WrapTools` takes your `[]Tool` and a governance client, and returns a new
`[]Tool` where every `Call` is governed:

```go
governed := assembly.WrapTools(myTools, nil)
```

The second argument is the `GovernanceClient` that talks to the gateway.
Passing `nil` gives you a **passthrough wrapper** — the tools run, but no
`Check`/`RecordResult` calls are made. That's the simplest starting point and
what the package's own runnable example uses; wire in a real client when you're
ready to enforce policy (see
[Handle allow/deny decisions and errors](../guides/handle-decisions-and-errors/)).

Hand `governed` to your agent in place of the originals. From here on, each call
against a governed tool is checked against the gateway policy before execution
and recorded after.

## Putting it together

```go
package main

import (
    "context"
    "log"

    "github.com/ai-agent-assembly/go-sdk/assembly"
)

// echoTool is a minimal Tool implementation.
type echoTool struct{}

func (echoTool) Name() string       { return "echo" }
func (echoTool) Description() string { return "returns its input unchanged" }
func (echoTool) Call(_ context.Context, input string) (string, error) {
    return input, nil
}

func main() {
    ctx := assembly.WithAgentID(context.Background(), "my-agent")

    a, err := assembly.Init(ctx,
        assembly.WithGatewayURL("https://gateway.example.com"),
        assembly.WithAPIKey("..."),
    )
    if err != nil {
        log.Fatalf("init: %v", err)
    }
    defer a.Close()

    tools := []assembly.Tool{echoTool{}}
    governed := assembly.WrapTools(tools, nil)

    out, err := governed[0].Call(ctx, "hello, governance")
    if err != nil {
        log.Fatalf("tool call: %v", err)
    }
    log.Println("result:", out) // result: hello, governance
}
```

### What to expect

- **`Init` succeeds** once a gateway is reachable (resolved or auto-started). If
  no gateway can be found and no `aasm` binary is on `PATH`, you'll get a typed
  `*assembly.ConfigurationError` — see [Troubleshooting](../troubleshooting/).
- **Tool calls run** and return the inner tool's result. With a real governance
  client wired in, a `deny` decision surfaces as a
  `*assembly.PolicyViolationError` and the inner tool never runs.

## Where to next

- [Core Concepts](../core-concepts/) — what's actually happening inside the SDK.
- [Guides](../guides/) — wrap a real agent, integrate a framework, handle decisions.
- [Configuration](../configuration/) — every `Init` option, defaults, and enforcement modes.
- [Troubleshooting](../troubleshooting/) — what to do when `Init` or a check fails.
