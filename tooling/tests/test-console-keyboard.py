#!/usr/bin/env python3
"""Check native terminal key bytes and legacy console editing in QEMU."""
import argparse
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[2]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--image', required=True, type=Path)
parser.add_argument('--console', required=True, type=Path)
parser.add_argument('--prefix', default='console-backspace-qualified')
args = parser.parse_args()
prefix = args.prefix
subprocess.run([
    'python3', str(root / 'tooling/run-opencode-smoke.py'),
    '--image', str(args.image.resolve()),
    '--binary', str(root / 'apps/tests/console-keyboard/console-keyboard.kex'),
    '--console-object', str(args.console.resolve()),
    '--input-script', str(root / 'tooling/tests/console-keyboard-input.json'),
    '--collect-file', 'KEYBOARD.txt', '--memory', '512', '--timeout', '48',
    '--prefix', prefix,
], cwd=root, check=True)
cache = root / '.build-cache/opencode'
report = (cache / (prefix + '-files/KEYBOARD.txt')).read_text()
assert report.endswith('PASS: terminal editing and legacy console input\n'), report
execution = json.loads((cache / (prefix + '-run.json')).read_text())
validation = {'native_keyboard_and_legacy_editing_passed': True,
              'execution': execution, 'report': report}
(cache / (prefix + '-files/validation.json')).write_text(json.dumps(validation, indent=2) + '\n')
print(report)
