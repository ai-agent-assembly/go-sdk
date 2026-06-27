---
name: release-tag-cut
description: Cut a go-sdk release tag and let goreleaser + docs-site publish from it. Use when an operator is ready to release a new go-sdk version (e.g. v0.0.1-beta.3) on a green master — covers the tag-push path, what the tagged workflows do automatically, how to validate, and the lockstep native-shim git-SHA pin that ties go-sdk to a specific agent-assembly core commit.
---

# release-tag-cut (go-sdk)

Executable contract for releasing `github.com/ai-agent-assembly/go-sdk`. go-sdk
is a **Go module distributed by git tag** — there is no binary publish step and
no registry. A release IS a `v*` tag on `master`; everything downstream is
triggered by that tag.

> This skill ends at `git push remote v<X>`. It does not own the agent-assembly
> core release that the native shim pins — see *Native shim pin: WHY it matters*
> below for the cross-repo relationship.

## When to use

Pick this skill when **all** of the following hold:

- The operator has decided go-sdk is ready for a new tag (e.g. cutting
  `v0.0.1-beta.3` after `v0.0.1-beta.2`; the series so far is alpha → beta).
- The most recent CI run on `master` is green.
- The working tree is clean and `master` is up to date with `remote/master`.
- If this release tracks a new agent-assembly core release, the native-shim pin
  has already been bumped + merged on `master` (see the pin section below).

Triggering operator phrasing: "Tag go-sdk beta.3", "Cut the next go-sdk
release", "Release go-sdk v0.0.1-beta.3".

## When NOT to use

- **The native pin still points at an old core SHA** while you intend to ship
  against a newer agent-assembly release — stop and bump the pin first (separate
  PR), then come back. Tagging with a stale pin ships a go-sdk that links an old
  core ABI.
- **Pre-conditions not met** — `master` dirty, behind `remote/master`, or red
  CI. Surface the gap; do not remediate from inside this skill.
- **Re-cutting an existing tag** — tags are immutable here; cut the next patch
  tag instead, never force-move a published tag.

## Required context

- `remote` is the configured remote pointing at `ai-agent-assembly/go-sdk`
  (project convention — **not** `origin`, which is the personal Chisanan232
  fork).
- The operator supplies `<X>` — the full tag literal **with** the leading `v`,
  e.g. `v0.0.1-beta.3`. The skill never invents a version number.

## Pre-conditions

All MUST hold before any step runs; if any fails, stop and report.

1. **Working tree clean** (`git status --porcelain` empty).
2. **On `master`, in sync with `remote/master`** — `git fetch remote` first,
   then confirm zero ahead/behind.
3. **Most recent CI run on master is green** —
   `gh run list --branch master --limit 5`. The release-gating check is
   `ci-success` (the `ci-success.yml` aggregate gate).
4. **Native pin is intentional** — `native/aa-ffi-go/Cargo.toml` pins the core
   SHA you mean to ship against (see below). If a core bump is pending, it must
   be merged before this skill runs.
5. **Target tag `<X>` provided** — the skill does not invent or bump it.

## Executable plan

Run from a clean `master` checkout. Substitute the operator-supplied `<X>`.

1. **Sync + verify** — `git fetch remote`, confirm `master` == `remote/master`,
   working tree clean, `ci-success` green on the tip.
2. **Update the `VERSION` file** if the project tracks it for the release (it is
   the human-facing version marker; keep it consistent with the tag literal,
   minus the leading `v`). Commit as
   `🔧 (release): Set VERSION to <X-without-v>` if changed.
