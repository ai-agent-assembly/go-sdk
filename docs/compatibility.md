---
title: Compatibility & Versioning
weight: 6
---

# Compatibility & Versioning

`go-sdk` is a **client** of the AI Agent Assembly gateway (the core runtime in
[ai-agent-assembly/agent-assembly](https://github.com/ai-agent-assembly/agent-assembly)).
It is built against a specific gateway **protocol version** and a minimum Go
toolchain, and it ships on its own release cadence. This page covers both the
compatibility contract and how releases are cut.

## Protocol version

The [`VERSION`](https://github.com/ai-agent-assembly/go-sdk/blob/master/VERSION)
file pins the gateway protocol version this SDK speaks (currently `0.0.0`). It
is the wire-compatibility contract, separate from the module's release tags:

- A released tag such as `v0.0.1-alpha.3` is the **SDK version** you `go get`.
- `VERSION` is the **gateway protocol** the SDK build targets.

When the gateway proto changes, `VERSION` is bumped, the vendored conformance
vectors are updated, and `make test` confirms wire compatibility before the
change ships.

> **Cross-SDK matrix.** The core↔SDK compatibility matrix — which SDK versions
> speak which gateway protocol, across the Go, Python, and Node SDKs — is
> published on the shared
> [documentation hub](https://ai-agent-assembly.github.io/agent-assembly-docs/).
> Consult it when pairing an SDK release with a gateway deployment.

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
| CGo | Not required — the pure-Go default builds with `CGO_ENABLED=0`. The native FFI transport is opt-in via `-tags aa_ffi_go`. |

## Release state

The SDK is **pre-release** on the `v0.0.1-alpha` line. The public `assembly`
package API may change between alpha tags; pin an exact tag and review the
[release notes](https://github.com/ai-agent-assembly/go-sdk/releases) before
upgrading.

## Release process

`go-sdk` follows the standard Go module release model: **a release is a git
tag**. There is no separate publish step to `go get` a tagged version — the Go
module proxy and [pkg.go.dev](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk)
index the tag automatically.

### Versioning

- **Semantic Versioning** (`vMAJOR.MINOR.PATCH`), with pre-release suffixes for
  the current alpha line (`v0.0.1-alpha.N`).
- The module path is stable — `github.com/ai-agent-assembly/go-sdk` — so a major
  bump past `v1` would add a `/vN` suffix per Go's import-compatibility rule.
- `VERSION` is **not** the release version; it pins the gateway protocol the SDK
  targets (see [Protocol version](#protocol-version) above).

### Cutting a release

1. Land all changes on `master` with CI green.
2. If the gateway protocol changed, bump
   [`VERSION`](https://github.com/ai-agent-assembly/go-sdk/blob/master/VERSION)
   and update the conformance vectors; confirm with `make test`.
3. Tag the commit and push the tag:

   ```bash
   git tag v0.0.1-alpha.4
   git push remote v0.0.1-alpha.4
   ```

4. Publish the **release notes** on the
   [GitHub Releases](https://github.com/ai-agent-assembly/go-sdk/releases) page
   for that tag.

### What happens after the tag

- `go get github.com/ai-agent-assembly/go-sdk@v0.0.1-alpha.4` resolves through
  the module proxy immediately.
- pkg.go.dev renders the godoc for the tag within minutes.
- The `goreleaser` config (`.goreleaser.yaml`) is validated on every push to
  `master` by the `goreleaser-check` job; it is set up for source archives, with
  binary builds skipped (this is a library, not a CLI).

### Consuming a pinned version

```go
require github.com/ai-agent-assembly/go-sdk v0.0.1-alpha.3
```

Pin an exact tag while the SDK is pre-release; review the release notes before
moving to a newer alpha.
