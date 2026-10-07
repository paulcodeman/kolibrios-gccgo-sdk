#!/usr/bin/env python3
"""Exercise the original CLI's MCP discovery and tool call in KolibriOS."""
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
    '--environment-file', 'apps/tests/opencode/mcp-stdio-smoke/original-cli.env',
    '--stage-file', 'apps/tests/opencode/mcp-stdio-smoke/opencode-mcp-stdio-smoke.kex=mcp.kex',
    '--stage-file', 'apps/tests/opencode/mcp-stdio-smoke/original-cli.json=.opencode.json',
    '--stage-file', 'apps/tests/opencode/mcp-stdio-smoke/original-cli.env=SERVICE.KEX.env',
    '--arguments', '-c /hd0/1 -p "Call the native MCP echo tool with Unicode text." -f json -q -d',
    '--memory', '512', '--collect-file', '.opencode/opencode.db',
    '--collect-file', '.opencode/opencode.db-wal', '--collect-file', '.opencode/debug.log',
    '--prefix', 'original-cli-native-mcp-env',
    '--expect-marker', 'OPENCODE_ORIGINAL_MCP_RESULTS_PASS', '--timeout', '120',
]
raise SystemExit(subprocess.call(arguments, cwd=root))
