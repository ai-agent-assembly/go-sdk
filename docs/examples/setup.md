---
title: Preparing the runtime environment
weight: 1
---

# Preparing the runtime environment

Everything you need to run **any** Go example in this section. Do this once; the
per-example pages then only cover what's specific to them.

## What you need

| Requirement | Why |
|---|---|
| **Go ≥ 1.26** | The floor declared in each example's `go.mod`. |
| **Git** | To clone the examples repository. |
| **Network access** (first run only) | `go mod download` fetches the SDK and dependencies from `proxy.golang.org`. |

> **No gateway required.** Every example in this section ships with an in-process
> mock `GovernanceClient`, so none of them need a live AI Agent Assembly gateway
> or an API key to run. The [CLI runtime integration]({{< relref "/examples/cli-runtime-integration" >}})
> example can *optionally* talk to a local `aasm` sidecar, but it falls back to
> the offline mock when `aasm` isn't installed.

## Install the SDK (for your own code)

When you write your own agent, add the SDK to your module:

```bash
go get github.com/ai-agent-assembly/go-sdk
```

The examples already declare this dependency in their own `go.mod`, so you don't
need to run `go get` to use them — `go mod download` (below) pulls it in.

## Clone the examples repository

All Go examples live under `go/` in the central examples repo:

```bash
git clone https://github.com/ai-agent-assembly/examples.git
cd agent-assembly-examples/go
```

Each subdirectory (`basic-agent`, `tool-policy`, `langchaingo`,
`cli-runtime-integration`) is its own Go module.

## Run an example

`cd` into the example you want, download its dependencies, then run it. The
pattern is the same for all four:

```bash
cd basic-agent        # or tool-policy, langchaingo, cli-runtime-integration
go mod download
go run .
```

Run an example's tests — all of them run **offline**, no gateway needed:

```bash
go test ./...
```

> Always run `go run .` from **inside** the example directory, not from the repo
> root. Running from the wrong directory is the most common reason an example
> behaves unexpectedly.

## Local / unauthenticated dev mode

These examples don't connect to a gateway, so there are no gateway or API-key
environment variables to set for them. The only environment knobs anywhere in
this section belong to the [CLI runtime integration]({{< relref "/examples/cli-runtime-integration" >}})
example's helper script, which honours:

| Variable | Default | Effect |
|---|---|---|
| `AASM_PORT` | `7878` | Port the `aasm serve` sidecar listens on. |
| `WAIT_SECONDS` | `5` | How long the script waits for the sidecar to become healthy. |

When you graduate to a **real gateway** in your own code, the SDK's `Init` reads
its gateway URL and API key from options, the `AAASM_GATEWAY_URL` /
`AAASM_API_KEY` environment variables, or a config file — and defaults to
`http://localhost:7391` for local development. See
[Quick Start]({{< relref "/quick-start" >}}) and
[Configuration]({{< relref "/configuration" >}}) for that path.

### Optional: install the `aasm` CLI

Only the [CLI runtime integration]({{< relref "/examples/cli-runtime-integration" >}})
example uses the `aasm` binary, and only for its full-sidecar path. Install it if
you want to exercise that path:

{{< tabs >}}
{{< tab name="Homebrew" >}}

```bash
brew install ai-agent-assembly/tap/aasm
```

{{< /tab >}}
{{< tab name="curl installer" >}}

```bash
curl -fsSL https://agent-assembly.com/install.sh | sh
```

{{< /tab >}}
{{< tab name="go install" >}}

```bash
go install github.com/ai-agent-assembly/agent-assembly/cmd/aasm@latest
```

{{< /tab >}}
{{< /tabs >}}

If `aasm` is not installed, that example detects it and continues in offline
fallback mode.

## Where to next

- [Basic agent]({{< relref "/examples/basic-agent" >}}) — the smallest governed agent.
- [Examples overview]({{< relref "/examples" >}}) — the full list of examples.
- [Quick Start]({{< relref "/quick-start" >}}) — install, init, and wrap in three steps.
