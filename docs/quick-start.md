---
title: Getting started
weight: 1
---

# Getting started

This walk-through takes you from zero to a governed tool call in three steps:
install the SDK, initialise the runtime, and wrap your tools so every call is
checked against the AI Agent Assembly gateway.

## Prerequisites

- **Go** ≥ 1.26 (matches the floor declared in `go.mod`)
- A gateway URL and API key for production, or nothing at all for local
  development — `Init` auto-discovers a gateway on `http://localhost:7391`
  when no URL is supplied. See [Configuration](../configuration/) for the
  full resolution order.
- *(Optional)* a C compiler if you want to enable the native FFI transport
  with `-tags aa_ffi_go`. The default transport is pure-Go and needs none.

## Install

```bash
go get github.com/ai-agent-assembly/go-sdk
```

## Initialise the runtime

`Init` returns an `*assembly.Assembly` — the handle you pass to the tool
wrappers below. Always `Close` it when you are done.

```go
package main

import (
    "context"
    "log"

    "github.com/ai-agent-assembly/go-sdk/assembly"
)

func main() {
    ctx := assembly.WithAgentID(context.Background(), "my-agent")

    a, err := assembly.Init(ctx,
        assembly.WithGatewayURL("https://gateway.example.com"),
        assembly.WithAPIKey("..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer a.Close()
}
```

`WithAgentID` stamps the calling agent's identity onto `ctx`; the SDK
forwards it to the gateway on every check and record. See
[Configuration](../configuration/#per-call-identity-context-helpers) for the
other per-call identifiers.

## Wrap your tools

`WrapTools` takes your tool slice and the `*Assembly` from `Init`, and returns
a slice of governed tools:

```go
governed := assembly.WrapTools(myTools, a)
```

Each call against a tool in `governed` is checked against the gateway policy
before execution and recorded after. Hand `governed` to your agent in place of
the originals.

## Where to next

- [Configuration](../configuration/) — every `Init` option, defaults, and enforcement modes
- [Architecture](../architecture/) — what's actually happening inside the SDK
- [API reference](../api-reference/) — godoc on pkg.go.dev (when indexing is unblocked)
- [Troubleshooting](../troubleshooting/) — what to do when `Init` or a check fails
- [Guides](../guides/) — deeper how-tos
