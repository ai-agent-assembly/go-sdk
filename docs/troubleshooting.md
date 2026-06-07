---
title: Troubleshooting
weight: 5
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
| `ErrInvalidGateway` from `Init` | `WithGatewayURL` not set (or empty) | Pass `assembly.WithGatewayURL("https://…")`. |
| `ErrInvalidAPIKey` from `Init` | `WithAPIKey` not set (or empty) | Pass `assembly.WithAPIKey("…")` with an operator-issued key. |
| `ErrRuntimeNotInitialized` | Using the runtime before a successful `Init`, or after `Close` | Check the `Init` error before use; don't reuse a closed `*Assembly`. |

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
[Release process](release-process/)) and pkg.go.dev picks it up within minutes.
Preview the working tree locally with `godoc -http=:6060`.

## Still stuck?

Open an issue at
[github.com/ai-agent-assembly/go-sdk/issues](https://github.com/ai-agent-assembly/go-sdk/issues).
For suspected security issues, use the
[security policy](https://github.com/ai-agent-assembly/go-sdk/security/policy)
instead of a public issue.
