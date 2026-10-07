#!/usr/bin/env python3
"""Verify the full original CLI's real LSP diagnostic and correction."""
from pathlib import Path
import json
import shutil
import sqlite3
import tempfile
from native_evidence import attach_run_record

root = Path(__file__).resolve().parents[2]
files = root / '.build-cache/opencode/original-cli-native-lsp-ready-files'
log = (files / '.opencode/debug.log').read_text()
kernel = (root / '.build-cache/opencode/original-cli-native-lsp-ready-kernel.log').read_text()
run_record = json.loads(files.with_name(files.name[:-6] + '-run.json').read_text())
if run_record.get('expected_marker'):
    assert 'OPENCODE_ORIGINAL_LSP_RESULTS_PASS' in kernel
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
assert (files / 'fixture.sh').read_text() == 'echo Привет\nif true; then\n echo okay\nfi\n'
with tempfile.TemporaryDirectory(prefix='opencode-lsp-evidence-') as temporary:
    database = Path(temporary) / 'opencode.db'
    for name in ('opencode.db', 'opencode.db-wal'):
        shutil.copyfile(files / '.opencode' / name, Path(temporary) / name)
    db = sqlite3.connect(str(database))
    assert db.execute('pragma integrity_check').fetchone()[0] == 'ok'
    rows = db.execute('select role,parts from messages order by created_at').fetchall()
    results = [part['data'] for role, parts in rows if role == 'tool'
               for part in json.loads(parts) if part['type'] == 'tool_result']
    assert len(results) == 4 and all(not result['is_error'] for result in results)
    assert 'if true; then' in results[0]['content']
    assert 'mvdan/sh' in results[1]['content'] and 'fi' in results[1]['content']
    assert 'echo okay' in results[3]['content']
    assert '<file_diagnostics>' not in results[3]['content']
    assert any(role == 'assistant' and 'KolibriOS original LSP integration PASS' in parts
               and 'end_turn' in parts for role, parts in rows)
    report = {'integrity_check': 'ok', 'original_CLI_completed': True,
              'kernel_debug_marker_observed': 'OPENCODE_ORIGINAL_LSP_RESULTS_PASS' in kernel,
              'scope': 'native original view/write tools and LSP client; real upstream mvdan Bash parsing',
              'tool_results': results}
    attach_run_record(report, files)
    (files / 'validation.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
print('Original native full CLI LSP PASS: real diagnostic, correction, cleared error, SQLite integrity and clean completion')
