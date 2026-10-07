#!/usr/bin/env python3
"""Import the upstream polling watcher and generate its FAT snapshot adapter."""
from pathlib import Path
import hashlib
import json
import subprocess

root = Path(__file__).resolve().parents[1]
module = json.loads(subprocess.check_output([
    '/root/sdk/go1.26.1/bin/go', 'mod', 'download', '-json',
    'github.com/radovskyb/watcher@v1.0.7'], text=True))
source = Path(module['Dir'])
destination = root / 'third_party/github.com/radovskyb/watcher'
destination.mkdir(parents=True, exist_ok=True)
records = {}
for name in ('watcher.go', 'ishidden.go', 'samefile.go', 'LICENSE'):
    data = (source / name).read_bytes()
    target = destination / name
    if target.exists() and target.read_bytes() != data:
        raise RuntimeError('modified upstream source: ' + str(target))
    target.write_bytes(data)
    records[name] = hashlib.sha256(data).hexdigest()
original = (source / 'watcher.go').read_text()
adapted = original.replace('fileList[name] = stat', 'fileList[name] = nativeSnapshot(name, stat)')
adapted = adapted.replace('fileList[path] = fInfo', 'fileList[path] = nativeSnapshot(path, fInfo)')
adapted = adapted.replace('fileList[path] = info', 'fileList[path] = nativeSnapshot(path, info)')
adapted = adapted.replace('oldInfo.ModTime() != info.ModTime()', 'nativeChanged(oldInfo, info)')
# An unbuffered completion send otherwise strands the polling goroutine on Close.
adapted = adapted.replace('done := make(chan struct{})', 'done := make(chan struct{}, 1)')
# fsnotify permits Add/Remove while polling. Do not compare or commit a snapshot
# taken before the watch set changed: that falsely reports every new watch gone.
def change(before, after):
    global adapted
    if adapted.count(before) != 1:
        raise RuntimeError('upstream watcher adaptation no longer matches: ' + before)
    adapted = adapted.replace(before, after)

change('mu           *sync.Mutex', 'generation   uint64\n\tmu           *sync.Mutex')
for method in ('Add', 'AddRecursive', 'Remove', 'RemoveRecursive'):
    before = 'func (w *Watcher) ' + method + '(name string) (err error) {\n\tw.mu.Lock()\n\tdefer w.mu.Unlock()'
    change(before, before + '\n\tw.generation++')
change('func (w *Watcher) retrieveFileList() map[string]os.FileInfo {',
       'func (w *Watcher) retrieveFileList() (map[string]os.FileInfo, uint64) {')
change('\treturn fileList\n}\n\n// Start begins', '\treturn fileList, w.generation\n}\n\n// Start begins')
change('fileList := w.retrieveFileList()', 'fileList, generation := w.retrieveFileList()')
change('w.pollEvents(fileList, evt, cancel)', 'w.pollEvents(fileList, evt, cancel, generation)')
change('\t\tw.files = fileList', '\t\tif w.generation == generation {\n\t\t\tw.files = fileList\n\t\t}')
change('\tcancel chan struct{}) {\n\tw.mu.Lock()\n\tdefer w.mu.Unlock()',
       '\tcancel chan struct{}, generation uint64) {\n\tw.mu.Lock()\n\tdefer w.mu.Unlock()\n\tif w.generation != generation {\n\t\treturn\n\t}')
(destination / 'watcher_kolibrios.go').write_text('//go:build kolibrios\n\n' + adapted)
(destination / 'kolibrios.sources.json').write_text(json.dumps({
    'reason': 'Retain upstream polling watcher; FAT content snapshots detect coarse-timestamp edits and generations protect concurrent watch changes.',
    'replace': {'watcher.go': 'watcher_kolibrios.go'}}, indent=2) + '\n')
(destination / 'UPSTREAM.json').write_text(json.dumps({
    'repository': 'https://github.com/radovskyb/watcher', 'version': module['Version'],
    'module_sum': module['Sum'], 'go_mod_sum': module['GoModSum'], 'files': records,
    'adaptations': {'watcher_kolibrios.go': {
        'sha256': hashlib.sha256((destination / 'watcher_kolibrios.go').read_bytes()).hexdigest(),
        'source': 'watcher.go',
        'reason': 'FAT has coarse modification timestamps and no watch syscalls: retain upstream poller, record content fingerprints, guard concurrent watch changes and avoid a stranded completion send on Close.'}},
}, indent=2) + '\n')
print('Imported upstream watcher v1.0.7 and generated FAT snapshot adaptation')
