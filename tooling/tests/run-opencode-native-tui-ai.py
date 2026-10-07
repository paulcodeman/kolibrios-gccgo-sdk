#!/usr/bin/env python3
"""Send a keyboard prompt through original Bubble Tea TUI over verified TLS."""
from pathlib import Path
import subprocess
import urllib.request

root = Path(__file__).resolve().parents[2]
with urllib.request.urlopen('http://127.0.0.1:18081/health', timeout=5) as response:
    assert response.status == 200
arguments = [
    'python3', 'tooling/run-opencode-smoke.py',
    '--image', '/mnt/c/Users/Paul/Desktop/kolibrios/golang-kolibrios/.cache/kolibri/kolibri.img',
    '--binary', 'apps/opencode/opencode.kex',
    '--service-binary', 'apps/tests/opencode/clipboard-native-service/opencode-clipboard-native-service.kex',
    '--kernel', '.build-cache/opencode/kernel-fat-case-packed.mnt',
    '--console-object', '.build-cache/opencode/console-vterm.obj',
    '--environment-file', '.build-cache/opencode/real-ai-server/tls/test.env',
    '--stage-file', '.build-cache/opencode/real-ai-server/tls/ca.pem=test-ca.pem',
    '--stage-file', 'apps/tests/opencode/tool-fixture/fixture.txt=project/fixture.txt',
    '--arguments', '-c /hd0/1 -d', '--memory', '512', '--disk-size-mib', '64', '--user-network',
    '--collect-file', '.opencode/opencode.db', '--collect-file', '.opencode/opencode.db-wal',
    '--collect-file', '.opencode/debug.log',
    '--prefix', 'original-cli-tui-verified-tls-native',
    '--input-script', 'tooling/tests/opencode-tui-clipboard-input.json', '--timeout', '300',
]
raise SystemExit(subprocess.call(arguments, cwd=root))
