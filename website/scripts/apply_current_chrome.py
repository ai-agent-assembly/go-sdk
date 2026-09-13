#!/usr/bin/env python3
"""Repair only the current pre-release shell; never rewrite release content.

Exact fragment checks deliberately reject unknown historical template variants.
The caller selects the moving pre-release channel, not every archived tag alias.
"""
from pathlib import Path
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
AUTHORED_H1 = r'{{ $hasAuthoredH1 := gt (len (findRE `(?i)<h1(?:\s|>)` .Content 1)) 0 }}'
HOME_TITLE_PATCH = AUTHORED_H1 + '\n        ' + TITLE.replace(
    '{{ if .Title }}', '{{ if and .Title (ne .Params.showTitle false) (not $hasAuthoredH1) }}')
SINGLE_TITLE_PATCH = AUTHORED_H1 + '\n        ' + TITLE.replace(
    '{{ if .Title }}', '{{ if and .Title (not $hasAuthoredH1) }}')
LIST_TITLE = '{{ if .Title }}<h1>{{ .Title }}</h1>{{ end }}'
LIST_TITLE_PATCH = AUTHORED_H1 + '\n          ' + LIST_TITLE.replace(
    '{{ if .Title }}', '{{ if and .Title (not $hasAuthoredH1) }}')
TOC = '{{ partial "toc.html" . }}'
TOC_PATCH = '{{ partial "custom/toc-if-entries.html" . }}'
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
    home = root / 'website/layouts/home.html'
    listing = root / 'website/layouts/list.html'
    single = root / 'website/layouts/single.html'
    toc = root / 'website/layouts/_partials/custom/toc-if-entries.html'
    navbar = root / 'website/layouts/_partials/navbar-title.html'
    css = root / 'website/assets/css/custom.css'
    home_text = replace_checked(home.read_text(), TITLE, HOME_TITLE_PATCH)
    list_text = replace_checked(listing.read_text(), LIST_TITLE, LIST_TITLE_PATCH)
    single_text = replace_checked(single.read_text(), TITLE, SINGLE_TITLE_PATCH)
    home_text = replace_checked(home_text, TOC, TOC_PATCH)
    list_text = replace_checked(list_text, TOC, TOC_PATCH)
    single_text = replace_checked(single_text, TOC, TOC_PATCH)
    helper_bytes = (Path(__file__).resolve().parents[1] / 'layouts/_partials/custom/toc-if-entries.html').read_bytes()
    if toc.exists() and toc.read_bytes() != helper_bytes:
        raise ValueError('Unsupported current-channel TOC helper; no files changed')
    navbar_text = replace_checked(navbar.read_text(), LINK, LINK_PATCH)
    navbar_text = replace_checked(navbar_text, LABEL, LABEL_PATCH)
    css_text = css.read_text()
    if not all(rule in css_text for rule in CSS.splitlines()[2:]):
        css_text += CSS
    # Validate every fragment before touching any file. Markdown and version
    # metadata are intentionally absent from these writes.
    home.write_text(home_text)
    listing.write_text(list_text)
    single.write_text(single_text)
    toc.parent.mkdir(parents=True, exist_ok=True)
    toc.write_bytes(helper_bytes)
    navbar.write_text(navbar_text)
    css.write_text(css_text)

if __name__ == '__main__':
    apply(sys.argv[1])
