---
title: Getting Started
weight: 1
---

# Getting Started

A short walk-through of installing the SDK, initialising it, and confirming
that a tool call is reaching the AI Agent Assembly gateway.

## Prerequisites

- **Go** ≥ 1.24 (matches the floor declared in `go.mod`)
- An AAASM **gateway URL** and **API key** (see your operator)
- *(Optional)* a C compiler if you want to enable the native FFI transport
  with `-tags aa_ffi_go`. The default transport is pure-Go and needs none.

## Install

```bash
go get github.com/AI-agent-assembly/go-sdk
```

> **Note** — `go get` and pkg.go.dev indexing are blocked today pending a
> module-path rename ticket (the `go.mod` declares
> `github.com/agent-assembly/go-sdk`, but the canonical GitHub URL is
> `github.com/AI-agent-assembly/go-sdk`). Until that ticket lands, clone
> the repo and use a `replace` directive in your consumer.

## Initialise

```go
ctx, err := assembly.Init(context.Background(),
    assembly.WithGatewayURL("https://gateway.example.com"),
    assembly.WithAPIKey("..."),
    assembly.WithFailClosed(false),
)
if err != nil {
    log.Fatal(err)
}
defer ctx.Close()
```

## Wrap your tools

```go
wrapped := assembly.WrapTools(myTools, ctx)
```

Each call against `wrapped` is checked against the gateway policy before
execution and recorded after.

## Where to next

- [Architecture](../architecture/) — what's actually happening inside the SDK
- [API reference](../api-reference/) — godoc on pkg.go.dev (when indexing is unblocked)
- [Guides](../guides/) — deeper how-tos
