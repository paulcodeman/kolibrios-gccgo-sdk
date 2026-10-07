#!/usr/bin/env python3
"""Derive a native OS adapter while retaining mvdan/sh's original handler.go."""
from pathlib import Path
import hashlib
import json

root = Path(__file__).resolve().parent.parent
directory = root / 'third_party/mvdan.cc/sh/v3/interp'
source = directory / 'handler.go'
original = source.read_bytes()
manifest = json.loads((directory.parent / 'UPSTREAM.json').read_text())
if hashlib.sha256(original).hexdigest() != manifest['files']['interp/handler.go']:
    raise RuntimeError('modified upstream handler.go')
text = original.decode()
text = text.replace('package interp\n', '//go:build kolibrios\n\npackage interp\n', 1)
before = 'if killTimeout <= 0 || runtime.GOOS == "windows" {'
assert text.count(before) == 1
text = text.replace(before, 'if killTimeout <= 0 || runtime.GOOS == "windows" || runtime.GOOS == "kolibrios" {')
before = '\t\tcase *exec.ExitError:\n'
assert text.count(before) == 1
text = text.replace(before, before + '''\t\t\t// Native loader status is carried by the SDK, not a Unix wait syscall.
\t\t\tif status, ok := err.Sys().(os.NativeProcessStatus); ok {
\t\t\t\tif status.Killed && ctx.Err() != nil { return ctx.Err() }
\t\t\t\tif status.Known && !status.Killed { return NewExitStatus(uint8(status.ExitCode)) }
\t\t\t\treturn NewExitStatus(1)
\t\t\t}
''')
before = 'if checkExec && runtime.GOOS != "windows" && m&0o111 == 0 {'
assert text.count(before) == 1
text = text.replace(before, '''if checkExec && runtime.GOOS == "kolibrios" {
\t\t// FAT has no executable permission bits. Use the SDK loader formats.
\t\tf, err := os.Open(file)
\t\tif err != nil { return "", err }
\t\tvar header [8]byte
\t\tn, err := io.ReadFull(f, header[:])
\t\t_ = f.Close()
\t\tif err != nil || n != len(header) || string(header[:]) != "MENUET01" && string(header[:4]) != "KPCK" {
\t\t\treturn "", fmt.Errorf("not a native executable")
\t\t}
\t}
\tif checkExec && runtime.GOOS != "windows" && runtime.GOOS != "kolibrios" && m&0o111 == 0 {''')
target = directory / 'handler_kolibrios.go'
data = text.encode()
record_path = directory / 'kolibrios.sources.json'
previous = json.loads(record_path.read_text()) if record_path.exists() else {}
if target.exists() and target.read_bytes() != data:
    if hashlib.sha256(target.read_bytes()).hexdigest() != previous.get('adapter_sha256'):
        raise RuntimeError('refusing to replace modified native shell handler')
target.write_bytes(data)
record_path.write_text(json.dumps({
    'reason': 'Original shell handler: native executable header replaces unavailable FAT permission bits, actual SDK exit status is preserved, native cancellation uses the supported Kill operation.',
    'replace': {'handler.go': 'handler_kolibrios.go'},
    'upstream_sha256': hashlib.sha256(original).hexdigest(),
    'adapter_sha256': hashlib.sha256(data).hexdigest(),
}, indent=2) + '\n')
print('Native shell OS handler derived from unchanged upstream source')
