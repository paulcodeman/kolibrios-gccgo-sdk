#!/usr/bin/env python3
"""Verify collected original CLI file-tool results without altering evidence."""
from pathlib import Path
import json,shutil,sqlite3,tempfile
from native_evidence import attach_run_record
root=Path(__file__).resolve().parents[2]
files=root/'.build-cache/opencode/original-cli-native-tools-files'
assert (files/'generated.txt').read_bytes()==b'Original edit PASS\n'
log=(files/'.opencode/debug.log').read_text()
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
with tempfile.TemporaryDirectory(prefix='opencode-tools-evidence-') as temporary:
    for name in ('opencode.db','opencode.db-wal'):
        shutil.copyfile(files/'.opencode'/name,Path(temporary)/name)
    db=sqlite3.connect(str(Path(temporary)/'opencode.db'))
    assert db.execute('pragma integrity_check').fetchone()[0]=='ok'
    rows=db.execute('select role,parts,model from messages order by created_at').fetchall()
    calls=[];results=[]
    for role,parts,model in rows:
        for part in json.loads(parts):
            if part['type']=='tool_call': calls.append(part['data']['name'])
            if part['type']=='tool_result': results.append(part['data'])
    assert calls==['view','write','view','edit','view'],calls
    assert len(results)==5 and all(not result['is_error'] for result in results)
    assert any('Привет, KolibriOS!' in result['content'] for result in results)
    assert any(role=='assistant' and 'KolibriOS original tool integration PASS' in parts for role,parts,model in rows)
    report={'integrity_check':'ok','original_CLI_completed':True,'original_tool_calls':calls,'all_tool_results_successful':True,'final_file':'Original edit PASS\n','scope':'original CLI tools using a controlled native SDK peer; actual model inference is checked separately'}
    attach_run_record(report, files)
    (files/'validation.json').write_text(json.dumps(report,indent=2)+'\n')
print('Original native view/write/edit pipeline PASS: all tool results, final file, SQLite integrity')
