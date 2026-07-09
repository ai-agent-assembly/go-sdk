#!/usr/bin/env python3
"""Reusable serializer for ``website/data/versions.toml`` (AAASM-3760).

This module is the single, standalone-callable entrypoint that renders the docs
version-selector data file. The logic used to live inlined inside the
``Recompute versions.toml from git tags`` step of
``.github/workflows/docs-site.yml``; it was extracted here verbatim so the same
serializer can be reused without copy-pasting (notably by the downstream docs
aggregator in ``docs``, see AAASM-3757). Channel computation
itself is owned by :mod:`versions_channels`; this module only handles parsing
the existing file, preserving the hand-curated ``latest`` entry, and emitting
the final TOML.

CLI usage (drop-in replacement for the previous inline ``python3 -`` heredoc)::

    python3 website/scripts/render_versions_toml.py <path> [tag ...]

The named ``<path>`` is read for its existing ``latest`` metadata and then
overwritten in place with the recomputed file. The same set of tags the
workflow discovers via ``git tag -l 'v*'`` is passed as the remaining argv.

Library usage::

    from render_versions_toml import render
    new_text = render(existing_text, ["v0.1.0", "v0.1.1-rc.1"])
"""

from __future__ import annotations

import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from versions_channels import compute_channels, parse_tag  # noqa: E402


def _emit(r: dict[str, str]) -> str:
    """Serialize a single ``[[versions]]`` record to TOML."""
    order = ["channel", "version", "label", "path", "tag", "default"]
    keys = order + [k for k in r if k not in order]
    lines = ["[[versions]]"]
    for k in keys:
        if k not in r:
            continue
        v = r[k]
        if v in ("true", "false"):
            lines.append(f"  {k} = {v}")
        else:
            lines.append(f'  {k} = "{v}"')
    return "\n".join(lines)


def render(text: str, raw_tags: list[str]) -> str:
    """Return the recomputed ``versions.toml`` content.

    ``text`` is the current file content (used to preserve any hand-curated
    ``latest`` metadata — label, default flag — the engineer set). ``raw_tags``
    is the full set of known release tags; channels are recomputed from scratch
    every call so a superseding stable correctly drops a now-behind pre-release.
    """
    # Parse existing entries so we can preserve any hand-curated "latest"
    # metadata (label, default flag) the engineer set in the checked-in file.
    blocks = re.split(r"(?m)^\[\[versions\]\]\s*$", text)
    head = blocks[0]
    existing = []
    for body in blocks[1:]:
        fields = dict(re.findall(r'(\w+)\s*=\s*"([^"]+)"', body))
        flags = dict(re.findall(r"(\w+)\s*=\s*(true|false)\b", body))
        fields.update({k: v for k, v in flags.items()})
        existing.append(fields)

    # Keep the latest (master) entry as-is. If the user removed it, fall
    # back to a sensible default so the moving "latest" channel always
    # renders.
    latest = [r for r in existing if r.get("channel") == "latest"]
    if not latest:
        latest = [
            {
                "channel": "latest",
                "version": "latest",
                "label": "latest (master)",
                "path": "/latest/",
                "default": "true",
            }
        ]

    # De-duplicate, keep only well-formed semver tags. Drop any raw_tags
    # entries that don't parse so a stray junk ref can't break the deploy.
    tag_set = {t for t in raw_tags if t and parse_tag(t) is not None}
    channels = compute_channels(sorted(tag_set))

    # Archived = one entry per concrete release tag, kept reachable.
    archived = []
    for tag in sorted(tag_set, key=lambda t: parse_tag(t).raw if parse_tag(t) else t):
        archived.append(
            {
                "channel": "archived",
                "version": tag,
                "label": tag,
                "path": f"/{tag}/",
            }
        )

    out_records = list(latest)
    for channel in ("stable", "pre-release"):
        tag = channels.get(channel)
        if tag:
            out_records.append(
                {
                    "channel": channel,
                    "version": channel,
                    "label": f"{channel} ({tag})",
                    "path": f"/{channel}/",
                    "tag": tag,
                }
            )
    out_records.extend(archived)

    out = head.rstrip("\n") + "\n"
    out += "\n" + "\n\n".join(_emit(r) for r in out_records) + "\n"
    return out


def main(argv: list[str]) -> int:
    path = argv[1]
    raw_tags = argv[2:]

    with open(path, encoding="utf-8") as fh:
        text = fh.read()

    out = render(text, raw_tags)

    with open(path, "w", encoding="utf-8") as fh:
        fh.write(out)

    tag_set = {t for t in raw_tags if t and parse_tag(t) is not None}
    channels = compute_channels(sorted(tag_set))
    print(f"Recomputed channels {channels} from {len(tag_set)} tag(s).")
    print(open(path, encoding="utf-8").read())
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
