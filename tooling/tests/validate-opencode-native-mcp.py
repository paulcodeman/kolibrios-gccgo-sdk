#!/usr/bin/env python3
"""Check the full original CLI's collected native MCP evidence."""
from pathlib import Path
import json
import shutil
import sqlite3
import tempfile
from native_evidence import attach_run_record

root = Path(__file__).resolve().parents[2]
files = root / '.build-cache/opencode/original-cli-native-mcp-env-files'
log = (files / '.opencode/debug.log').read_text()
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
with tempfile.TemporaryDirectory(prefix='opencode-mcp-evidence-') as temporary:
    for name in ('opencode.db', 'opencode.db-wal'):
        shutil.copyfile(files / '.opencode' / name, Path(temporary) / name)
    database = sqlite3.connect(str(Path(temporary) / 'opencode.db'))
    assert database.execute('pragma integrity_check').fetchone()[0] == 'ok'
    rows = database.execute('select role,parts,model from messages order by created_at').fetchall()
    calls = []
    results = []
    for role, parts, model in rows:
        for part in json.loads(parts):
            if part['type'] == 'tool_call':
                calls.append(part['data']['name'])
            if part['type'] == 'tool_result':
                results.append(part['data'])
    assert calls == ['native_echo'], calls
    assert len(results) == 1 and not results[0]['is_error'], results
    assert results[0]['content'] == 'Привет 😀 from original MCP', results
    assert any(role == 'assistant' and 'KolibriOS original MCP integration PASS' in parts for role, parts, model in rows)
    report = {
        'integrity_check': 'ok', 'original_tool_calls': calls,
        'native_child_tool_result': results[0]['content'],
        'original_CLI_completed': True,
        'scope': 'original CLI MCP discovery, permissions, native child stdio, tool execution and persistence using a controlled native provider peer',
    }
    attach_run_record(report, files)
    (files / 'validation.json').write_text(json.dumps(report, indent=2, ensure_ascii=False) + '\n')
print('Original native CLI MCP integration PASS: exact Unicode result, SQLite integrity, clean completion')
