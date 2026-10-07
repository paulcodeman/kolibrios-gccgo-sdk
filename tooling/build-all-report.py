#!/usr/bin/env python3
"""Record verified outputs from the per-folder batch build logs."""
import csv
import hashlib
import json
from pathlib import Path
import sys

root = Path(__file__).resolve().parent.parent
report = Path(sys.argv[1])
records = []
with report.open() as source:
    for row in csv.DictReader(source, delimiter='\t'):
        if row['result'] == 'PASS':
            # build-app.sh prints the actual Makefile output as its last line.
            output = Path(Path(row['log']).read_text().splitlines()[-1])
            row['output'] = str(output.relative_to(root))
            row['bytes'] = output.stat().st_size
            row['sha256'] = hashlib.sha256(output.read_bytes()).hexdigest()
        records.append(row)
data = {'targets': len(records), 'passed': sum(r['result'] == 'PASS' for r in records),
        'failed': sum(r['result'] == 'FAIL' for r in records), 'results': records}
report.with_suffix('.json').write_text(json.dumps(data, indent=2) + '\n')
latest = report.parent.parent / 'latest.json'
latest.write_text(json.dumps({'report': str(report.with_suffix('.json')), **data}, indent=2) + '\n')
