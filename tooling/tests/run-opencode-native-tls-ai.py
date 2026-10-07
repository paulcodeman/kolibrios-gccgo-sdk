#!/usr/bin/env python3
"""Run the original native CLI against actual host inference over verified TLS."""
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
    '--kernel', '.build-cache/opencode/kernel-fat-case-packed.mnt',
    '--console-object', '.build-cache/opencode/console-vterm.obj',
    '--environment-file', '.build-cache/opencode/real-ai-server/tls/test.env',
    '--stage-file', '.build-cache/opencode/real-ai-server/tls/ca.pem=test-ca.pem',
    '--arguments', '-c /hd0/1 -p "Say hello in one short sentence." -f json -q -d',
    '--memory', '512', '--disk-size-mib', '64', '--user-network',
    '--collect-file', '.opencode/opencode.db', '--collect-file', '.opencode/opencode.db-wal',
    '--collect-file', '.opencode/debug.log',
    '--prefix', 'original-cli-real-ai-verified-tls-native', '--timeout', '180',
]
raise SystemExit(subprocess.call(arguments, cwd=root))
