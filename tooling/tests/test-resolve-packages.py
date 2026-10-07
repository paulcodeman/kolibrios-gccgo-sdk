#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import subprocess
import shutil
import sys
import tempfile
import time
import unittest

TOOLING = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(TOOLING))
spec = importlib.util.spec_from_file_location('resolver', TOOLING / 'resolve-packages.py')
resolver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resolver)
from go_file_filter import list_package_go_files


class ResolverTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which('make') and shutil.which('gccgo-15'),
                         'requires GNU make and gccgo-15')
    def test_staged_entrypoint_uses_same_sources_for_discovery_and_compilation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            app, stage, package = root / 'folder-name', root / 'stage', root / 'packages/selected'
            for path in (app, stage, package):
                path.mkdir(parents=True)
            (app / 'main.go').write_text('package main\nimport _ "missing-local-import"\n')
            (stage / 'main.go').write_text(
                'package main\nimport "selected"\nvar Value = selected.Value\nfunc main() {}\n')
            (package / 'pkg.go').write_text('package selected\nconst Value = 7\n')
            (app / 'Makefile').write_text(
                'PROGRAM = actual-output-name\nGO_PACKAGE = main\n'
                'ROOT = ' + str(TOOLING.parent) + '\n'
                'APP_SOURCE_DIR = ' + str(stage) + '\n'
                'FIRST_PARTY_DIRS = ' + str(root / 'packages') + '\n'
                'THIRD_PARTY_DIRS =\nPACKAGE_DIRS =\n'
                'BUILD_CACHE_ROOT = cache\nGO = gccgo-15\n'
                'include $(ROOT)/tooling/kolibri-app.mk\n'
                'check: $(APP_OBJ)\n')
            result = subprocess.run(['make', '-s', 'check', 'print-output'], cwd=app,
                                    capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(result.stdout.strip(), str(app / 'actual-output-name.kex'))
            self.assertTrue((app / 'actual-output-name.gccgo.o').exists())
            self.assertIn('selected', (app / 'cache/actual-output-name.packages.mk').read_text())

    def test_missing_import_is_reported_before_dependency_metadata_is_written(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'main.go').write_text('package main\nimport _ "fixture/missing"\n')
            graph = root / 'packages.mk'
            graph.write_text('previous graph\n')
            result = subprocess.run([
                sys.executable, str(TOOLING / 'resolve-packages.py'),
                '--root', directory, '--stdlib', directory,
                '--app-dir', directory, '--make-deps', str(graph),
            ], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('missing Go package fixture/missing', result.stderr)
            self.assertEqual(graph.read_text(), 'previous graph\n')
            self.assertEqual(result.stdout, '')

    def test_foreign_platform_only_package_is_reported(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / 'fixture/foreign'
            package.mkdir(parents=True)
            (package / 'backend_windows.go').write_text('package foreign\n')
            (root / 'main.go').write_text('package main\nimport _ "fixture/foreign"\n')
            result = subprocess.run([
                sys.executable, str(TOOLING / 'resolve-packages.py'),
                '--root', directory, '--stdlib', directory, '--app-dir', directory,
            ], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('no sources for kolibrios/386: fixture/foreign', result.stderr)

    @unittest.skipUnless(shutil.which('make') and shutil.which('gccgo-15'),
                         'requires GNU make and gccgo-15')
    def test_make_rebuilds_changed_imports_without_rebuilding_unrelated_packages(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            app = root / 'app'
            app.mkdir()
            packages = root / 'packages'
            for name, body in {
                'alpha': 'package alpha\nconst Value = 11\n',
                'beta': 'package beta\nimport "fixture/alpha"\nconst Value = alpha.Value\n',
                'unused': 'package unused\nconst Value = 99\n',
            }.items():
                package = packages / 'fixture' / name
                package.mkdir(parents=True)
                (package / 'pkg.go').write_text(body)
            # Foreign platform sources must not create target dependencies.
            (packages / 'fixture/beta/extra_windows.go').write_text(
                'package beta\nimport _ "fixture/unused"\n')
            (app / 'main.go').write_text('package main\nimport _ "fixture/beta"\nfunc main() {}\n')
            (app / 'Makefile').write_text(
                'PROGRAM = incremental-check\nGO_PACKAGE = main\n'
                'ROOT = ' + str(TOOLING.parent) + '\n'
                'FIRST_PARTY_DIRS = ' + str(packages) + '\n'
                'THIRD_PARTY_DIRS =\nPACKAGE_DIRS = fixture/unused\n'
                'BUILD_CACHE_ROOT = cache\nGO = gccgo-15\n'
                'include $(ROOT)/tooling/kolibri-app.mk\n')
            targets = ['cache/pkg/fixture/beta.gox', 'cache/pkg/fixture/unused.gox']

            def build():
                result = subprocess.run(['make', '-s'] + targets, cwd=app,
                                        capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

            build()
            objects = {name: app / ('cache/pkg/fixture/' + name + '.gccgo.go.o')
                       for name in ('alpha', 'beta', 'unused')}
            first = {name: path.stat().st_mtime_ns for name, path in objects.items()}
            build()
            self.assertEqual(first, {name: path.stat().st_mtime_ns for name, path in objects.items()})
            time.sleep(0.02)
            (packages / 'fixture/alpha/pkg.go').write_text('package alpha\nconst Value = 13\n')
            build()
            second = {name: path.stat().st_mtime_ns for name, path in objects.items()}
            self.assertGreater(second['alpha'], first['alpha'])
            self.assertGreater(second['beta'], first['beta'])
            self.assertEqual(second['unused'], first['unused'])
            exports = subprocess.check_output([
                'readelf', '--string-dump=.go_export', str(app / targets[0])], text=True)
            self.assertRegex(exports, r'const Value[^\n]*\b13\b')
            graph = (app / 'cache/incremental-check.packages.mk').read_text()
            self.assertIn('PACKAGE_IMPORTS_fixture/beta := fixture/alpha\n', graph)
            self.assertIn('PACKAGE_RESOLVED_SOURCES_fixture/alpha := ' + str(packages / 'fixture/alpha/pkg.go') + '\n', graph)
            # AUTO_DEPS already selected the package files. A separate selector
            # invocation must be unnecessary even on a forced package rebuild.
            result = subprocess.run(['make', '-s', '-B', 'APP_SOURCES=main.go',
                                     'SELECT_GO_FILES=/bin/false'] + targets,
                                    cwd=app, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_zos_filename_is_not_selected_for_kolibrios(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'main.go').write_text('package sample\n')
            (root / 'sockcmsg_zos.go').write_text('package sample\n')
            selected = list_package_go_files(directory, 'kolibrios', '386', ['gccgo'])
            self.assertEqual([Path(path).name for path in selected], ['main.go'])

    def test_alias_and_side_effect_imports(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'main.go'
            source.write_text('''package main
import _ "embed"
import tea `example.com/tea`
import . "fmt"
import (
    "context"
    x "example.com/x"
)
''')
            self.assertEqual(resolver.parse_imports(str(source)),
                             ['embed', 'example.com/tea', 'fmt', 'context', 'example.com/x'])

    def test_target_filter_applies_to_entrypoint_imports(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            app = root / 'app'
            app.mkdir()
            for name in ('selected', 'windowsOnly', 'tagOnly'):
                package = root / name
                package.mkdir()
                (package / 'pkg.go').write_text('package ' + name + '\n')
            (app / 'main.go').write_text('package main\nimport _ "selected"\n')
            (app / 'extra_windows.go').write_text('package main\nimport "windowsOnly"\n')
            (app / 'extra.go').write_text('//go:build custom\n\npackage main\nimport "tagOnly"\n')
            result = subprocess.check_output([
                sys.executable, str(TOOLING / 'resolve-packages.py'),
                '--root', str(root), '--stdlib', str(root), '--app-dir', str(app),
            ], text=True)
            self.assertEqual(result.strip(), 'selected')


if __name__ == '__main__':
    unittest.main()
