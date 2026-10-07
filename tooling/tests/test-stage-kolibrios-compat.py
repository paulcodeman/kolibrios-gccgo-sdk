#!/usr/bin/env python3
"""Verify shared adapter staging, version checks and preservation of local edits."""
import hashlib
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('compat', ROOT / 'tooling/stage-kolibrios-compat.py')
compat = importlib.util.module_from_spec(spec)
spec.loader.exec_module(compat)


class StagingTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / 'source'
        self.vendor = self.root / 'vendor'
        self.record = self.root / 'record.json'
        self.module = 'example.org/library'
        self.relative = self.module + '/backend_kolibrios.go'
        path = self.source / self.relative
        path.parent.mkdir(parents=True)
        path.write_bytes(b'package library\n')
        self.manifest = {
            'tested_modules': {self.module: 'v1.2.3'},
            'adapters': {self.relative: {'module': self.module, 'sha256': hashlib.sha256(path.read_bytes()).hexdigest()}},
        }
        self.write_manifest()
        self.vendor.mkdir()
        (self.vendor / 'modules.txt').write_text('# ' + self.module + ' v1.2.3\n')

    def write_manifest(self):
        (self.source / 'ADAPTERS.json').write_text(json.dumps(self.manifest))

    def stage(self, **kwargs):
        return compat.stage(self.vendor, self.record, self.source, **kwargs)

    def test_preserves_upstream_and_checks_without_writing(self):
        upstream = self.vendor / self.module / 'api.go'
        upstream.parent.mkdir(parents=True)
        upstream.write_bytes(b'original source\n')
        self.assertEqual(self.stage(), 1)
        before = {p: (p.read_bytes(), p.stat().st_mtime_ns) for p in self.vendor.rglob('*') if p.is_file()}
        self.assertEqual(self.stage(check=True), 1)
        self.assertEqual(before, {p: (p.read_bytes(), p.stat().st_mtime_ns) for p in before})
        self.assertEqual(upstream.read_bytes(), b'original source\n')

    def test_modified_file_blocks_whole_selection(self):
        self.stage()
        (self.vendor / self.relative).write_bytes(b'local edit\n')
        other = 'example.org/library/aaa_kolibrios.go'
        (self.source / other).write_bytes(b'new adapter\n')
        self.manifest['adapters'][other] = {'module': self.module, 'sha256': hashlib.sha256(b'new adapter\n').hexdigest()}
        self.write_manifest()
        with self.assertRaisesRegex(RuntimeError, 'refusing to overwrite'):
            self.stage()
        self.assertFalse((self.vendor / other).exists())
        self.assertEqual((self.vendor / self.relative).read_bytes(), b'local edit\n')

    def test_previously_recorded_adapter_can_be_updated(self):
        self.stage()
        data = b'package library\n// updated\n'
        (self.source / self.relative).write_bytes(data)
        self.manifest['adapters'][self.relative]['sha256'] = hashlib.sha256(data).hexdigest()
        self.write_manifest()
        self.stage()
        self.assertEqual((self.vendor / self.relative).read_bytes(), data)

    def test_version_and_fingerprint_mismatches_do_not_write(self):
        (self.vendor / 'modules.txt').write_text('# ' + self.module + ' v9.0.0\n')
        with self.assertRaisesRegex(RuntimeError, 'version mismatch'):
            self.stage()
        (self.vendor / 'modules.txt').write_text('# ' + self.module + ' v1.2.3\n')
        (self.source / self.relative).write_bytes(b'unrecorded change\n')
        with self.assertRaisesRegex(RuntimeError, 'fingerprint mismatch'):
            self.stage()
        self.assertFalse((self.vendor / self.relative).exists())
        self.assertFalse(self.record.exists())

    def test_absent_modules_are_skipped(self):
        (self.vendor / 'modules.txt').write_text('# example.org/other v1.0.0\n')
        self.assertEqual(self.stage(), 0)
        self.assertFalse((self.vendor / self.relative).exists())

    def test_escape_and_unknown_module_are_rejected(self):
        self.manifest['adapters']['../escape'] = self.manifest['adapters'].pop(self.relative)
        self.write_manifest()
        with self.assertRaisesRegex(RuntimeError, 'invalid adapter path'):
            self.stage()
        with self.assertRaisesRegex(RuntimeError, 'unknown adapter modules'):
            self.stage(modules=['example.org/unknown'])


@unittest.skipUnless(shutil.which('make'), 'GNU make is required')
class MakeIntegrationTests(unittest.TestCase):
    def run_make(self, build_kind, version):
        with tempfile.TemporaryDirectory() as temporary:
            app = Path(temporary)
            vendor = app / 'vendor'
            module = 'github.com/muesli/cancelreader'
            upstream = vendor / module / 'cancelreader.go'
            upstream.parent.mkdir(parents=True)
            upstream.write_text('package cancelreader\n')
            (upstream.parent / 'cancelreader_default.go').write_text('package cancelreader\n')
            (vendor / 'modules.txt').write_text('# ' + module + ' ' + version + '\n')
            package = 'main' if build_kind == 'app' else 'fixture'
            (app / 'main.go').write_text('package ' + package + '\nimport _ "' + module + '"\n')
            (app / 'Makefile').write_text(
                'PROGRAM = fixture\nBUILD_CACHE_ROOT = .build-cache\nGO_PACKAGE = ' + package + '\nROOT = ' + str(ROOT) + '\n'
                'include $(ROOT)/tooling/kolibri-' + build_kind + '.mk\n'
                '.PHONY: inspect-vendor\ninspect-vendor:\n'
                '\t@printf "%s\\n" "$(SDK_VENDOR_ROOT)" "$(call FIND_PACKAGE_DIR,' + module + ')"\n')
            result = subprocess.run(['make', '--no-print-directory', 'inspect-vendor'], cwd=app, capture_output=True, text=True)
            if version == 'v0.2.2':
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertEqual(result.stdout.strip().splitlines(), [str(vendor), str(vendor / module)])
                if build_kind == 'app':
                    # Package metadata belongs to the temporary fixture cache.
                    graph = app / '.build-cache/fixture.packages.mk'
                    self.assertIn(str(vendor / module / 'cancelreader_kolibrios.go'), graph.read_text())
                self.assertTrue((vendor / module / 'cancelreader_kolibrios.go').is_file())
                self.assertTrue((vendor / module / 'kolibrios.sources.json').is_file())
                self.assertEqual(upstream.read_text(), 'package cancelreader\n')
            else:
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('adapter version mismatch', result.stderr)
                self.assertFalse((vendor / module / 'cancelreader_kolibrios.go').exists())

    def test_application_and_library_discover_vendor_automatically(self):
        for kind in ('app', 'lib'):
            with self.subTest(kind=kind):
                self.run_make(kind, 'v0.2.2')

    def test_application_and_library_report_version_mismatch(self):
        for kind in ('app', 'lib'):
            with self.subTest(kind=kind):
                self.run_make(kind, 'v9.0.0')


if __name__ == '__main__':
    unittest.main()
