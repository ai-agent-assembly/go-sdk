# AAASM-2704 — Verification: vendor thin aa-ffi-go shim + real enforcement + fail-closed

Story **AAASM-2704** (Epic AAASM-2552). Subtasks **2709** (vendor crate), **2710**
(cgo bridge + binding interface), **2711** (fail-closed fallback), **2712** (build
glue + CI). This is **AAASM-2713**.

Aligns go-sdk with node/python: the FFI shim now lives **in go-sdk** (`native/aa-ffi-go`,
SHA-pinned `aa-sdk-client`) instead of consuming a monorepo staticlib artifact
(removed by the sibling AAASM-2703), and the no-transport default is **fail-closed**.

## How verified

| # | Method |
|---|--------|
| 1 | `cargo build --manifest-path native/aa-ffi-go/Cargo.toml` → `libaa_ffi_go.{a,dylib}`; `cargo test` (in crate) 6 pass; clippy `-D warnings` 0; cbindgen header in sync (`diff` clean) |
| 2 | `CGO_ENABLED=1 go build -tags aa_ffi_go ./internal/ffi/` links the vendored lib; `TestCgoBridgeConnectDisconnect` round-trips through the real binding (connect→FFI→`aa-sdk-client`→disconnect); `send_event` without a live runtime returns `ffi channel closed` (status 6 → `ErrChannelClosed`) — proves marshalling + status mapping |
| 3 | `make test-native` (build shim + `CGO_ENABLED=1 go test -tags aa_ffi_go ./...`) → all green; `go test ./...` (default fail-closed) → all green |
| 4 | `gofmt -l` clean; `go vet ./...` clean; `grep` confirms 0 `aa_query_policy`/`queryPolicy` refs and 0 `{"allow":true}` literals in non-test ffi |

## Acceptance criteria

| AC | Result | Evidence |
|----|--------|----------|
| `native/aa-ffi-go/` thin shim; `Cargo.toml` pins `aa-sdk-client` by git-SHA; builds to a linkable lib; cbindgen header committed | ✅ Pass | crate present (mirrors `node-sdk/native/aa-ffi-node`), `rev = "9cf8a033…"`, builds `.a`+`.dylib`, `include/aa_ffi_go.h` committed + in sync, `Cargo.lock` committed |
| cgo path links the vendored shim + routes register/send through the runtime; no `aa_query_policy` | ✅ Pass | `cgo_bridge.go` `#include`s the cbindgen header + links `native/aa-ffi-go/target/debug`; `aa_send_event(event_type, details)`; `aa_query_policy` removed across the whole `binding` interface |
| Default / no-transport **fail-closed** (`ErrRuntimeUnavailable`); no `{"allow":true}`; error propagates through `boot()`/register/check | ✅ Pass | `fallbackUDSBridge` returns `statusRuntimeUnavailable` on every op; old `{"allow":true,"reason":"fallback-uds"}` no-op deleted; `boot()` falls through to the real `sidecarConnector` (no fake success) |
| CI builds the shim + runs `-tags aa_ffi_go` tests | ✅ Pass | `.github/workflows/native-ffi.yml` (Rust stable + protoc) runs `make test-native` — the only lane that compiles/links the native path |
| `fmt` / lint / `go test` green on both default (fail-closed) and cgo (real-transport) builds | ✅ Pass | `go test ./...` and `make test-native` both green; gofmt + go vet clean (golangci-lint runs in CI) |

## Notes

- **Interface change:** `binding.sendEvent(handle, eventType, details)` + `Client.SendEvent(eventType, details)`; `QueryPolicy` removed entirely (policy is server-side). `runtime.boot()` reports the registration event as `("register", <json>)`.
- One assembly test (`TestInit_ExplicitOptionsBypassResolver`) was made build-tag-agnostic by pinning a capturing FFI client via the `newFFIClient` seam (the topology tests' existing pattern) — under `-tags aa_ffi_go` the real binding has no live runtime, so transport-incidental tests must not depend on the binding being a silent no-op.
- Native ABI status codes 5/6/7 (IPC/channel/panic) are now mapped Go-side.
- `native-ffi.yml` is a standalone check; adding it to the `CI Success` aggregate + required checks is an owner branch-protection action.

## Outcome

All ACs **pass**. go-sdk is now self-contained and consistent with node/python:
one vendored thin shim per SDK over the SHA-pinned `aa-sdk-client`, with a
fail-closed default — completing the FFI consolidation (Epic AAASM-2552) for Go.
