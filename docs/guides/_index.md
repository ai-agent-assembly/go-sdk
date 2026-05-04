---
title: Guides
weight: 3
---

# Guides

Topical how-tos and deeper explanations. Pages land here as Subtasks
under [AAASM-316](https://lightning-dust-mite.atlassian.net/browse/AAASM-316)
and follow-up Stories complete.

## Planned

- **Context propagation** — `AgentID`, `TraceID`, `RunID` flow; OpenTelemetry
  span context fallback; `EnsureRunID` for stable run IDs.
- **FFI transport modes** — choosing between the pure-Go UDS fallback and
  the CGo native bridge; build-tag selection; `CGO_ENABLED=0` support.
- **Error handling** — sentinel errors, `errors.Is` / `errors.As`, wrapping
  with `%w`.
- **Sidecar mode** — running with a local proxy instead of direct gateway
  calls.

If a topic you need isn't here yet, open an issue against the
[go-sdk repo](https://github.com/AI-agent-assembly/go-sdk/issues).
