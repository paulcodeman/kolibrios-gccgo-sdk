"""Tie semantic native checks to the exact binaries that were booted."""
from pathlib import Path
import json


def attach_run_record(report, files):
    files = Path(files)
    assert files.name.endswith('-files'), files
    record_path = files.with_name(files.name[:-6] + '-run.json')
    record = json.loads(record_path.read_text())
    assert record['status'] in ('captured', 'native_marker_verified'), record
    report['execution'] = record

