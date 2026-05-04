---
title: API Reference
weight: 90
---

# API Reference

The canonical, version-pinned API reference for `go-sdk` is hosted on
**[pkg.go.dev](https://pkg.go.dev/github.com/agent-assembly/go-sdk)**. It is
auto-generated from the godoc comments in the source on every released tag,
so there is **no separate publish workflow** to maintain — push a `vX.Y.Z`
tag and pkg.go.dev picks it up within minutes.

The page above lists every exported package, type, function, constant, and
variable, with their godoc and source links.

> **Note** — pkg.go.dev indexing is **blocked today** pending a module-path
> rename. The `go.mod` declares `github.com/agent-assembly/go-sdk` while the
> canonical GitHub URL is `github.com/AI-agent-assembly/go-sdk`. Until that
> rename ticket lands, the link above will resolve to a 404. Use the
> [local godoc preview](#local-preview) below to read the same information
> from a clone of the repo.
