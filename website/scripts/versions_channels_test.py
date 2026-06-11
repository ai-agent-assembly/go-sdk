#!/usr/bin/env python3
"""Unit tests for the pre-release channel gate (AAASM-2754).

Run: ``python3 -m unittest website/scripts/versions_channels_test.py`` (or
``python3 website/scripts/versions_channels_test.py``). Pure stdlib, no Hugo.
"""

from __future__ import annotations

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from versions_channels import compare, compute_channels, parse_tag  # noqa: E402


class TestSemverComparator(unittest.TestCase):
    def _lt(self, a: str, b: str) -> None:
        va, vb = parse_tag(a), parse_tag(b)
        assert va is not None and vb is not None
        self.assertEqual(compare(va, vb), -1, f"{a} should be < {b}")
        self.assertEqual(compare(vb, va), 1, f"{b} should be > {a}")

    def test_prerelease_below_release(self) -> None:
        self._lt("v0.1.0-rc.1", "v0.1.0")

    def test_alpha_beta_rc_order(self) -> None:
        self._lt("v0.1.0-alpha.5", "v0.1.0-beta.1")
        self._lt("v0.1.0-beta.2", "v0.1.0-rc.1")

    def test_numeric_identifiers_numeric(self) -> None:
        self._lt("v0.1.0-alpha.5", "v0.1.0-alpha.6")
        # 2 < 10 numerically, not lexically.
        self._lt("v0.1.0-alpha.2", "v0.1.0-alpha.10")

    def test_numeric_below_alphanumeric(self) -> None:
        self._lt("v1.0.0-1", "v1.0.0-alpha")

    def test_core_version_ordering(self) -> None:
        self._lt("v0.0.2", "v0.1.0")
        self._lt("v0.1.0", "v0.2.0-alpha.1")

    def test_equal(self) -> None:
        va = parse_tag("v0.1.0")
        assert va is not None
        self.assertEqual(compare(va, va), 0)


class TestChannelGate(unittest.TestCase):
    # Scenario 1: pre-releases ahead of the newest stable -> shown.
    SET1 = [
        "v0.0.2",
        "v0.1.0-alpha.5",
        "v0.1.0-alpha.6",
        "v0.1.0-beta.1",
        "v0.1.0-beta.2",
        "v0.1.0-rc.1",
    ]

    def test_scenario_1_pre_ahead_of_stable(self) -> None:
        channels = compute_channels(self.SET1)
        self.assertEqual(channels.get("stable"), "v0.0.2")
        self.assertEqual(channels.get("pre-release"), "v0.1.0-rc.1")

    def test_scenario_2_superseding_stable_drops_pre(self) -> None:
        channels = compute_channels(self.SET1 + ["v0.1.0"])
        self.assertEqual(channels.get("stable"), "v0.1.0")
        self.assertNotIn("pre-release", channels)

    def test_scenario_3_newer_pre_returns_after_stable(self) -> None:
        channels = compute_channels(self.SET1 + ["v0.1.0", "v0.2.0-alpha.1"])
        self.assertEqual(channels.get("stable"), "v0.1.0")
        self.assertEqual(channels.get("pre-release"), "v0.2.0-alpha.1")

    def test_no_stable_shows_pre(self) -> None:
        channels = compute_channels(["v0.1.0-alpha.1", "v0.1.0-beta.1"])
        self.assertNotIn("stable", channels)
        self.assertEqual(channels.get("pre-release"), "v0.1.0-beta.1")

    def test_pre_equal_core_as_stable_is_behind(self) -> None:
        # v0.1.0-rc.1 < v0.1.0, so a stable v0.1.0 hides it.
        channels = compute_channels(["v0.1.0", "v0.1.0-rc.1"])
        self.assertEqual(channels.get("stable"), "v0.1.0")
        self.assertNotIn("pre-release", channels)

    def test_empty(self) -> None:
        self.assertEqual(compute_channels([]), {})


if __name__ == "__main__":
    unittest.main(verbosity=2)
