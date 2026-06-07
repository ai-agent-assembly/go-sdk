---
title: Cross-SDK Hugo Setup Template
weight: 99
---

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
│   ├── getting-started.md
│   ├── architecture.md
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
| Hosting | GitHub Pages, project-site URL |
| URL scheme | `https://ai-agent-assembly.github.io/<sdk-name>/` |
| Default branch (deploy trigger) | `master` |
| Build command | `cd website && hugo --minify` |
| Local preview | `cd website && hugo server -D` |
| Pages source | "GitHub Actions" (set once in repo Settings → Pages) |

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

- Custom domain (DNS / CNAME) — leave on `*.github.io` until there is a
  unified landing page for all three SDKs.
- Versioned docs — single-version (latest `master`) for now; add Hugo's
  version selector if/when an LTS branch emerges.

## Adopting this template in `python-sdk` / `node-sdk`

1. Copy the `website/` directory verbatim; bump the `module` line in
   `website/go.mod` to `github.com/ai-agent-assembly/<sdk-name>/website`.
2. Copy this `docs/` directory; replace the `_index.md`, `getting-started.md`,
   and `TEMPLATE.md` to point at your SDK.
3. Copy `.github/workflows/docs-site.yml` verbatim; no changes needed if the
   path conventions match.
4. Set Pages source = "GitHub Actions" once.
5. First push to `master` deploys to `https://ai-agent-assembly.github.io/<sdk-name>/`.