3. **Bump `sonar.projectVersion`** — the static `sonar.projectVersion` literal in
   `sonar-project.properties` is the source-of-truth / local-scan fallback and
   must track the release line. `coverage.yml` overrides it dynamically at scan
   time (`git describe --tags` → strip the leading `v` and any pre-release suffix
   → release line, e.g. `v0.0.1-rc.1` → `0.0.1`), so a fresh tag updates the
   SonarCloud version automatically and drift never breaks CI. Because the CI
   override resolves to the *release line*, no manual edit is needed when the new
   tag stays on the current line; bump the static literal in the same prep commit
   as the `VERSION` file (step 2) whenever the release crosses to a **new**
   release line (e.g. `0.0.1` → `0.1.0`). Commit as
   `🔧 (sonar): Bump projectVersion fallback to <release-line>` if changed. Never
   leave it at `0.0.0` — that leaves the SonarCloud gate "Not computed". (Mirrors
   the core's `release-tag-cut` automation, AAASM-3819.)
4. **Create the annotated tag** — `git tag -a "<X>" -m "go-sdk <X>"` on the
   release commit.
5. **Push the tag** — `git push remote "<X>"`. This is tag-only and touches no
   branch. NEVER `--no-verify` / never force-push. The push triggers the tagged
   workflows in *What's auto-handled*.

## Post-conditions

After step 5, both MUST hold:

1. **Tag exists on remote** — `git ls-remote --tags remote "<X>"` returns it.
2. **The tagged workflows are running** — `gh run list --workflow goreleaser.yml`
   and `gh run list --workflow docs-site.yml` show a run for `headBranch=<X>`
   `queued` / `in_progress`.

## What's auto-handled (do NOT manually run)

A pushed `v*` tag fires two workflows. Do not replicate either by hand:

- **`goreleaser.yml`** — runs goreleaser against `.goreleaser.yaml`. That config
  is deliberately **source-only** (`source.enabled: true`, `builds: [skip:true]`):
  go-sdk ships no compiled binary, so goreleaser produces the GitHub Release +
  source archive, not platform binaries. Consumers fetch the module by tag via
  `go get`. On PRs the same workflow only runs `goreleaser check` (config lint).
- **`docs-site.yml`** — on a `v*` tag, publishes the versioned documentation
  site and selects the channel (alpha/beta/stable) from the tag shape.

## Native shim pin: WHY it matters (cross-repo lockstep)

This is the load-bearing detail that makes a go-sdk release correct.

go-sdk has an **optional native FFI path**: the vendored Rust shim
`native/aa-ffi-go/` is a thin C-ABI over agent-assembly's `aa-sdk-client`. It is
compiled into go-sdk only via cgo with `-tags aa_ffi_go`; the default Go build
uses a pure-Go fallback (`CGO_ENABLED=0`), so a plain `go get` of a tag does
**not** require Rust. The native path exists to give Go programs the same shared
SDK runtime client the python/node SDKs use.

`native/aa-ffi-go/Cargo.toml` pins **two** agent-assembly crates by git SHA:

- `aa-sdk-client` — the shared runtime/IPC client.
- `aa-proto` — the generated protobuf types (`aa-sdk-client` does not re-export
  them, so the shim depends on it directly for `CheckActionRequest` /
  `CheckActionResponse`).

**Both pins MUST be the same SHA, and that SHA MUST point at the agent-assembly
core release commit go-sdk is shipping against.** WHY:

- A single resolved checkout — if the two crates pinned different revs, cargo
  would resolve two agent-assembly checkouts and `aa-proto`'s wire types could
  skew from what `aa-sdk-client` expects, silently breaking the cgo binding's
  policy-query round-trip. The same SHA guarantees one checkout, one ABI.
- Lockstep with core — when agent-assembly cuts a release, its FFI source-pin
  fanout advances this SHA so go-sdk links the matching core ABI. Releasing
  go-sdk against a stale SHA ships a binding compiled against an outdated core,
  which is the failure this whole mechanism prevents. Keep the SHA in lockstep
  with the node/python shims (`native/aa-ffi-node`, `rust/aa-ffi-python`), which
  pin the same core commit.

**The `native-pin-consistency.yml` gate enforces the invariant in CI.** It runs
on any PR/push that touches `native/aa-ffi-go/Cargo.toml` and asserts that every
`agent-assembly` git dependency pins **exactly one** rev (ADR 0003 lockstep). A
PR that bumps only one of the two crates — or introduces a second rev — fails
this gate. So the pin bump always lands as its own PR (both crates, same SHA)
**before** this release skill runs; this skill never edits the pin.

cgo build note: building/testing the native path needs a Rust toolchain **and**
`protoc` on PATH (`aa-proto` regenerates protobufs). `make native` builds the
shim; `make test-native` runs the suite against the real cgo binding. CI does
this in `native-ffi.yml` (the only lane that sets `-tags aa_ffi_go`); the
default test matrix never compiles the native path.

## Docs / version references

go-sdk is the simplest case here. Unlike the python/node SDKs, **go-sdk has no
in-repo pinned version strings in its docs** — install is
`go get github.com/ai-agent-assembly/go-sdk@vX`, which resolves the module by tag,
so there is no checked-in version literal to bump beyond the `assembly/version.go`
`Version` const (and the `VERSION` file), which the release flow / goreleaser owns.

The one place a published go-sdk version *is* pinned lives **outside this repo**:
go-sdk's runnable **examples live in the `agent-assembly-examples` repo** and pin a
published go-sdk version in their `go.mod` + README prerequisite tables. Those track
the **currently-published** tag, so they must be bumped **after** this release
publishes (the consumer-repo timing rule) — **not** as part of cutting the tag.
Bumping them before the tag is published points `go get` at a tag that does not yet
exist and breaks the example build.

For the canonical statement of this principle, see the agent-assembly core
`release-docs-sync` skill: a doc/example pin to the release that *ships* a feature
is a correct forward-reference, not a stale pin — which is exactly why the example
bump follows publication rather than preceding the tag.

## What this skill explicitly does not do

- Bump the native-shim core SHA pin (separate, pre-release PR gated by
  `native-pin-consistency.yml`).
- Cut the agent-assembly core release the pin points at (that lives in the
  agent-assembly monorepo's own `release-tag-cut`).
- Build or publish any binary — go-sdk is tag-distributed; there is none.
- Force-move or re-cut an existing tag.
- Touch repos other than `ai-agent-assembly/go-sdk`.
