#!/usr/bin/env python3
"""Check the actual Hugo HTML shell, not Markdown regexes or source templates."""
from pathlib import Path
from html import unescape
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
    rendered = unescape(re.sub(r'<[^>]+>', '', headings[0])).replace('’', "'")
    if title not in rendered:
        raise AssertionError(f'{relative}: wrong surviving article H1: {rendered!r}')

if 'aria-label="Table of contents"' in (site / 'guides/index.html').read_text():
    raise AssertionError('guides/index.html: empty category TOC shell remains')

quickstart = (site / 'quick-start/index.html').read_text()
if 'aria-label="Table of contents"' not in quickstart:
    raise AssertionError('quick-start lost its useful article TOC')
if 'aria-label="Table of contents"' not in (site / 'examples/index.html').read_text():
    raise AssertionError('examples lost its useful category TOC')
if 'agent registration is not reachable' not in quickstart.lower():
    raise AssertionError('quick-start lost the current registration warning')

print(f'PASS: one authored H1, useful TOC, no empty category TOC ({site})')
