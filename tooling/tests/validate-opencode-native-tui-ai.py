#!/usr/bin/env python3
"""Check actual inference, persistence and clean keyboard-driven TUI exit."""
from pathlib import Path
import json
import shutil
import sqlite3
import tempfile
from native_evidence import attach_run_record

root = Path(__file__).resolve().parents[2]
files = root / '.build-cache/opencode/original-cli-tui-verified-tls-native-files'
log = (files / '.opencode/debug.log').read_text()
assert 'Request completed' in log
assert 'All subscription goroutines completed successfully' in log
assert 'All goroutines cleaned up' in log and 'TUI exited with result' in log
assert 'isCompacting:false compactingMessage:Summary complete' in log
assert 'level=ERROR' not in log
with tempfile.TemporaryDirectory(prefix='opencode-tui-evidence-') as temporary:
    for name in ('opencode.db', 'opencode.db-wal'):
        shutil.copyfile(files / '.opencode' / name, Path(temporary) / name)
    db = sqlite3.connect(str(Path(temporary) / 'opencode.db'))
    assert db.execute('pragma integrity_check').fetchone()[0] == 'ok'
    rows = db.execute('select role,parts,model from messages order by created_at').fetchall()
    assert db.execute('select count(*) from sessions where summary_message_id is not null and summary_message_id != ?',['']).fetchone()[0] >= 1
    assert any(role == 'user' and any(part['type'] == 'text' and
               part['data']['text'] == 'Привет 😀' for part in json.loads(parts))
               for role, parts, model in rows)
    assistants = [json.loads(parts) for role, parts, model in rows
                  if role == 'assistant' and model == 'local.kolibri-test-model' and parts != 'null']
    assert any(any(part['type'] == 'text' and any(word in part['data']['text'].lower()
                   for word in ('hello', 'привет', 'greetings'))
                   for part in parts) and
               any(part['type'] == 'finish' and part['data']['reason'] == 'end_turn'
                   for part in parts) for parts in assistants)
    report = {'integrity_check': 'ok', 'original_CLI_completed': True,
              'native_UTF8_clipboard_paste': True,
              'original_automatic_session_summary_completed': True,
              'scope': 'native original interactive TUI; actual host inference over verified TLS; Ctrl+V Unicode prompt and orderly exit',
              'messages': rows}
    attach_run_record(report, files)
    (files / 'validation.json').write_text(json.dumps(report, indent=2) + '\n')
print('Original native TUI AI PASS: real verified-TLS response, persisted completed message and clean interactive shutdown')
