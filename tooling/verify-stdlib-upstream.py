#!/usr/bin/env python3
"""Verify imported production Go sources against their upstream SHA-256 records."""

import hashlib
import json
from pathlib import Path


def main():
    stdlib = Path(__file__).resolve().parent.parent / 'stdlib'
    manifest = json.loads((stdlib / 'OPENCODE_UPSTREAM.json').read_text())
    exact = adapted = 0
    failures = []
    for pkg, record in manifest['packages'].items():
        for name, upstream_hash in record['files'].items():
            relative = pkg + '/' + name
            path = stdlib / relative
            if not path.is_file():
                failures.append('missing: ' + relative)
                continue
            actual = hashlib.sha256(path.read_bytes()).hexdigest()
            adaptation = manifest.get('adaptations', {}).get(relative)
            if actual == upstream_hash:
                exact += 1
            elif adaptation and actual == adaptation['sha256']:
                adapted += 1
                print('ADAPTED ' + relative + ': ' + adaptation['reason'])
            else:
                failures.append('unexpected source changes: ' + relative)
    checked_support = 0
    for relative, record in manifest.get('support_sources', {}).items():
        if not isinstance(record, dict):
            continue  # Legacy descriptions of individual extracted functions.
        path = stdlib / relative
        if not path.is_file():
            failures.append('missing support source: ' + relative)
        elif hashlib.sha256(path.read_bytes()).hexdigest() != record['sha256']:
            failures.append('unexpected support source changes: ' + relative)
        elif record.get('upstream_sha256', record['sha256']) != record['sha256'] and not record.get('adaptation'):
            failures.append('undeclared support adaptation: ' + relative)
        else:
            checked_support += 1
    for failure in failures:
        print('FAIL ' + failure)
    print(f'{exact} production files match upstream exactly; {adapted} declared platform adaptations')
    print(f'{checked_support} support source fingerprints verified')
    return int(bool(failures))


if __name__ == '__main__':
    raise SystemExit(main())
