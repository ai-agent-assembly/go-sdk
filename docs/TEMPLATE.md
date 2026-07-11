---
title: Cross-SDK Hugo Setup Template
weight: 99
excludeSearch: true
sidebar:
  exclude: true
---

<!--
This page is an internal maintainer convention reference for the docs site
shape, not user-facing SDK documentation. It is excluded from the sidebar nav
and site search on purpose.
-->


# Cross-SDK Hugo Setup Template

This page records the **convention** used by `go-sdk` for its docs site, so
that `python-sdk`, `node-sdk`, and any future SDK can adopt the same shape
without re-deriving it.

> Source of truth: this `go-sdk` repo. If this template diverges from
> what is shipped here, the live setup wins — update this page.

## Repo layout

```
<sdk-repo>/
├── README.md                 # at root (GitHub convention)
├── CONTRIBUTING.md           # at root (GitHub convention)
├── docs/                     # Markdown content — browseable on GitHub
│   ├── _index.md             # landing page
│   ├── quick-start.md
│   ├── core-concepts.md
│   ├── api-reference.md
│   ├── guides/
│   └── TEMPLATE.md           # this file
├── website/                  # Hugo project (configs only — no content)
│   ├── hugo.toml
│   ├── go.mod                # Hugo modules incl. Hextra
│   ├── layouts/              # only theme overrides if Hextra defaults insufficient
│   ├── static/               # favicon, og-image
│   └── archetypes/
└── .github/workflows/
    └── docs-site.yml         # builds Hugo, deploys to GitHub Pages
```

**Why split `website/` from `docs/`** — Markdown stays browseable on
GitHub (no Hugo shortcodes hidden inside the source) while the Hugo
project plumbing stays out of the SDK's main eye-line.

## Conventions

| Setting | Value |
|---|---|
| Static-site generator | Hugo (extended build) |
| Theme | [Hextra](https://github.com/imfing/hextra) via Hugo Modules |
| Hosting | GitHub Pages, project-site build target — see "Two-tier URL model" below |
| Reader-facing URL | `https://docs.agent-assembly.com/<sdk-name>/` |
| Internal Pages build target (not reader-facing) | `https://ai-agent-assembly.github.io/<sdk-name>/` |
| Default branch (deploy trigger) | `master` |
| Build command | `cd website && hugo --minify` |
| Local preview | `cd website && hugo server -D` |
| Pages source | "GitHub Actions" (set once in repo Settings → Pages) |

## Two-tier URL model

There are two URLs in play, and only one of them is ever shown to a reader:

1. **`https://docs.agent-assembly.com/<sdk-name>/` — canonical, reader-facing.**
   Owned by the [`docs`](https://github.com/ai-agent-assembly/docs) repo's hub
   aggregation pipeline (`aggregate.sh`), which either rebuilds the source
   repo's site from scratch, or — for MkDocs-based repos — clones an
   already-published `gh-pages` branch verbatim. `website/hugo.toml`'s
   `baseURL` must be set to this pattern. This repo's own `website/hugo.toml`
   sets `baseURL = "https://docs.agent-assembly.com/go-sdk/"` — copy that
   shape, swapping in the new SDK's name.
2. **`https://ai-agent-assembly.github.io/<sdk-name>/` — internal-only
   infrastructure.** This is the raw GitHub Pages deploy target that
   `.github/workflows/docs-site.yml` publishes to; the aggregation pipeline in
   (1) consumes it as an input. It must never be linked to or surfaced to
   readers directly.

## Shared brand styling (Path A)

Styling now follows the **shared documentation brand kit** tracked in the
[`agent-assembly-docs`](https://github.com/ai-agent-assembly/docs)
repo under `design/`. The kit is applied here (Path A — vendor the snippet into
each SDK site) by dropping `design/snippets/hextra-custom.css` at
`website/assets/css/custom.css` (Hextra auto-loads it) and the brand logo /
favicon from `design/brand/` into `website/static/images/`. Re-sync from
`agent-assembly-docs` `design/` when the kit changes.

## Required `hugo.toml` keys

- `baseURL` — must end with trailing slash for project-site relative URLs to resolve
- `[module] [[module.imports]] path = "github.com/imfing/hextra"`
- `[[module.mounts]] source = "../docs" target = "content"` — content lives in the sibling `docs/` directory
- `[markup.goldmark.renderer] unsafe = true` — Hextra needs raw HTML
- `[markup.highlight] noClasses = false` — let the theme drive syntax highlighting

## Required GitHub Actions setup

1. Repo Settings → Pages → Source = **GitHub Actions** (one-time, manual)
2. Workflow uses `actions/configure-pages`, `actions/upload-pages-artifact@v3`,
   `actions/deploy-pages@v4`
3. Permissions block on the workflow:

   ```yaml
   permissions:
     contents: read
     pages: write
     id-token: write
   ```

## Out-of-scope decisions (deferred to per-SDK choice)

- Custom domain — already resolved, not deferred: the reader-facing URL is
  `https://docs.agent-assembly.com/<sdk-name>/`, served by the `docs` repo's
  hub aggregation pipeline. Set `website/hugo.toml`'s `baseURL` to that
  pattern (see "Two-tier URL model" above for the worked example from this
  repo). Do not leave `baseURL` on `*.github.io` — that URL is internal-only
  infrastructure, not the public entry point.
- Versioned docs — single-version (latest `master`) for now; add Hugo's
  version selector if/when an LTS branch emerges.

## Adopting this template in `python-sdk` / `node-sdk`

1. Copy the `website/` directory verbatim; bump the `module` line in
   `website/go.mod` to `github.com/ai-agent-assembly/<sdk-name>/website`.
2. Copy this `docs/` directory; replace the `_index.md`, `quick-start.md`,
   and `TEMPLATE.md` to point at your SDK.
3. Copy `.github/workflows/docs-site.yml` verbatim; no changes needed if the
   path conventions match.
4. Set Pages source = "GitHub Actions" once.
5. First push to `master` deploys to the internal GitHub Pages build target
   `https://ai-agent-assembly.github.io/<sdk-name>/`. That is not the
   reader-facing URL — the `docs` repo's aggregation pipeline picks it up
   from there and republishes it at the canonical
   `https://docs.agent-assembly.com/<sdk-name>/`, which is what `baseURL` in
   `website/hugo.toml` must be set to (see "Two-tier URL model" above).
