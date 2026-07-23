---
title: Troubleshooting
weight: 8
---

# Troubleshooting

Every error the SDK returns is a typed value. Match on the sentinels with
`errors.Is` (or `errors.As` for structured types) rather than string
comparison — wrapping with `%w` preserves the chain.

```go
a, err := assembly.Init(ctx, opts...)
if errors.Is(err, assembly.ErrInvalidGateway) {
    // missing or empty gateway URL
}
```

## Initialisation errors

| Symptom | Cause | Fix |
| --- | --- | --- |
| `ErrInvalidGateway` from `Init` | No gateway URL from any source — the option, `AA_GATEWAY_URL`, the config file, and the local default all came back empty (the local auto-start also failed). | Pass `assembly.WithGatewayURL("https://…")`, or make sure a local gateway is reachable on `http://localhost:7391`. See the [resolution chain]({{< relref "/configuration#gateway-and-credential-resolution" >}}). |
| `*ConfigurationError` from `Init` | The SDK needed the local default but couldn't bring it up — typically `aasm` is missing from `PATH`. | Install the `aasm` CLI (the error message includes the `go install …` hint), or point at an existing gateway with `WithGatewayURL`. |
| `*GatewayError` from `Init` | A gateway URL is known, but the auto-started gateway didn't answer `/healthz` within the timeout window. | Check the gateway is healthy and reachable; raise the start window or start it yourself before `Init`. |
| `ErrRuntimeNotInitialized` | Using the runtime before a successful `Init`, or after `Close` | Check the `Init` error before use; don't reuse a closed `*Assembly`. |

An empty API key is **not** an error — local mode accepts unauthenticated
agents. Set `WithAPIKey` (or `AA_API_KEY`) only when your gateway requires
authentication.

> **Note:** `AAASM_GATEWAY_URL` / `AAASM_API_KEY` are accepted as deprecated
> aliases for backward compatibility and emit a one-time deprecation warning
> at runtime; use the canonical `AA_*` names in new configurations. (See
> `assembly/gateway_resolver.go`.)

## Sidecar mode

| Symptom | Cause | Fix |
| --- | --- | --- |
| `ErrSidecarUnavailable` | Sidecar mode selected but no reachable sidecar | Confirm the sidecar process is running and the configured address/binary is correct. |
| `ErrBinaryNotFound` | `WithSidecarBinary` path (or the `aasm` binary) does not exist | Install the binary or correct the path; see the install hint in the error message. |

## Timeouts

- The default gateway check timeout is **500 ms**, applied only when the call
  `ctx` carries **no** deadline. Raise it with `assembly.WithTimeout(...)`.
- If the call `ctx` is already cancelled, `Check` **fails fast** and does not
  contact the gateway — this is expected, not a bug.
- A request `ctx` deadline always wins over the SDK default.

## Build & transport

- **Pure-Go is the default.** The SDK builds and runs with `CGO_ENABLED=0`; no
  C compiler is required.
- The native FFI transport is **opt-in**: build with `-tags aa_ffi_go` and
  `CGO_ENABLED=1`, which needs a C compiler. Without the tag, the pure-Go UDS
  fallback is selected automatically.
- `undefined: aa_ffi…` / linker errors usually mean the `aa_ffi_go` build tag
  was set without `CGO_ENABLED=1` or without the native library available.

## API reference not showing on pkg.go.dev

The reference on [pkg.go.dev](https://pkg.go.dev/github.com/ai-agent-assembly/go-sdk)
is generated per released tag. If a symbol is missing, it is almost always
because no tag has been pushed for that change yet — push a `vX.Y.Z` tag (see
[Compatibility & Versioning]({{< relref "/compatibility#release-process" >}})) and pkg.go.dev
picks it up within minutes.
Preview the working tree locally with `godoc -http=:6060`.

## Still stuck?

Open an issue at
[github.com/ai-agent-assembly/go-sdk/issues](https://github.com/ai-agent-assembly/go-sdk/issues).
For suspected security issues, use the
[security policy](https://github.com/ai-agent-assembly/go-sdk/security/policy)
instead of a public issue.
