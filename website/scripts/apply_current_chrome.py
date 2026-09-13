#!/usr/bin/env python3
"""Repair only the current pre-release shell; never rewrite release content.

Exact fragment checks deliberately reject unknown historical template variants.
The caller selects the moving pre-release channel, not every archived tag alias.
"""
from pathlib import Path
import re
import sys

CSS = """
/* AAASM-6099 current-channel compact chrome; release content is unchanged. */
.aa-navbar-title-compact { display: none; }
@media (max-width: 767px) {
  .aa-navbar-brand { flex-shrink: 0; white-space: nowrap; }
  .aa-navbar-title-full { display: none; }
  .aa-navbar-title-compact { display: inline; }
}
"""
TITLE = '{{ if .Title }}<h1 class="hx:text-center hx:mt-2 hx:text-4xl hx:font-bold hx:tracking-tight hx:text-slate-900 hx:dark:text-slate-100">{{ .Title }}</h1>{{ end }}'
TITLE_PATCH = '{{/* AAASM-6099: the release Markdown owns the single page H1. */}}'
TITLE_MODERN = TITLE.replace('{{ if .Title }}', '{{ if and .Title (ne .Params.showTitle false) }}')
LINK = '<a class="hx:flex hx:items-center hx:hover:opacity-75" href="{{ $logoLink }}">'
LINK_PATCH = '<a class="aa-navbar-brand hx:flex hx:items-center hx:hover:opacity-75" href="{{ $logoLink }}" aria-label="{{ .Site.Title }}">'
LABEL = '<span class="hx:mr-2 hx:font-extrabold hx:inline hx:select-none">{{- .Site.Title -}}</span>'
LABEL_PATCH = '<span class="aa-navbar-title-full hx:mr-2 hx:font-extrabold hx:inline hx:select-none">{{- .Site.Title -}}</span>\n    <span class="aa-navbar-title-compact hx:mr-2 hx:font-extrabold hx:select-none" aria-hidden="true">Go SDK</span>'

def replace_checked(text, before, after):
    if text.count(after) == 1 and before not in text:
        return text
    if text.count(before) != 1:
        raise ValueError('Unsupported current-channel template fragment; no files changed')
    return text.replace(before, after, 1)

def apply(root):
    root = Path(root)
    # Suppressing the template title is safe only when the release owns one H1.
    markdown = (root / 'docs/_index.md').read_text()
    if len(re.findall(r'^# ', markdown, re.M)) != 1:
        raise ValueError('Release entry does not own exactly one H1')
    home = root / 'website/layouts/home.html'
    navbar = root / 'website/layouts/_partials/navbar-title.html'
    css = root / 'website/assets/css/custom.css'
    home_text = home.read_text()
    if not (TITLE_MODERN in home_text and re.search(r'^showTitle:\s*false\s*$', markdown, re.M)):
        home_text = replace_checked(home_text, TITLE, TITLE_PATCH)
    navbar_text = replace_checked(navbar.read_text(), LINK, LINK_PATCH)
    navbar_text = replace_checked(navbar_text, LABEL, LABEL_PATCH)
    css_text = css.read_text()
    if not all(rule in css_text for rule in CSS.splitlines()[2:]):
        css_text += CSS
    # Validate every fragment before touching any file. Markdown and version
    # metadata are intentionally absent from these writes.
    home.write_text(home_text)
    navbar.write_text(navbar_text)
    css.write_text(css_text)

if __name__ == '__main__':
    apply(sys.argv[1])
