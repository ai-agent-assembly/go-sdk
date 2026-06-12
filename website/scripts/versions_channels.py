#!/usr/bin/env python3
"""Channel computation for the versioned docs selector (AAASM-2754).

The Go SDK docs are published under moving CHANNELS (latest / stable /
pre-release) plus frozen ARCHIVED snapshots, one per concrete release tag. This
module owns the rule that decides which moving channel entries appear in
``website/data/versions.toml`` and is invoked by ``.github/workflows/docs-site.yml``
on every tag push.

The pre-release channel gate
----------------------------
The ``pre-release`` channel is emitted ONLY IF the newest pre-release tag is
strictly greater (by semver precedence) than the newest stable tag. Otherwise no
pre-release channel is rendered: a stable release that supersedes the newest
pre-release silently drops the ``pre-release`` selector entry and its
``/pre-release/`` alias build. The superseded pre-release stays reachable as an
archived entry. With no stable release at all, any pre-release is shown.

Channels are recomputed from the FULL set of known tags every run (never
incrementally appended), so a later superseding stable correctly removes a
previously-shown pre-release channel.

Semver precedence (subset of semver.org §11 needed here)
  * ``X.Y.Z-<pre> < X.Y.Z`` (a pre-release is lower than its release)
  * release identifiers compared numerically by major, minor, patch
  * pre-release compared dot-by-dot: numeric ids numerically, alphanumeric ids
    lexically (ASCII), numeric < alphanumeric, fewer fields < more fields
  * ``alpha < beta < rc`` falls out of the lexical ordering of identifiers
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from functools import cmp_to_key

# vMAJOR.MINOR.PATCH with an optional -prerelease suffix.
_TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)(?:-(.+))?$")


@dataclass(frozen=True)
class SemVer:
    """A parsed release tag (build metadata is not used and not parsed)."""

    major: int
    minor: int
    patch: int
    # None for a stable release; the dot-separated identifier list otherwise.
    prerelease: tuple[str, ...] | None
    raw: str

    @property
    def is_prerelease(self) -> bool:
        return self.prerelease is not None


def parse_tag(tag: str) -> SemVer | None:
    """Parse a ``vX.Y.Z`` / ``vX.Y.Z-pre`` tag, or return None if malformed."""
    m = _TAG_RE.match(tag)
    if not m:
        return None
    major, minor, patch, pre = m.group(1), m.group(2), m.group(3), m.group(4)
    prerelease = tuple(pre.split(".")) if pre is not None else None
    return SemVer(int(major), int(minor), int(patch), prerelease, tag)


def _cmp_prerelease_ids(a: str, b: str) -> int:
    a_num, b_num = a.isdigit(), b.isdigit()
    if a_num and b_num:
        ai, bi = int(a), int(b)
        return (ai > bi) - (ai < bi)
    if a_num != b_num:
        # Numeric identifiers always have lower precedence than alphanumeric.
        return -1 if a_num else 1
    return (a > b) - (a < b)


def compare(a: SemVer, b: SemVer) -> int:
    """Return -1/0/1 for semver precedence of ``a`` vs ``b``."""
    for x, y in ((a.major, b.major), (a.minor, b.minor), (a.patch, b.patch)):
        if x != y:
            return -1 if x < y else 1

    # Equal core version: a release outranks any pre-release of it.
    if a.prerelease is None and b.prerelease is None:
        return 0
    if a.prerelease is None:
        return 1
    if b.prerelease is None:
        return -1

    for ai, bi in zip(a.prerelease, b.prerelease):
        c = _cmp_prerelease_ids(ai, bi)
        if c != 0:
            return c
    # All shared fields equal: the longer identifier list has higher precedence.
    la, lb = len(a.prerelease), len(b.prerelease)
    return (la > lb) - (la < lb)


def _newest(versions: list[SemVer]) -> SemVer | None:
    if not versions:
        return None
    return max(versions, key=cmp_to_key(compare))


def compute_channels(tags: list[str]) -> dict[str, str]:
    """Compute the moving channel -> tag map from the full set of release tags.

    Returns a dict that may contain the keys ``"stable"`` and ``"pre-release"``,
    each mapping to the concrete tag the channel points at. The ``latest``
    channel (master) is not a release tag and is handled separately by the
    selector data, so it is never returned here.

    The ``pre-release`` key is present only when the gate passes: the newest
    pre-release is strictly greater than the newest stable (or there is no
    stable). A superseding stable omits ``pre-release`` entirely.
    """
    parsed = [v for v in (parse_tag(t) for t in tags) if v is not None]
    stables = [v for v in parsed if not v.is_prerelease]
    prereleases = [v for v in parsed if v.is_prerelease]

    newest_stable = _newest(stables)
    newest_pre = _newest(prereleases)

    channels: dict[str, str] = {}
    if newest_stable is not None:
        channels["stable"] = newest_stable.raw

    if newest_pre is not None:
        # Gate: only show the pre-release channel when it is ahead of stable.
        if newest_stable is None or compare(newest_pre, newest_stable) > 0:
            channels["pre-release"] = newest_pre.raw

    return channels
