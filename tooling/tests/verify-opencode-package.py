#!/usr/bin/env python3
"""Verify release checksums, archive contents and corresponding native sources."""
from pathlib import Path
import hashlib
import json
import zipfile

root = Path(__file__).resolve().parents[2]
cache = root / '.build-cache/opencode'
digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
binary_hash = digest(root / 'apps/opencode/opencode.kex')
release = cache / 'release' / ('opencode-kolibrios-' + binary_hash[:12])
manifest = json.loads((release / 'manifest.json').read_text())
assert manifest['unchanged_application_files'] == 140
assert digest(release / 'opencode/opencode.kex') == binary_hash
assert digest(release / 'opencode/gosh.kex') == digest(root / 'apps/tools/gosh/gosh.kex')
assert digest(release / 'opencode/certs/ca-bundle.crt') == digest(root / 'third_party/ca-certificates/ca-bundle.crt')
for name, expected in manifest['files'].items():
    assert digest(release / name) == expected, name
for name, original in [('CONSOLE.OBJ', 'system/LIB/CONSOLE.OBJ'),
                       ('kernel-packed.mnt', 'system/KERNEL.MNT')]:
    assert digest(cache / 'release-verification' / name) == digest(release / original), name
archive_path = release.with_suffix('.zip')
with zipfile.ZipFile(archive_path) as archive:
    assert archive.testzip() is None
    expected_files = {release.name + '/' + name for name in manifest['files']}
    expected_files.add(release.name + '/manifest.json')
    assert {name for name in archive.namelist() if not name.endswith('/')} == expected_files
    for name in expected_files:
        assert archive.read(name) == (release.parent / name).read_bytes(), name
    assert not any(name.endswith('.key') or 'test-ca.pem' in name or 'SERVICE.KEX' in name
                   for name in archive.namelist())
report = {'archive_sha256': digest(archive_path), 'files_verified': len(manifest['files']),
          'kernel_and_console_rebuilt_from_distributed_sources': True,
          'public_ca_bundle_unchanged': True, 'test_services_and_private_keys_excluded': True}
(cache / 'release-verification/validation.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps(report, indent=2))
