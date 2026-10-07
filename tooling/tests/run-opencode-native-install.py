#!/usr/bin/env python3
"""Boot the release layout with its HOME, shell and public certificate bundle."""
from pathlib import Path
import hashlib
import subprocess
import tempfile

root = Path(__file__).resolve().parents[2]
binary = root / 'apps/opencode/opencode.kex'
revision = hashlib.sha256(binary.read_bytes()).hexdigest()[:12]
release = root / '.build-cache/opencode/release' / ('opencode-kolibrios-' + revision)
with tempfile.TemporaryDirectory(prefix='opencode-install-fixture-') as temporary:
    environment = Path(temporary) / 'installed.env'
    # Retain the distributed public bundle. The separate test-only directory
    # trusts the loopback inference certificate; it is never put in the release.
    environment.write_text((release / 'opencode/opencode.kex.env').read_text() +
                           'LOCAL_ENDPOINT=https://10.0.2.2:18443/v1\n'
                           'SSL_CERT_DIR=/hd0/1/test-certs\nOPENCODE_DEV_DEBUG=true\n')
    arguments = ['python3', 'tooling/run-opencode-smoke.py',
                 '--image', '/mnt/c/Users/Paul/Desktop/kolibrios/golang-kolibrios/.cache/kolibri/kolibri.img',
                 '--binary', str(release / 'opencode/opencode.kex'),
                 '--binary-relative-path', 'opencode/opencode.kex',
                 '--kernel', str(release / 'system/KERNEL.MNT'),
                 '--console-object', str(release / 'system/LIB/CONSOLE.OBJ'),
                 '--environment-file', str(environment),
                 '--stage-file', '.build-cache/opencode/real-ai-server/tls/ca.pem=test-certs/test-ca.pem',
                 '--stage-file', 'apps/tests/opencode/tool-fixture/fixture.txt=project/fixture.txt',
                 '--arguments', '-c /hd0/1/project -p "Say hello in one short sentence." -f json -q -d',
                 '--memory', '512', '--user-network', '--timeout', '300',
                 '--prefix', 'original-cli-installed-package-tls',
                 '--collect-file', 'project/.opencode/opencode.db',
                 '--collect-file', 'project/.opencode/opencode.db-wal',
                 '--collect-file', 'project/.opencode/debug.log']
    for path in (release / 'opencode').rglob('*'):
        if path.is_file() and path.name not in ('opencode.kex', 'opencode.kex.env'):
            arguments.extend(['--stage-file', str(path) + '=' + path.relative_to(release).as_posix()])
    placeholder = Path(temporary) / 'empty.txt'
    placeholder.write_text('')
    for directory in ('opencode/tmp', 'opencode/.config/opencode'):
        arguments.extend(['--stage-file', str(placeholder) + '=' + directory + '/.keep'])
    result = subprocess.call(arguments, cwd=root)
raise SystemExit(result)
