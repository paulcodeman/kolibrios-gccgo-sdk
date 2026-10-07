#!/usr/bin/env python3
"""Check target tags and portable platform selection without rewriting sources."""
from pathlib import Path
import json
import tempfile
import unittest

from go_file_filter import default_tag_set, list_package_go_files


class PlatformSelection(unittest.TestCase):
    def test_declared_platform_replacement_keeps_original(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'original.go').write_text('package target\n')
            (root / 'native_kolibrios.go').write_text('package target\n')
            (root / 'optional.go').write_text('//go:build native_feature\n\npackage target\n')
            config = {'reason': 'native OS backend', 'tags': ['native_feature'],
                      'replace': {'original.go': 'native_kolibrios.go'}}
            (root / 'kolibrios.sources.json').write_text(json.dumps(config))
            names = lambda os: {Path(p).name for p in list_package_go_files(str(root), os, '386')}
            self.assertEqual(names('kolibrios'), {'native_kolibrios.go', 'optional.go'})
            self.assertEqual(names('linux'), {'original.go'})
            self.assertEqual((root / 'original.go').read_text(), 'package target\n')
            (root / 'native_kolibrios.go').unlink()
            with self.assertRaises(ValueError):
                names('kolibrios')

    def test_unix_tag(self):
        self.assertNotIn('unix', default_tag_set('kolibrios', '386'))
        self.assertIn('unix', default_tag_set('linux', '386'))
        self.assertIn('linux', default_tag_set('android', '386'))
        self.assertNotIn('unix', default_tag_set('windows', '386'))

    def test_portable_fallback_and_native_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for filename, condition in (
                ('base.go', None),
                ('os_unix.go', 'unix'),
                ('os_other.go', '!unix && !windows'),
                ('native_kolibrios.go', 'kolibrios'),
                ('generic_unix.go', None),
                ('generic_unix_386.go', None),
                ('sys_linux_386.go', None),
                ('sys_linux_amd64.go', None),
            ):
                tag = '//go:build ' + condition + '\n\n' if condition else ''
                (root / filename).write_text(tag + 'package target\n')
            selected = lambda os: {Path(p).name for p in list_package_go_files(str(root), os, '386', ['gccgo'])}
            shared = {'base.go', 'generic_unix.go', 'generic_unix_386.go'}
            self.assertEqual(selected('kolibrios'), shared | {'os_other.go', 'native_kolibrios.go'})
            self.assertEqual(selected('linux'), shared | {'os_unix.go', 'sys_linux_386.go'})


if __name__ == '__main__':
    unittest.main()
