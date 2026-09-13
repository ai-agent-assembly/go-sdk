"""Compatibility, idempotence and immutable-content tests for the shell patch."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('chrome', Path(__file__).with_name('apply_current_chrome.py'))
chrome = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(chrome)
ROOT = Path(__file__).resolve().parents[2]
TAG = 'v0.0.1-rc.6'
FILES = ['docs/_index.md', 'website/layouts/home.html',
         'website/layouts/_partials/navbar-title.html', 'website/assets/css/custom.css',
         'website/data/versions.toml']

class CurrentChromeTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.original = {}
        for name in FILES:
            content = subprocess.check_output(['git', 'show', f'{TAG}:{name}'], cwd=ROOT)
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(content)
            self.original[name] = content

    def test_exact_release_preserves_content_and_versions(self):
        chrome.apply(self.root)
        for name in ['docs/_index.md', 'website/data/versions.toml']:
            self.assertEqual((self.root / name).read_bytes(), self.original[name])
        navbar = (self.root / FILES[2]).read_text()
        self.assertIn(chrome.LINK_PATCH, navbar)
        self.assertIn('partial "version-selector.html" .', navbar)
        first = {name: (self.root / name).read_bytes() for name in FILES}
        chrome.apply(self.root)
        self.assertEqual(first, {name: (self.root / name).read_bytes() for name in FILES})

    def test_unknown_template_fails_without_partial_write(self):
        (self.root / FILES[2]).write_text('unsupported future navbar')
        before = {name: (self.root / name).read_bytes() for name in FILES}
        with self.assertRaises(ValueError):
            chrome.apply(self.root)
        self.assertEqual(before, {name: (self.root / name).read_bytes() for name in FILES})

    def test_missing_content_heading_is_not_suppressed(self):
        (self.root / FILES[0]).write_text('No authored heading')
        with self.assertRaises(ValueError):
            chrome.apply(self.root)
        self.assertEqual((self.root / FILES[1]).read_bytes(), self.original[FILES[1]])

    def test_future_release_with_approved_source_fix_is_idempotent(self):
        for name in FILES:
            (self.root / name).write_bytes((ROOT / name).read_bytes())
        before = {name: (self.root / name).read_bytes() for name in FILES}
        chrome.apply(self.root)
        self.assertEqual(before, {name: (self.root / name).read_bytes() for name in FILES})

if __name__ == '__main__':
    unittest.main()
