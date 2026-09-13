#!/usr/bin/env python3
"""Check the actual Hugo HTML shell, not Markdown regexes or source templates."""
from pathlib import Path
import re
import sys

if len(sys.argv) != 2:
    raise SystemExit('usage: test_rendered_headings.py <version-output-dir>')
site = Path(sys.argv[1])
pages = {
    'index.html': 'go-sdk · AI Agent Assembly',
    'quick-start/index.html': 'Quick Start',
    'api-reference/index.html': 'API Reference',
    'guides/index.html': 'Guides',
    'guides/govern-an-agents-tools/index.html': "Govern an agent's tools",
    'examples/index.html': 'Examples',
}
for relative, title in pages.items():
    html = (site / relative).read_text()
    headings = re.findall(r'<h1(?:\s[^>]*)?>(.*?)</h1>', html, re.I | re.S)
    if len(headings) != 1:
        raise AssertionError(f'{relative}: expected one visible H1; found {len(headings)}')
    rendered = re.sub(r'<[^>]+>', '', headings[0])
    if title not in rendered:
        raise AssertionError(f'{relative}: wrong surviving article H1: {rendered!r}')

for relative in ('guides/index.html', 'examples/index.html'):
    html = (site / relative).read_text()
    if 'aria-label="Table of contents"' in html:
        raise AssertionError(f'{relative}: empty category TOC shell remains')

quickstart = (site / 'quick-start/index.html').read_text()
if 'aria-label="Table of contents"' not in quickstart:
    raise AssertionError('quick-start lost its useful article TOC')
if 'Registration' not in quickstart:
    raise AssertionError('quick-start lost the current registration warning')

print(f'PASS: one authored H1, useful TOC, no empty category TOC ({site})')
