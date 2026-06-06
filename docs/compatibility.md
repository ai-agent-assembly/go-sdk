---
title: Compatibility
weight: 6
---

# Core-runtime compatibility

`go-sdk` is a **client** of the AI Agent Assembly gateway (the core runtime in
[AI-agent-assembly/agent-assembly](https://github.com/AI-agent-assembly/agent-assembly)).
It is built against a specific gateway **protocol version** and a minimum Go
toolchain.

## Protocol version

The [`VERSION`](https://github.com/AI-agent-assembly/go-sdk/blob/master/VERSION)
file pins the gateway protocol version this SDK speaks (currently `0.0.0`). It
is the wire-compatibility contract, separate from the module's release tags:

- A released tag such as `v0.0.1-alpha.3` is the **SDK version** you `go get`.
- `VERSION` is the **gateway protocol** that SDK build targets.

When the gateway proto changes, `VERSION` is bumped, the vendored conformance
vectors are updated, and `make test` confirms wire compatibility before the
change ships.

## Wire contract

- **Transports** — the SDK talks to the gateway over **gRPC** and **HTTP**.
- **Enforcement modes** — `EnforcementMode` mirrors `aa_core::EnforcementMode`;
  the `enforce` / `observe` / `disabled` tokens are sent verbatim on the wire.
  See [Configuration](configuration/) for the per-agent posture.
- **Identity propagation** — agent / trace / run IDs flow to the gateway on
  every `Check` and `RecordResult`; the trace ID falls back to the active
  OpenTelemetry span context when not set explicitly.

## Toolchain

| Requirement | Value |
| --- | --- |
| Minimum Go | **1.26** (the `go` directive in `go.mod`) |
| CGo | Not required — pure-Go default builds with `CGO_ENABLED=0`. The native FFI transport is opt-in via `-tags aa_ffi_go`. |

## Release state

The SDK is **pre-release** on the `v0.0.1-alpha` line. The public `assembly`
package API may change between alpha tags; pin an exact tag and review the
[release notes](https://github.com/AI-agent-assembly/go-sdk/releases) before
upgrading. See [Release process](release-process/) for how versions are cut.
