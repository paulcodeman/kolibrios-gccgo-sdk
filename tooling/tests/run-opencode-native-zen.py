#!/usr/bin/env python3
"""Verify public Zen discovery, the original model picker and chat in KolibriOS.

Run on Windows with the prepared Desktop/opencode-qemu runtime and WSL mtools.
Uses private disk copies; never touches the user's running VM or local model.
"""
from pathlib import Path
import hashlib
import json
import shutil
import socket
import sqlite3
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
CACHE = ROOT / '.build-cache/opencode/zen-public/native'
RUNTIME = Path.home() / 'Desktop/opencode-qemu'


def linux(path):
    return '/mnt/' + path.drive[0].lower() + path.as_posix()[2:]


def mcopy(image, source, destination):
    subprocess.run(['wsl.exe', '--exec', 'mcopy', '-o', '-i', image, source, destination], check=True)


def main():
    CACHE.mkdir(parents=True, exist_ok=True)
    disk, boot = CACHE / 'runtime.img', CACHE / 'boot.img'
    shutil.copyfile(RUNTIME / 'opencode-runtime.img', disk)
    shutil.copyfile(RUNTIME / 'opencode-boot.img', boot)
    disk_spec = linux(disk) + '@@1048576'
    binary = ROOT / 'apps/opencode/opencode.kex'
    mcopy(disk_spec, linux(binary), '::/opencode/opencode.kex')
    resources = ROOT / 'apps/opencode/assets'
    subprocess.run(['wsl.exe', '--exec', 'mmd', '-i', disk_spec, '::/opencode/assets'],
                   capture_output=True)
    for name in ('gosh.kex', 'ca-bundle.crt'):
        mcopy(disk_spec, linux(resources / name), '::/opencode/assets/' + name)
    environment = CACHE / 'public.env'
    environment.write_text((ROOT / 'apps/opencode/opencode.kex.env').read_text(encoding='utf-8') +
                           'OPENCODE_DEV_DEBUG=true\n', encoding='utf-8')
    mcopy(disk_spec, linux(environment), '::/opencode/opencode.kex.env')
    autorun_file = CACHE / 'autorun.dat'
    autorun_file.unlink(missing_ok=True)
    mcopy(linux(boot), '::/SETTINGS/AUTORUN.DAT', linux(autorun_file))
    autorun = autorun_file.read_text().replace('opencode.kex ""', 'opencode.kex "-d"')
    autorun = autorun.replace('/hd0/1/opencode/opencode.kex',
                              '/SYS/DEVELOP/BOARD "" 0\n/hd0/1/opencode/opencode.kex')
    autorun_file.write_bytes(autorun.replace('\r\n', '\n').replace('\n', '\r\n').encode('ascii'))
    mcopy(linux(boot), linux(autorun_file), '::/SETTINGS/AUTORUN.DAT')
    with socket.socket() as probe:
        probe.bind(('127.0.0.1', 0))
        port = probe.getsockname()[1]
    qemu = RUNTIME / 'qemu/qemu-system-i386.exe'
    arguments = [str(qemu), '-machine', 'pc', '-accel', 'tcg', '-cpu', 'max', '-m', '512',
                 '-vga', 'std', '-display', 'none', '-boot', 'order=a',
                 '-drive', f'file={boot},format=raw,if=floppy,index=0',
                 '-drive', f'file={disk},format=raw,if=ide,index=0',
                 '-netdev', 'user,id=net0', '-device', 'rtl8139,netdev=net0',
                 '-qmp', f'tcp:127.0.0.1:{port},server=on,wait=off',
                 '-debugcon', 'file:' + str(CACHE / 'kernel.log')]
    with (CACHE / 'qemu.log').open('w') as log:
        process = subprocess.Popen(arguments, stdout=log, stderr=subprocess.STDOUT,
                                   creationflags=subprocess.CREATE_NO_WINDOW)
        try:
            for _ in range(150):
                try:
                    connection = socket.create_connection(('127.0.0.1', port), timeout=2)
                    break
                except OSError:
                    if process.poll() is not None:
                        raise RuntimeError((CACHE / 'qemu.log').read_text())
                    time.sleep(0.1)
            else:
                raise RuntimeError('QMP timeout')
            with connection:
                connection.settimeout(15)
                stream = connection.makefile('rwb')
                greeting = json.loads(stream.readline())

                def execute(command, arguments=None):
                    request = {'execute': command}
                    if arguments is not None:
                        request['arguments'] = arguments
                    stream.write((json.dumps(request) + '\n').encode())
                    stream.flush()
                    while True:
                        response = json.loads(stream.readline())
                        if 'error' in response:
                            raise RuntimeError(response['error'])
                        if 'return' in response:
                            return response['return']

                execute('qmp_capabilities')
                events = {90: ['n'], 100: ['ctrl', 'o'], 106: ['up'], 107: ['down'],
                          108: ['ret'], 120: ['h'], 121: ['i'], 125: ['ret'],
                          200: ['ctrl', 'o'], 205: ['up'], 206: ['ret'],
                          240: ['ctrl', 'c'], 242: ['y']}
                captures = [30, 60, 95, 105, 115, 150, 180, 208, 210, 250]
                start = time.monotonic()
                for instant in sorted(set(captures) | set(events)):
                    time.sleep(max(0, start + instant - time.monotonic()))
                    if instant in events:
                        execute('send-key', {'keys': [{'type': 'qcode', 'data': key}
                                                     for key in events[instant]], 'hold-time': 80})
                    if instant in captures:
                        filename = CACHE / f'{instant}s.ppm'
                        execute('screendump', {'filename': str(filename)})
                        print('Captured native Zen at ' + str(instant) + 's', flush=True)
                execute('quit')
                process.wait(timeout=15)
        finally:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=15)
    files = CACHE / 'files'
    files.mkdir(exist_ok=True)
    for name, guest in [('debug.log', '.opencode/debug.log'),
                        ('opencode.db', '.opencode/opencode.db'),
                        ('opencode.db-wal', '.opencode/opencode.db-wal'),
                        ('models.json', '.cache/opencode/zen-public-models.json'),
                        ('config.json', '.opencode.json')]:
        destination = files / name
        destination.unlink(missing_ok=True)
        result = subprocess.run(['wsl.exe', '--exec', 'mcopy', '-o', '-i', disk_spec,
                                 '::/opencode/' + guest, linux(destination)], capture_output=True, text=True)
        if result.returncode and name not in ('opencode.db-wal', 'config.json'):
            raise RuntimeError(result.stderr)
    kernel = (CACHE / 'kernel.log').read_text(errors='replace')
    for failure in ('runtime panic:', 'K : Page fault', 'K : General protection'):
        assert failure not in kernel, failure
    log = (files / 'debug.log').read_text(encoding='utf-8')
    assert 'level=ERROR' not in log, log
    for marker in ('Request completed', 'All goroutines cleaned up', 'TUI exited with result'):
        assert marker in log, marker
    discovered = json.loads((files / 'models.json').read_text(encoding='utf-8'))
    assert discovered and all(m['context_window'] > 4096 and
                             m['cost_per_1m_in'] == m['cost_per_1m_out'] == 0 for m in discovered)
    reference = json.loads((CACHE.parent / 'catalog.json').read_text(encoding='utf-8'))['opencode']['models']
    for model in discovered:
        entry = reference[model['api_model']]
        assert entry['cost']['input'] == entry['cost']['output'] == 0
        assert entry.get('status') != 'deprecated' and entry['tool_call']
        assert model['context_window'] == entry['limit']['context']
    saved = json.loads((files / 'config.json').read_text(encoding='utf-8'))
    assert saved['agents']['coder']['model'] == 'local.zen-big-pickle', saved
    with sqlite3.connect(str(files / 'opencode.db')) as db:
        assert db.execute('pragma integrity_check').fetchone()[0] == 'ok'
        rows = db.execute('select role,parts,model from messages order by created_at').fetchall()
    replies = [json.loads(parts) for role, parts, model in rows
               if role == 'assistant' and model == 'local.zen-space-bunny-free' and parts != 'null']
    assert any(any(p['type'] == 'text' and p['data']['text'].strip() for p in reply) and
               any(p['type'] == 'finish' and p['data']['reason'] == 'end_turn' for p in reply)
               for reply in replies), rows
    assert all(not any(p['type'] == 'finish' and p['data']['reason'] == 'summary' for p in reply)
               for reply in replies)
    report = {'QEMU': greeting, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(),
              'public_API_key': 'public', 'remote_endpoint': 'https://opencode.ai/zen/v1',
              'native_model_picker_keys': 'Ctrl+O, Up, Down, Enter',
              'model_selection_persisted': True,
              'selected_after_reply': 'local.zen-big-pickle',
              'native_Enter_remote_reply': True, 'sqlite_integrity': 'ok',
              'clean_exit': True, 'models': discovered, 'messages': rows,
              'binary_path': 'apps/opencode/opencode.kex',
              'assets_relative_paths': True,
              'assets_sha256': {name: hashlib.sha256((resources / name).read_bytes()).hexdigest()
                                for name in ('gosh.kex', 'ca-bundle.crt')}}
    (CACHE / 'validation.json').write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    print('PASS: native discovery, original model picker, public remote reply, SQLite and clean exit')


if __name__ == '__main__':
    main()
