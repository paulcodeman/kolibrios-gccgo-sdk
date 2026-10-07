#!/usr/bin/env python3
"""Check preserved native CLI evidence from the verified local-model TLS run."""
from pathlib import Path
import json,shutil,sqlite3,tempfile
from native_evidence import attach_run_record
root=Path(__file__).resolve().parents[2]
files=root/'.build-cache/opencode/original-cli-real-ai-verified-tls-native-files/.opencode'
log=(files/'debug.log').read_text()
assert 'Request completed' in log and 'Non-interactive run completed' in log
assert 'level=ERROR' not in log
environment=(root/'.build-cache/opencode/real-ai-server/tls/test.env').read_text()
assert 'LOCAL_ENDPOINT=https://10.0.2.2:18443/v1' in environment
assert 'SSL_CERT_FILE=/hd0/1/test-ca.pem' in environment
with tempfile.TemporaryDirectory(prefix='opencode-tls-evidence-') as temporary:
    for name in ('opencode.db','opencode.db-wal'):
        shutil.copyfile(files/name,Path(temporary)/name)
    db=sqlite3.connect(str(Path(temporary)/'opencode.db'))
    assert db.execute('pragma integrity_check').fetchone()[0]=='ok'
    rows=db.execute('select role,parts,model from messages order by created_at').fetchall()
    assert any(role=='user' and 'Say hello in one short sentence.' in parts for role,parts,model in rows)
    assistants = [json.loads(parts) for role,parts,model in rows
                  if role=='assistant' and model=='local.kolibri-test-model' and parts!='null']
    assert any(any(part['type']=='text' and 'hello' in part['data']['text'].lower()
                   for part in parts) and
               any(part['type']=='finish' and part['data']['reason']=='end_turn'
                   for part in parts) for parts in assistants)
    report={'integrity_check':'ok','verified_TLS_endpoint':'https://10.0.2.2:18443/v1','original_CLI_completed':True,'messages':rows,'scope':'native original CLI; host Qwen inference behind a TLS endpoint trusted by explicit test CA'}
    attach_run_record(report, files.parent)
    (files.parent/'validation.json').write_text(json.dumps(report,indent=2)+'\n')
print('Original native actual AI over verified TLS PASS: real assistant response, SQLite integrity, clean command completion')
