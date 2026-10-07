#!/usr/bin/env python3
"""Verify original AI completion from the actual release installation paths."""
from pathlib import Path
import json
import shutil
import sqlite3
import tempfile
from native_evidence import attach_run_record

root = Path(__file__).resolve().parents[2]
files = root / '.build-cache/opencode/original-cli-installed-package-tls-files'
data = files / 'project/.opencode'
log = (data / 'debug.log').read_text()
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
with tempfile.TemporaryDirectory(prefix='opencode-installed-evidence-') as temporary:
    for name in ('opencode.db', 'opencode.db-wal'):
        shutil.copyfile(data / name, Path(temporary) / name)
    db = sqlite3.connect(str(Path(temporary) / 'opencode.db'))
    assert db.execute('pragma integrity_check').fetchone()[0] == 'ok'
    rows = db.execute('select role,parts,model from messages order by created_at').fetchall()
    assert any(role == 'user' and 'Say hello in one short sentence.' in parts for role, parts, model in rows)
    assistants = [json.loads(parts) for role, parts, model in rows
                  if role == 'assistant' and model == 'local.kolibri-test-model' and parts != 'null']
    assert any(any(part['type'] == 'text' and 'hello' in part['data']['text'].lower() for part in parts) and
               any(part['type'] == 'finish' and part['data']['reason'] == 'end_turn' for part in parts)
               for parts in assistants)
    report = {'original_CLI_completed': True, 'integrity_check': 'ok',
              'installation_path': '/hd0/1/opencode/opencode.kex',
              'project_directory': '/hd0/1/project',
              'scope': 'release binary, kernel, console, HOME and public CA bundle; actual verified-TLS inference with a separate test-only CA directory',
              'messages': rows}
    attach_run_record(report, files)
    (files / 'validation.json').write_text(json.dumps(report, indent=2) + '\n')
print('Installed original CLI PASS: release paths, real verified-TLS AI, SQLite integrity and clean completion')
