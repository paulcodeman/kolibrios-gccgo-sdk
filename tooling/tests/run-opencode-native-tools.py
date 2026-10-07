#!/usr/bin/env python3
"""Exercise the original view/write/edit tool pipeline on native FAT."""
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
    '--environment-file', 'apps/tests/opencode/provider-tool-test-service/test.env',
    '--stage-file', 'apps/tests/opencode/tool-fixture/fixture.txt=fixture.txt',
    '--arguments', '-c /hd0/1 -p "Read fixture.txt, create generated.txt, read it, replace its line, and verify the result." -f json -q -d',
    '--memory', '512', '--collect-file', 'generated.txt',
    '--collect-file', '.opencode/opencode.db', '--collect-file', '.opencode/opencode.db-wal',
    '--collect-file', '.opencode/debug.log', '--prefix', 'original-cli-native-tools',
    '--expect-marker', 'OPENCODE_ORIGINAL_TOOL_RESULTS_PASS', '--timeout', '120',
]
raise SystemExit(subprocess.call(arguments, cwd=root))
