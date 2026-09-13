#!/usr/bin/env python3
"""Correct the audited rc5 entry shell without changing archived article content."""
from pathlib import Path
import sys

from apply_current_chrome import AUTHORED_H1, TITLE, replace_checked

PATCH = AUTHORED_H1 + '\n        ' + TITLE.replace(
    '{{ if .Title }}', '{{ if and .Title (not $hasAuthoredH1) }}')


def apply(root):
    home = Path(root) / 'website/layouts/home.html'
    corrected = replace_checked(home.read_text(), TITLE, PATCH)
    home.write_text(corrected)


if __name__ == '__main__':
    apply(sys.argv[1])
