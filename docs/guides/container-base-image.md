---
title: Use the governed container base image
weight: 4
---

# Use the governed container base image

When you ship a Go agent as a container, you can skip the "install the SDK and
the CLI into the image" step entirely. The project publishes a **governed Go
base image** that already bundles the `aasm` CLI and the
`github.com/ai-agent-assembly/go-sdk` module, so an agent built `FROM` it is
governed out of the box — no extra install layer, no version drift between the
CLI and the SDK in your image.

This guide covers what the image is, how to pick a tag, and how to build and run
a governed Go agent on top of it.

## What it is

The image lives in GitHub Container Registry:

```text
ghcr.io/ai-agent-assembly/go:{1.24,1.25,1.26}-alpine
```

Each variant is a standard Alpine-based Go toolchain image (one per supported Go
runtime) with two things added on top:

- **The `aasm` CLI**, the AI Agent Assembly operator front-end, on `PATH`. You
  can run `aasm topology`, `aasm policy`, and the other subcommands from inside
  the container without installing anything.
- **The `github.com/ai-agent-assembly/go-sdk` module**, pre-installed via
  `go install` so the SDK resolves from your build cache. Your agent's
  `import "github.com/ai-agent-assembly/go-sdk/assembly"` works against a known,
  pinned SDK version baked into the image.

The result: a containerised Go agent built on this base is governed with no
extra install step. You write your agent against the
[`assembly` package]({{< relref "/quick-start" >}}) as usual, and the
governance tooling is already present.

## Tags: how to choose

The image is published under two kinds of tags. Pick based on whether you value
reproducibility (pin the immutable tag) or "track the latest" convenience.

| Tag form | Example | Mutability | Use it for |
|---|---|---|---|
| `go:<runtime>-<core-version>` | `go:1.26-alpine-v0.0.1-rc.1` | **Immutable** — never re-pointed | CI and production. Reproducible, byte-for-byte rebuildable. |
| `go:<runtime>` | `go:1.26-alpine` | Moving — follows the latest core release for that runtime | Local development, quick experiments. |
| `go:latest` | `go:latest` | Moving — latest runtime + latest core release | Throwaway / "just give me the newest" only. |

`<runtime>` is the Go toolchain version (`1.24`, `1.25`, `1.26`). `<core-version>`
is the **Agent Assembly core / `aa-runtime` release** baked into the image — and
it is also the version of the `aasm` CLI inside the image. Pinning
`go:<runtime>-<core-version>` therefore pins *both* the Go toolchain and the
governance tooling, which is exactly what you want for a reproducible build.

> In CI and production, **always pin the immutable
> `go:<runtime>-<core-version>` tag**. The moving tags (`go:<runtime>`,
> `go:latest`) are re-pointed when a new core release ships, so a rebuild can
> silently pick up a different toolchain or runtime.

## Quick start

Build your agent `FROM` an immutable tag, then run it. A minimal `Dockerfile`:

```dockerfile
# Pin the immutable tag: Go 1.26 toolchain + core v0.0.1-rc.1 aasm CLI + go-sdk.
FROM ghcr.io/ai-agent-assembly/go:1.26-alpine-v0.0.1-rc.1

WORKDIR /app

# Your agent's module. go-sdk is already resolvable from the base image's
# module cache, so `go build` finds it without a separate install step.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/agent ./cmd/agent

ENTRYPOINT ["/usr/local/bin/agent"]
```

Build and run it:

```bash
docker build -t my-governed-agent .
docker run --rm my-governed-agent
```

Two things are true inside this image without any extra setup:

- **`aasm --version` works** — the CLI is on `PATH` and reports the core version
  baked into the tag you pinned.
- **The go-sdk module resolves** — `import "github.com/ai-agent-assembly/go-sdk/assembly"`
  compiles against the SDK version installed in the image.

The default build uses the pure-Go, fail-closed transport (`CGO_ENABLED=0`), so
no C compiler is needed in your build stage. See
[Core Concepts]({{< relref "/core-concepts#the-ffi-transport-bridge" >}}) for the
opt-in native FFI transport.

## Choosing the SDK: the `SDK_VERSION` build-arg

The base image is built with an **optional** `SDK_VERSION` build argument that
controls which release of the go-sdk module is `go install`-ed into the image:

| `SDK_VERSION` | Result |
|---|---|
| *unset* (default) | The **latest stable** go-sdk release. If no stable release exists yet, the latest **pre-release** is used. |
| set to a tag | That **exact** version is pinned — e.g. `--build-arg SDK_VERSION=v0.0.1-beta.3`. |

```bash
# Build the base image (or your own derivative) pinned to a specific SDK release.
docker build --build-arg SDK_VERSION=v0.0.1-beta.3 -t my-go-base .
```

The **published** `ghcr.io/ai-agent-assembly/go` images always pin `SDK_VERSION`
to a concrete release, so each immutable tag has one known SDK version. A bare
`docker build` with no `--build-arg` gets the default (latest stable, else latest
pre-release). For a reproducible build, pin it explicitly.

## Best practices

- **Pin the immutable tag in CI and production.** Use
  `go:<runtime>-<core-version>` (e.g. `go:1.26-alpine-v0.0.1-rc.1`); never build
  production images `FROM ...:latest`.
- **Pair the image with the `aa-runtime` sidecar for enforcement.** The SDK's
  in-process interception layer is the fast path, but it is **not a security
  boundary on its own** — a determined agent process can bypass an in-process
  check. Run the `aa-runtime` alongside your container so policy is enforced out
  of process. See
  [Handle allow/deny decisions and errors]({{< relref "/guides/handle-decisions-and-errors" >}})
  for fail-closed vs fail-open posture.
- **Keep the image core-version and your runtime version aligned.** The
  `<core-version>` in the tag is the `aasm` / `aa-runtime` release; run a sidecar
  of the matching version so the SDK, CLI, and runtime all speak the same
  protocol. See [Compatibility & Versioning]({{< relref "/compatibility" >}}).
- **Rebuild per release.** When a new core version ships, bump the pinned tag and
  rebuild rather than relying on a moving tag to drift you forward.

## See also

- **Canonical core guide** —
  [Container base images](https://github.com/ai-agent-assembly/agent-assembly/blob/HEAD/docs/src/usage-guide/container-base-images.md)
  in the agent-assembly core repo, covering the image family across all SDKs.
- **ADR 0009** —
  [Versioned base image tags and SDK pinning](https://github.com/ai-agent-assembly/agent-assembly/blob/HEAD/docs/src/adr/0009-versioned-base-image-tags-and-sdk-pinning.md),
  the design rationale for the immutable/moving tag scheme and the `SDK_VERSION`
  build-arg.
