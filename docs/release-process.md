---
title: Release process
weight: 7
---

# Release process

`go-sdk` follows the standard Go module release model: **a release is a git
tag**. There is no separate publish step to `go get` a tagged version — the Go
module proxy and [pkg.go.dev](https://pkg.go.dev/github.com/AI-agent-assembly/go-sdk)
index the tag automatically.

## Versioning

- **Semantic Versioning** (`vMAJOR.MINOR.PATCH`), with pre-release suffixes for
  the current alpha line (`v0.0.1-alpha.N`).
- The module path is stable — `github.com/AI-agent-assembly/go-sdk` — so a
  major bump past `v1` would add a `/vN` suffix per Go's import-compatibility
  rule.
- [`VERSION`](https://github.com/AI-agent-assembly/go-sdk/blob/master/VERSION)
  is **not** the release version; it pins the gateway protocol the SDK targets.
  See [Compatibility](compatibility/).

## Cutting a release

1. Land all changes on `master` with CI green.
2. If the gateway protocol changed, bump [`VERSION`](https://github.com/AI-agent-assembly/go-sdk/blob/master/VERSION)
   and update the conformance vectors; confirm with `make test`.
3. Tag the commit and push the tag:

   ```bash
   git tag v0.0.1-alpha.4
   git push remote v0.0.1-alpha.4
   ```

4. Publish the **release notes** on the
   [GitHub Releases](https://github.com/AI-agent-assembly/go-sdk/releases) page
   for that tag.

## What happens after the tag

- `go get github.com/AI-agent-assembly/go-sdk@v0.0.1-alpha.4` resolves through
  the module proxy immediately.
- pkg.go.dev renders the godoc for the tag within minutes.
- The `goreleaser` config (`.goreleaser.yaml`) is validated on every push to
  `master` by the `goreleaser-check` job; it is set up for source archives,
  with binary builds skipped (this is a library, not a CLI).

## Consuming a pinned version

```go
require github.com/AI-agent-assembly/go-sdk v0.0.1-alpha.3
```

Pin an exact tag while the SDK is pre-release; review the release notes before
moving to a newer alpha.
