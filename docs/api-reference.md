---
title: API Reference
weight: 90
---

# API Reference

The canonical, version-pinned API reference for `go-sdk` is hosted on
**[pkg.go.dev](https://pkg.go.dev/github.com/AI-agent-assembly/go-sdk)**. It is
auto-generated from the godoc comments in the source on every released tag,
so there is **no separate publish workflow** to maintain — push a `vX.Y.Z`
tag and pkg.go.dev picks it up within minutes.

The page above lists every exported package, type, function, constant, and
variable, with their godoc and source links.

## Local preview

Run godoc against the working tree to read the same content offline:

```bash
go install golang.org/x/tools/cmd/godoc@latest
cd go-sdk
godoc -http=:6060
```

Then open <http://localhost:6060/pkg/github.com/AI-agent-assembly/go-sdk/> in
your browser. Every exported symbol on `master` is rendered with the same
godoc the public page would show.

Run `godoc` from the repo root (or any directory inside the module) — it
follows `GOPATH`/module resolution to locate the source.

## See also

- [Architecture](architecture/) — module layout, FFI bridge, interceptor
  flow, context propagation, and tool wrapping. Read this first if you want
  to know *why* a particular API is shaped the way it is.
- [Getting Started](getting-started/) — install, init, wrap your tools.
- [Guides](guides/) — topical how-tos (context propagation, FFI modes,
  error handling).
- [Contributing](https://github.com/AI-agent-assembly/go-sdk/blob/master/CONTRIBUTING.md) —
  conventions enforced in code review (context-first, `%w` wrapping,
  `io.Closer`, functional options, `internal/` boundary).
