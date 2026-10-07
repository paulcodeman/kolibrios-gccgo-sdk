#!/usr/bin/env python3
"""Exercise diagnostics, write, change and cleanup in the full original CLI."""
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[2]
arguments = [
    'python3', 'tooling/run-opencode-smoke.py',
    '--image', '/mnt/c/Users/Paul/Desktop/kolibrios/golang-kolibrios/.cache/kolibri/kolibri.img',
    '--binary', 'apps/opencode/opencode.kex',
    '--kernel', '.build-cache/opencode/kernel-fat-case-packed.mnt',
    '--console-object', '.build-cache/opencode/console-vterm.obj',
    '--service-binary', 'apps/tests/opencode/provider-tool-test-service/opencode-provider-tool-test-service.kex',
    '--environment-file', 'apps/tests/opencode/lsp-native-smoke/original-cli.env',
    '--stage-file', 'apps/tests/opencode/lsp-native-smoke/original-cli.env=SERVICE.KEX.env',
    '--stage-file', 'apps/tests/opencode/lsp-native-smoke/opencode-lsp-native-smoke.kex=bash-lsp.kex',
    '--stage-file', 'apps/tests/opencode/lsp-native-smoke/original-cli.json=.opencode.json',
    '--stage-file', 'apps/tests/opencode/lsp-native-smoke/fixture.sh=fixture.sh',
    '--arguments', '-c /hd0/1 -p "Check the Bash syntax diagnostic, fix the file, and check it again." -f json -q -d',
    '--memory', '512', '--collect-file', '.opencode/opencode.db',
    '--collect-file', '.opencode/opencode.db-wal', '--collect-file', '.opencode/debug.log',
    '--collect-file', 'fixture.sh', '--prefix', 'original-cli-native-lsp-ready',
    '--expect-marker', 'OPENCODE_ORIGINAL_LSP_RESULTS_PASS', '--timeout', '120',
]
raise SystemExit(subprocess.call(arguments, cwd=root))
