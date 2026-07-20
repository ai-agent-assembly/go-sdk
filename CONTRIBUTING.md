# Contributing to go-sdk

Thanks for taking the time to contribute. This guide describes how to set up
a development environment, run the test suite, and submit a pull request.

## Development Environment

### Required

- **Go ≥ 1.26** — matches the floor declared in `go.mod`; install via the official distribution or `brew install go`.
- **golangci-lint** — `brew install golangci-lint` (or see the
  [project's install guide](https://golangci-lint.run/welcome/install/)).
- **make** — for the `Makefile` targets (`fmt`, `lint`, `test`).

### Optional

- **A C compiler + Rust toolchain (+ protoc)** — required only when building or
  testing the native FFI transport (`-tags aa_ffi_go`). The default build uses a
  **fail-closed** pure-Go fallback (no native binding → `ErrRuntimeUnavailable`,
  never a silent allow). The native binding is the vendored `native/aa-ffi-go`
  crate — a thin C-ABI shim over the SHA-pinned `aa-sdk-client`. Build it with
  `make native`; the cgo bridge links it via `native/aa-ffi-go/{include,target/debug}`
  (see `internal/ffi/cgo_bridge.go`).
- **Hugo (extended) ≥ 0.146** — required only if you want to preview the docs
  site locally (`cd website && hugo server`).

### First-time setup

```bash
git clone https://github.com/ai-agent-assembly/go-sdk.git
cd go-sdk
go mod download
make test
```

If the test run is green, your environment is ready.

## Running Tests and Checks

```bash
# Full validation, equivalent to what CI runs.
make fmt
make lint
make test

# Static analysis (also covered by 'make lint' but useful in isolation).
go vet ./...

# A single package or a single test (regex match).
go test ./assembly
go test ./assembly -run TestRegisterAgent

# Race detector — required for any change that touches concurrency.
go test -count=1 -race ./...

# Native FFI build (opt-in; builds native/aa-ffi-go then runs the real cgo binding).
make test-native

# Memory regression harness (1M sends; opt-in, takes a few minutes).
AAASM_MEMORY_HARNESS=1 go test ./internal/ffi -run TestMemoryRegressionHarness
```

CI runs the full matrix across Go 1.26.x, ubuntu / macOS, and `CGO_ENABLED` 0 / 1. If your local run passes, CI almost certainly will.

## Idiomatic Go Conventions

These conventions are load-bearing in this codebase. Reviewers will push back
on PRs that skip them.

### Context first

Every public function takes `context.Context` as its first argument. Don't add
positional arguments before `ctx`. Don't call `context.Background()` inside
library code unless you are explicitly creating a new lifecycle scope (the
`Assembly` runtime is the only place that does this today).

### Errors are wrapped with `%w`

Use `fmt.Errorf("operation: %w", err)` so callers can recover the chain via
`errors.Is` / `errors.As`. Sentinel errors live in
`assembly/governance_errors.go` and the structured `PolicyViolationError`
type is already exported — prefer those over ad-hoc string messages.

### Functional options

`Init` and any future config-heavy entry point takes `(ctx, opts ...Option)`.
New configuration knobs land as `WithXxx(value)` functions in
`assembly/options.go`, never as new positional arguments to `Init`.

### `io.Closer` for cleanup

Anything that holds a long-lived resource (gateway connection, sidecar
process, file handle) implements `io.Closer` and is paired with a `defer
x.Close()` at the call site. Don't invent ad-hoc `Shutdown` / `Stop`
methods when `Close` already covers it.

### `internal/` is internal

`internal/ffi/` is for low-level shims that are not part of the public API.
If you find yourself needing to import from `internal/` outside the module,
that's a design break — promote the API to `assembly/` rather than reaching
across the boundary.

## Shared docs metadata (`metadata/sdk.yaml`, `go generate`)

Drift-prone values used across README, godoc, and generated Go constants are
sourced from three authoritative places:

- **`go.mod`** — module path.
- **`VERSION`** — pinned gateway protocol version.
- **`metadata/sdk.yaml`** — canonical URLs (docs, repo, releases), Go floor,
  and install-command template.

`scripts/gen-metadata.go` reads those inputs and writes:

- `internal/version/metadata.go` — generated Go constants with the standard
  `// Code generated ... DO NOT EDIT.` header.
- `assembly/version.go` — the public `assembly.Version` constant, sourced
  from the same `VERSION` file that feeds `internal/version.ProtocolVersion`.
  AAASM-4326 replaced the previously hand-maintained literal so the public
  API surface can no longer drift from the shared metadata source. To bump
  the SDK version, edit `VERSION` and re-run `go generate ./...` — never
  edit `assembly/version.go` by hand.
- The bounded `<!-- BEGIN GENERATED: sdk-metadata --> ... <!-- END GENERATED:
  sdk-metadata -->` block inside `README.md` (surrounding prose is preserved
  byte-for-byte; only the block body is rewritten).
- The inline `<!-- BEGIN GENERATED: protocol-version --> ... <!-- END GENERATED:
  protocol-version -->` block inside `README.md`'s "Project status" prose — the
  protocol-version literal narrated mid-sentence. It is sourced from `VERSION`
  so the prose can no longer drift from the SoT (ADR 0013 / AAASM-4920); the
  same `Docs Metadata` gate below fails the build if it goes stale.

### Add or update a shared value

1. Edit the source of truth — `metadata/sdk.yaml` for docs-facing values,
   `go.mod` for the module path, `VERSION` for the protocol version (also
   feeds `assembly.Version`). Do NOT edit `internal/version/metadata.go`,
   `assembly/version.go`, or the README block by hand.
2. If you need a new field, extend `sharedMetadata`/`resolvedMetadata` in
   `scripts/gen-metadata.go`, add a rendering hook only where a real
   consumer exists (a Go caller, or a README/doc surface). Skip fields that
   nothing consumes — do not add "for symmetry" constants.
3. Run:

   ```bash
   go generate ./...
   git diff --exit-code
   go test ./...
   ```

4. Commit the regenerated files together with the source change. The
   `Docs Metadata` CI workflow will fail on any PR whose generated output is
   stale relative to the source files.

### Opting a new Markdown surface into the generated block

Paste an empty sentinel pair into the target file where you want the block
to render:

```
<!-- BEGIN GENERATED: sdk-metadata -->
<!-- END GENERATED: sdk-metadata -->
```

Then extend `scripts/gen-metadata.go` to rewrite that file too. Historical
changelog / release-note content must stay literal — do not template it
through this pipeline.

## Pull Request Checklist

Before opening a PR, please confirm:

- [ ] Branch follows `<release>/<ticket>/<type>/<short_summary>` (e.g. `v0.0.1/AAASM-1143/feat/readme_contributing`).
- [ ] Commits are small, single-purpose, and use the GitEmoji convention (`✨ feat:`, `🐛 fix:`, `📝 docs:`, `♻️ refactor:`, `✅ test:`, `🚨 lint:`).
- [ ] `make fmt`, `make lint`, `make test` all pass locally.
- [ ] `go vet ./...` is clean.
- [ ] If you touched `metadata/sdk.yaml`, `VERSION`, `go.mod`, or `scripts/gen-metadata.go`: `go generate ./... && git diff --exit-code` is clean.
- [ ] If you touched concurrency: `go test -race ./...` is clean.
- [ ] If you touched `internal/ffi/`: the native FFI build passes (`go test -tags aa_ffi_go ./...`) and the pure-Go fallback still passes (`CGO_ENABLED=0 go test ./...`).
- [ ] Public API additions have full-sentence godoc comments.
- [ ] No `//nolint` directives without an accompanying explanation.
- [ ] The PR template in `.github/PULL_REQUEST_TEMPLATE.md` is filled in (Ticket, Summary, Change Scope, Validation, Rollout Notes).
- [ ] At least one Pioneer-team reviewer is requested.
- [ ] PR title format: `[<ticket>] <emoji> (<scope>): <imperative summary>`.

CI will re-run all of the above. The local checklist exists to keep the review cycle short, not to replace CI.
