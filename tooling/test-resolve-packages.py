#!/usr/bin/env python3
import importlib.util
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('resolve_packages', Path(__file__).with_name('resolve-packages.py'))
resolver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resolver)

class GoImports(unittest.TestCase):
    def parse(self, source):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'source.go'
            path.write_text(source)
            return resolver.parse_imports(str(path))

    def test_source_template_is_not_an_import(self):
        self.assertEqual(self.parse('''package migration
import "fmt"
const template = `package example
import (
  "self"
)
`
'''), ['fmt'])

    def test_comments_aliases_and_multiple_declarations(self):
        self.assertEqual(self.parse('''// import "false"
package sample
import alias "example.com/module"
/* import "false" */
import (
  _ "first"
  . `second`
  // import "false"
)
func main() { println("import \\\"false\\\"") }
'''), ['example.com/module', 'first', 'second'])


class SourceSelection(unittest.TestCase):
    def test_older_vendor_sources_rebuild_and_unchanged_selection_is_stable(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first, second = root / 'original.go', root / 'vendor.go'
            first.write_text('original')
            second.write_text('vendor')
            for source in (first, second):
                os.utime(source, (100, 100))
            artifacts = root / 'cache'
            artifacts.mkdir()
            obj = artifacts / 'sample.gccgo.go.o'
            obj.write_text('original')
            os.utime(obj, (200, 200))
            (artifacts / 'sample.generics.json').write_text(json.dumps({
                'sources': [str(first)],
                'source_sha256': {str(first): hashlib.sha256(first.read_bytes()).hexdigest()},
            }))
            target = ['kolibrios', '386', ['gccgo']]
            selection = resolver.source_selection(artifacts, 'sample', [str(first)], target)
            self.assertEqual(selection.stat().st_mtime_ns, obj.stat().st_mtime_ns)
            selection = resolver.source_selection(artifacts, 'sample', [str(second)], target)
            makefile = root / 'Makefile'
            makefile.write_text(str(obj) + ': ' + str(second) + ' ' + str(selection) + '\n\tcp ' + str(second) + ' ' + str(obj) + '\n')
            subprocess.run(['make', '-s', '-f', str(makefile)], check=True)
            self.assertEqual(obj.read_text(), 'vendor')
            modified = selection.stat().st_mtime_ns
            resolver.source_selection(artifacts, 'sample', [str(second)], target)
            self.assertEqual(selection.stat().st_mtime_ns, modified)
            self.assertEqual(subprocess.run(['make', '-q', '-f', str(makefile)]).returncode, 0)
            resolver.source_selection(artifacts, 'sample', [str(first), str(second)], target)
            self.assertEqual(subprocess.run(['make', '-q', '-f', str(makefile)]).returncode, 1)

    def test_modified_compiler_record_cannot_skip_rebuild(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'source.go'
            source.write_text('updated')
            obj = root / 'sample.gccgo.go.o'
            obj.write_text('old')
            os.utime(obj, (100, 100))
            (root / 'sample.generics.json').write_text(json.dumps({
                'sources': [str(source)], 'source_sha256': {str(source): 'stale'},
            }))
            selection = resolver.source_selection(root, 'sample', [str(source)], ['kolibrios', '386', ['gccgo']])
            self.assertGreater(selection.stat().st_mtime_ns, obj.stat().st_mtime_ns)

if __name__ == '__main__':
    unittest.main()
