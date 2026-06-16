# CLAUDE.md — go-sdk

Guidance for Claude Code (and humans) working in this repository. This file holds
**repo-specific** context only; universal engineering policy lives in the global
config. When a fact here duplicates `CONTRIBUTING.md`, the `Makefile`, `README.md`,
or `go.mod`, treat those as the source of truth and update them, not just this file.

## What this repo is

The **Go SDK** for AI Agent Assembly — runtime governance for AI agent tool calls.
Module path `github.com/ai-agent-assembly/go-sdk`. The public surface is the
`assembly/` package: it initialises in a few lines, propagates agent identity
through `context.Context`, wraps a tool slice with policy enforcement, and forwards
every check + result to the AAASM gateway over gRPC or HTTP. Anything outside
`assembly/` (`internal/`, `examples/`) is not public API and may change without notice.

This SDK is the **SDK layer** of the product's three-layer interception model — the
fastest, in-process path (the sidecar proxy and eBPF layers live in the core
monorepo and catch what the SDK misses). The SDK reaches the runtime through a thin
**cgo FFI** binding (`internal/ffi/cgo_bridge.go`, `-tags aa_ffi_go`) over the
vendored `native/aa-ffi-go` crate — a thin C-ABI shim (cbindgen header) that
delegates to the SHA-pinned **`aa-sdk-client`** in the
[`ai-agent-assembly/agent-assembly`](https://github.com/ai-agent-assembly/agent-assembly)
monorepo. That monorepo is the source of truth for the protocol, policy semantics,
and the `aa-*` crates this SDK pins.

### Fail-closed FFI

The default build is a **pure-Go UDS fallback** that runs cleanly with
`CGO_ENABLED=0`. It is **fail-closed**: with no native binding and no reachable
runtime it returns `ErrRuntimeUnavailable`, never a silent allow. The native cgo
transport is opt-in (`-tags aa_ffi_go` + `CGO_ENABLED=1`) and requires a C compiler,
a Rust toolchain, and `protoc` to build the vendored shim (`make native`).

## Build, test, lint

See `CONTRIBUTING.md`, `README.md`, and the `Makefile` for the full list. Common
commands:

```bash
make fmt                                      # gofmt -w on all .go files
make lint                                     # golangci-lint run ./...
make test                                     # go test ./...
go vet ./...                                  # static analysis
go test ./assembly                            # one package
go test ./assembly -run TestRegisterAgent     # one test
go test -count=1 -race ./...                  # race detector (touching concurrency)
```

- **Native FFI (opt-in):** `make native` builds `native/aa-ffi-go`; `make test-native`
  then runs the suite against the real cgo binding (`-tags aa_ffi_go`, `CGO_ENABLED=1`).
  Needs a Rust toolchain + `protoc` on `PATH`.
- **Proto regen:** `make proto` regenerates Go stubs from the sibling
  `agent-assembly/proto` checkout (`$AA_PROTO_DIR`, defaults `../agent-assembly/proto`);
  needs `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`.
- **Memory harness (opt-in, slow):**
  `AAASM_MEMORY_HARNESS=1 go test ./internal/ffi -run TestMemoryRegressionHarness`.

## Idiomatic Go conventions (see `CONTRIBUTING.md` — don't duplicate)

These are load-bearing; reviewers push back on PRs that skip them.

- **Context first** — every public function takes `context.Context` as its first
  argument. Don't add positional args before `ctx`; don't call `context.Background()`
  in library code (only the `Assembly` runtime owns a lifecycle scope).
- **Wrap errors with `%w`** — `fmt.Errorf("op: %w", err)` so `errors.Is`/`errors.As`
  work. Sentinels live in `assembly/governance_errors.go`; prefer the exported
  `PolicyViolationError` over ad-hoc strings.
- **Functional options** — config knobs land as `WithXxx(value)` in
  `assembly/options.go`, never as new positional args to `Init`.
- **`io.Closer` for cleanup** — long-lived resources implement `Close`, paired with a
  `defer x.Close()`. Don't invent ad-hoc `Shutdown`/`Stop`.
- **`internal/` is internal** — needing to import `internal/ffi/` from outside the
  module is a design break; promote the API to `assembly/` instead.

## Conventions (see `CONTRIBUTING.md`)

- **Commits:** `<emoji> (<scope>): <imperative summary>` (gitmoji). One logical unit
  per commit; bisectable. Utils/mocks/tests are separate preceding commits.
- **Branch:** `<release-or-phase>/<ticket>/<type>/<short_summary>`
  (e.g. `v0.0.1/AAASM-1143/feat/readme_contributing`).
- **PR title:** `[<ticket>] <emoji> (<scope>): <summary>`; base branch **always
  `master`**; body fills in `.github/PULL_REQUEST_TEMPLATE.md` (Ticket, Summary,
  Change Scope, Validation, Rollout Notes); ≥1 Pioneer-team approval.

## Repo-specific gotchas

- **`origin` is a personal fork; the canonical remote is `remote`.** In this checkout
  `origin` → `Chisanan232/go-sdk` (a fork) and `remote` → `ai-agent-assembly/go-sdk`
  (canonical), which is usually **ahead** of the fork. Always scope work against
  `remote/master`, and **push to `remote`, not `origin`**. The fork's `go.mod` may
  even show the wrong (uppercase) module path — the canonical path is lowercase
  `github.com/ai-agent-assembly/go-sdk`. A "repository moved" redirect notice on push
  is harmless.
- **Never `--no-verify`; never force-push.** A docs-only / `.go`-free change normally
  skips the Go hooks; if a hook fails in a fresh worktree, fix the cause.
- **CGo build matrix:** if you touch `internal/ffi/`, validate both paths — the native
  binding (`go test -tags aa_ffi_go ./...`) **and** the pure-Go fallback
  (`CGO_ENABLED=0 go test ./...`). CI runs Go 1.24/1.25 × ubuntu/macOS × `CGO_ENABLED`
  0/1.
- **Org GitHub Actions can be billing-blocked** — jobs may abort in seconds with a
  payments message. Check run **annotations** before triaging as a code bug; **validate
  locally** rather than waiting on CI.

## Project policy

- **JIRA:** project AAASM; set **Component** (`customfield_10041`) to this repo
  (`AI-agent-assembly/go-sdk`); Team (`customfield_10001`) = Pioneer.
  Epic → Story → Subtask (one Subtask ≈ one commit) + a `Verify …` subtask per Story.
- **Self-hosted deployment is out of scope** product-wide — don't propose
  Helm/Terraform/air-gapped/migration work even if the spec mentions it.
- **The Protocol Specification stays in the `agent-assembly` monorepo** — this SDK
  only pins the wire protocol; don't put spec work here or in a separate spec repo.

## Documentation conventions — document the WHY, not the WHAT

Doc comments exist to capture intent the code cannot: rationale, constraints,
invariants, and non-obvious decisions. Restating what the signature already says is
noise that rots out of sync — avoid it.

- **Package doc (`package assembly` + `doc.go`):** yes — the package's role in the
  three-layer model, the fail-closed contract, and the public-API boundary.
- **Exported identifiers (`//` doc-comments on `Pub` fns/types/vars):** yes, and per
  godoc convention **start the comment with the identifier name**
  (`// WrapTools returns …`). State the contract: behavior, errors returned, units,
  side effects, and any context/goroutine/cgo/ordering constraints — especially the
  surprising ones (e.g. "fails fast when `ctx` is already cancelled", "applies the
  500ms default when `ctx` has no deadline", "fail-closed when the runtime is
  unavailable").
- **Inline `//` why-comments:** for workarounds, the cgo bridge `#cgo` directives,
  build-tag selection, security rationale, and the `aa-sdk-client` SHA pin (say *why*
  it is pinned, not just that it is).
- **Skip:** unexported trivial helpers, getters, type-restating, and anything a reader
  infers from the signature. No per-field doc comments.
- **Big architectural decisions → ADRs**, not scattered doc-comments; link code to the
  ADR. Concept docs already live under `docs/` — reference them.

> Net: a new contributor (human or LLM) should read a package's doc comment and an
> exported item's `//` comment and understand *why it is the way it is* without
> reverse-engineering it. If a comment only says *what*, delete it.
