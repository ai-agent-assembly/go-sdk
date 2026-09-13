#!/usr/bin/env python3
"""The historical heading correction is exact and touches only the shell."""
import sys
import tempfile
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from apply_rc5_heading import apply, PATCH, TITLE


class Rc5HeadingTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / 'website/layouts').mkdir(parents=True)
        (self.root / 'website/content').mkdir(parents=True)
        self.home = self.root / 'website/layouts/home.html'
        self.article = self.root / 'website/content/_index.md'
        self.home.write_text('before\n        ' + TITLE + '\nafter\n')
        self.article.write_text('# Authored historical title\n\nArchived body.\n')

    def test_only_exact_shell_changes_and_is_idempotent(self):
        apply(self.root)
        self.assertIn(PATCH, self.home.read_text())
        self.assertEqual(self.article.read_text(), '# Authored historical title\n\nArchived body.\n')
        first = self.home.read_bytes()
        apply(self.root)
        self.assertEqual(self.home.read_bytes(), first)

    def test_unknown_shell_fails_without_writes(self):
        self.home.write_text('different archive template')
        with self.assertRaises(ValueError):
            apply(self.root)
        self.assertEqual(self.home.read_text(), 'different archive template')


if __name__ == '__main__':
    unittest.main()
