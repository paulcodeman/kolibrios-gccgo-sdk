#!/usr/bin/env python3
"""Validate original CLI shell state, failure status and native search results."""
from pathlib import Path
import json
import shutil
import sqlite3
import tempfile
from native_evidence import attach_run_record

root = Path(__file__).resolve().parents[2]
files = root / '.build-cache/opencode/original-cli-native-shell-search-case-files'
log = (files / '.opencode/debug.log').read_text()
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
with tempfile.TemporaryDirectory(prefix='opencode-shell-evidence-') as temporary:
    for name in ('opencode.db', 'opencode.db-wal'):
        shutil.copyfile(files / '.opencode' / name, Path(temporary) / name)
    database = sqlite3.connect(str(Path(temporary) / 'opencode.db'))
    assert database.execute('pragma integrity_check').fetchone()[0] == 'ok'
    rows = database.execute('select role,parts,model from messages order by created_at').fetchall()
    calls, results = [], []
    for role, parts, model in rows:
        for part in json.loads(parts):
            if part['type'] == 'tool_call':
                calls.append(part['data']['name'])
            if part['type'] == 'tool_result':
                results.append(part['data'])
    assert calls == ['bash', 'bash', 'grep', 'glob', 'ls'], calls
    assert len(results) == 5 and all(not result['is_error'] for result in results), results
    assert 'first=Привет' in results[0]['content']
    assert all(text in results[1]['content'] for text in ('second=Привет', 'shell stderr', 'Exit code 7'))
    assert 'Привет, KolibriOS!' in results[2]['content']
    assert all('fixture.txt' in result['content'] for result in results[3:])
    assert any(role == 'assistant' and 'KolibriOS original shell and search integration PASS' in parts for role, parts, model in rows)
    report = {'integrity_check': 'ok', 'original_tool_calls': calls,
              'persistent_shell_state': True, 'stderr_and_exit_7': True,
              'native_grep_glob_ls': True, 'original_CLI_completed': True,
              'scope': 'controlled provider peer, original CLI tools, upstream mvdan shell and native FAT; model inference checked separately'}
    attach_run_record(report, files)
    (files / 'validation.json').write_text(json.dumps(report, indent=2) + '\n')
print('Original native shell and search PASS: persistent state, stderr, exit 7, grep, glob, ls, SQLite integrity, clean completion')
