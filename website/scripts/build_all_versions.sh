#!/usr/bin/env bash
# Build every documentation version listed in website/data/versions.toml into
# the Pages artifact tree (AAASM-2830).
#
# The Pages deploy artifact is a full-site replacement, so every deploy must
# include every version subpath, not just the version under publication. This
# script walks the recomputed versions.toml and, for each entry, materialises
# the appropriate Hugo build into public/<subpath>/.
#
# Mapping:
#   channel = "latest"                 -> built from master's HEAD
#   channel = "archived", version=v..  -> built from that tag
#   channel = "stable", tag=v..        -> built from that tag (alias subpath)
#   channel = "pre-release", tag=v..   -> built from that tag (alias subpath)
#
# For each historical build the recomputed versions.toml is copied INTO the
# worktree before Hugo runs, so the version selector in every snapshot lists
# the current full set of channels + archived entries.
#
# Required environment:
#   PAGES_BASE   site root, e.g. https://ai-agent-assembly.github.io/go-sdk
#   PUBLIC_DIR   absolute path to website/public (created if missing)
#   REPO_ROOT    absolute path to the repo root (for git worktree add)
#   MASTER_REF   git ref to build for the "latest" channel (defaults to master)
#
# Usage: bash website/scripts/build_all_versions.sh
set -euo pipefail

: "${PAGES_BASE:?PAGES_BASE must be set}"
: "${PUBLIC_DIR:?PUBLIC_DIR must be set}"
: "${REPO_ROOT:?REPO_ROOT must be set}"
MASTER_REF="${MASTER_REF:-origin/master}"

VERSIONS_TOML="${REPO_ROOT}/website/data/versions.toml"
if [ ! -f "$VERSIONS_TOML" ]; then
  echo "::error::versions.toml not found at $VERSIONS_TOML"
  exit 1
fi

mkdir -p "$PUBLIC_DIR"

# Parse versions.toml into "<channel>\t<version>\t<path>\t<tag>" lines and read
# them into ENTRIES. Avoid `mapfile -t` because the macOS dev environment ships
# bash 3.2 (no mapfile) and the script is occasionally invoked there for
# pre-deploy smoke checks; CI on ubuntu has bash 5 either way.
ENTRIES=()
while IFS= read -r line; do
  ENTRIES+=("$line")
done < <(python3 - "$VERSIONS_TOML" <<'PY'
import re, sys
with open(sys.argv[1], encoding="utf-8") as fh:
    text = fh.read()
for body in re.split(r'(?m)^\[\[versions\]\]\s*$', text)[1:]:
    fields = dict(re.findall(r'(\w+)\s*=\s*"([^"]+)"', body))
    print("\t".join([
        fields.get("channel", ""),
        fields.get("version", ""),
        fields.get("path", ""),
        fields.get("tag", ""),
    ]))
PY
)

# Resolve the build ref for an entry. Returns the git ref to check out, or the
# string "__skip__" if the entry doesn't correspond to a buildable subpath.
resolve_ref() {
  local channel="$1" version="$2" tag="$3"
  case "$channel" in
    latest)      printf '%s' "$MASTER_REF" ;;
    archived)    printf '%s' "$version" ;;
    stable|pre-release)
      if [ -n "$tag" ]; then printf '%s' "$tag"; else printf '%s' "__skip__"; fi
      ;;
    *)           printf '%s' "__skip__" ;;
  esac
}

# Build one (subpath, ref) target into PUBLIC_DIR/<subpath>/.
build_one() {
  local subpath="$1" ref="$2"
  local worktree
  # mktemp -d is fine but we need it under a path git allows; use the runner
  # temp area when available, otherwise a sibling of the repo.
  worktree="$(mktemp -d -t docs-build-XXXXXX)"
  # Clean up the worktree even on failure so reruns don't leak state.
  trap 'git -C "$REPO_ROOT" worktree remove --force "$worktree" >/dev/null 2>&1 || true; rm -rf "$worktree"' RETURN

  echo "::group::Build subpath=$subpath ref=$ref"
  git -C "$REPO_ROOT" worktree add --detach "$worktree" "$ref"

  # Overlay the recomputed versions.toml so historical snapshots show the
  # current channel + archived listing in their selector dropdowns. Pre-AAASM-2754
  # tags (alpha-1..alpha-4) predate the versioning feature and therefore have no
  # website/data/ directory; create it on demand. Tags that predate the version
  # selector partial render without a selector — that is the correct archival
  # behaviour, since the selector was a later addition.
  mkdir -p "$worktree/website/data"
  cp "$VERSIONS_TOML" "$worktree/website/data/versions.toml"

  (
    cd "$worktree/website"
    # Older tags may pin different Hugo modules; tidy + build per tag so each
    # snapshot resolves its own dependency set.
    hugo mod tidy
    hugo --gc --minify \
      --baseURL "${PAGES_BASE}/${subpath}/" \
      --destination "${PUBLIC_DIR}/${subpath}"
  )

  git -C "$REPO_ROOT" worktree remove --force "$worktree" >/dev/null 2>&1 || true
  rm -rf "$worktree"
  trap - RETURN
  echo "::endgroup::"
}

# Walk every parsed entry and build it. Skipping silently when an entry's git
# ref does not exist (e.g. a stale archived row pointing at a deleted tag) keeps
# the deploy from failing on bookkeeping drift; the warning is printed instead.
for line in "${ENTRIES[@]}"; do
  IFS=$'\t' read -r channel version path tag <<<"$line"
  [ -z "$channel" ] && continue
  ref="$(resolve_ref "$channel" "$version" "$tag")"
  if [ "$ref" = "__skip__" ]; then
    echo "Skipping entry channel=$channel version=$version (no buildable ref)."
    continue
  fi
  if ! git -C "$REPO_ROOT" rev-parse --verify --quiet "${ref}^{commit}" >/dev/null; then
    echo "::warning::Git ref '$ref' for channel=$channel version=$version not found; skipping."
    continue
  fi
  # Strip the leading + trailing slash off the path field to get the subpath
  # that lands under public/.
  subpath="${path#/}"
  subpath="${subpath%/}"
  if [ -z "$subpath" ]; then
    echo "::warning::Empty subpath for channel=$channel version=$version; skipping."
    continue
  fi
  build_one "$subpath" "$ref"
done

echo "Built subpaths:"
ls -1 "$PUBLIC_DIR"
