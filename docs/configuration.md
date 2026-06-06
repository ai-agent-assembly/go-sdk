---
title: Configuration
weight: 4
---

# Configuration

`assembly.Init` takes a `context.Context` and a variadic list of functional
options. New configuration is always added as a `WithXxx(value)` option, so
call sites never break when an option is introduced.

```go
a, err := assembly.Init(ctx,
    assembly.WithGatewayURL("https://gateway.example.com"),
    assembly.WithAPIKey("..."),
    assembly.WithFailClosed(true),
    assembly.WithTimeout(750*time.Millisecond),
    assembly.WithEnforcementMode(assembly.EnforcementModeObserve),
)
```

## Required options

| Option | Type | If missing |
| --- | --- | --- |
| `WithGatewayURL` | `string` | `Init` returns `ErrInvalidGateway` |
| `WithAPIKey` | `string` | `Init` returns `ErrInvalidAPIKey` |

## Optional options

| Option | Type | Default | Purpose |
| --- | --- | --- | --- |
| `WithFailClosed` | `bool` | `false` | When `true`, a gateway failure **blocks** the action (fail-closed). When `false`, the action is allowed if the gateway is unreachable (fail-open). |
| `WithTimeout` | `time.Duration` | `500ms` | Gateway check timeout applied when the call `ctx` carries no deadline. |
| `WithEnforcementMode` | `EnforcementMode` | `enforce` | Per-agent governance posture sent to the gateway at registration. |
| `WithSelfAgentID` | `string` | _(unset)_ | Records this agent's own ID for lineage tracking. |
| `WithParentAgentID` | `string` | _(unset)_ | Parent agent ID for topology tracking. |
| `WithTeamID` | `string` | _(unset)_ | Team ID for budget and policy scoping. |
| `WithDelegationReason` | `string` | _(unset)_ | Human-readable reason this agent was delegated work. |
| `WithSpawnedByTool` | `string` | _(unset)_ | Name of the tool that spawned this agent. |
| `WithSidecarBinary` | `string` | _(unset)_ | Path to a sidecar binary for managed-lifecycle (sidecar) mode. |

## Enforcement modes

`WithEnforcementMode` accepts the values mirrored from `aa_core::EnforcementMode`
on the wire:

| Constant | Token | Behavior |
| --- | --- | --- |
| `EnforcementModeEnforce` | `enforce` | Default. A `deny` blocks the action; `redact` strips secrets. |
| `EnforcementModeObserve` | `observe` | Dry-run. The gateway records what *would* have happened but does not block. |
| `EnforcementModeDisabled` | `disabled` | Policy evaluation is skipped entirely. |

## Per-call identity (context helpers)

Identity that varies per request is carried on `context.Context`, not on
`Init`. The SDK forwards these to the gateway on every `Check` and
`RecordResult`:

| Helper | Reader | Notes |
| --- | --- | --- |
| `WithAgentID` | `AgentIDFromContext` | The calling agent's identity. |
| `WithTraceID` | `TraceIDFromContext` | Falls back to the OpenTelemetry span-context trace ID when unset. |
| `WithRunID` | `RunIDFromContext` | Run identity for a single agent run. |
| `EnsureRunID` | — | Guarantees a stable run ID within the same context tree. |

```go
ctx = assembly.WithAgentID(ctx, "my-agent")
ctx = assembly.EnsureRunID(ctx)
```

## Where to next

- [Getting Started](getting-started/) — install, init, wrap your tools.
- [Troubleshooting](troubleshooting/) — what each configuration error means.
- [Architecture](architecture/) — how options flow into the runtime.
