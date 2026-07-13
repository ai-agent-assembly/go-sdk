# Quick-start per-framework tab snippets (vendored)

Source data for the per-framework tabs in `docs/quick-start.md` "Govern your
first agent" step (Epic **AAASM-4511**, this SDK's slice **AAASM-4515**).

## What lives here

- `manifest.json` — the data-driven tab index for the Go SDK. `frameworks[]`
  mirrors the `sdks.go[]` slice of the examples repo snippet manifest;
  `quickstart_tabs[]` is the ordered subset surfaced as quick-start tabs.
- `_snippets/*.go.txt` — the extracted `quickstart` region (the `WrapTools` +
  wiring slice) of each example entrypoint, one file per framework, copied
  byte-for-byte. These are illustrative fragments, not compilable packages: the
  leading `_` keeps the directory out of `go build ./...`, and the `.go.txt`
  extension keeps the fragments out of `make fmt`'s `gofmt` glob (`find … -name
  '*.go'`) while the `.txt` file still highlights as Go via the docs code fence.

## Vendored, not fetched

Everything here is **vendored** (copied) from the examples repo
(`ai-agent-assembly/examples`, produced by the AAASM-4512 snippet tooling,
PR examples#267). The docs build never reaches out to the examples repo; the
committed copy is the single source of truth for this SDK, and a CI drift check
(`go generate ./...` + `git diff`, see `.github/workflows/docs-metadata.yml`)
fails the build if the rendered tabs in `docs/quick-start.md` fall out of sync
with these files.

## Tab set

Go's per-framework surface is thin today: **LangChainGo** is the only real
framework example; `basic-agent` is the framework-agnostic "Plain" path. Those
two are the quick-start tabs (`quickstart_tabs`), matching the cross-SDK
`{ LangChainGo, Plain }` decision on AAASM-4515.

`tool-policy` and `cli-runtime-integration` are vendored and tracked for
parity with the examples manifest, but they are **patterns** (allow/deny policy
demo, sidecar wiring), not "your first agent" frameworks, so they are
intentionally excluded from the quick-start tabs. The list is data-driven: when
a new Go **framework** example lands, vendor its snippet + manifest entry and
add its `framework_id` to `quickstart_tabs`, and it appears automatically.

## Regenerate

```bash
go generate ./...        # from the repo root; rewrites the tabs block in docs/quick-start.md
```
