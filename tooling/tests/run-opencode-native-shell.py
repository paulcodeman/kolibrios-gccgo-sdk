#!/usr/bin/env python3
"""Exercise original shell, grep fallback, glob and ls through the full CLI."""
from pathlib import Path
import subprocess

root=Path(__file__).resolve().parents[2]
arguments=[
 'python3','tooling/run-opencode-smoke.py',
 '--image','/mnt/c/Users/Paul/Desktop/kolibrios/golang-kolibrios/.cache/kolibri/kolibri.img',
 '--binary','apps/opencode/opencode.kex',
 '--kernel','.build-cache/opencode/kernel-fat-case-packed.mnt',
 '--console-object','.build-cache/opencode/console-vterm.obj',
 '--service-binary','apps/tests/opencode/provider-tool-test-service/opencode-provider-tool-test-service.kex',
 '--environment-file','apps/tests/opencode/shell-native-smoke/original-cli.env',
 '--stage-file','apps/tests/opencode/shell-native-smoke/original-cli.env=SERVICE.KEX.env',
 '--stage-file','apps/tools/gosh/gosh.kex=gosh.kex',
 '--stage-file','apps/tests/opencode/shell-native-smoke/original-cli.json=.opencode.json',
 '--stage-file','apps/tests/opencode/tool-fixture/fixture.txt=fixture.txt',
 '--arguments','-c /hd0/1 -p "Run the shell and search tools, checking persistent state and exit status." -f json -q -d',
 '--memory','512','--collect-file','.opencode/opencode.db',
 '--collect-file','.opencode/opencode.db-wal','--collect-file','.opencode/debug.log',
 '--prefix','original-cli-native-shell-search-case',
 '--expect-marker','OPENCODE_ORIGINAL_SHELL_TOOLS_PASS','--timeout','120',
]
raise SystemExit(subprocess.call(arguments,cwd=root))
