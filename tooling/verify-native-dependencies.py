#!/usr/bin/env python3
"""Verify upstream shell, MCP and watcher sources and declared adaptations."""
from pathlib import Path
import hashlib
import json

root = Path(__file__).resolve().parents[1]
files = adaptations = 0
for relative in ('mvdan.cc/sh/v3', 'github.com/mark3labs/mcp-go', 'github.com/radovskyb/watcher',
                 'github.com/charmbracelet/x/term', 'github.com/muesli/cancelreader',
                 'golang.org/x/term', 'golang.org/x/sync/errgroup'):
    directory = root / 'third_party' / relative
    manifest = json.loads((directory / 'UPSTREAM.json').read_text())
    for name, expected in manifest['files'].items():
        assert hashlib.sha256((directory / name).read_bytes()).hexdigest() == expected, (relative, name)
        files += 1
    for name, record in manifest.get('adaptations', {}).items():
        assert record.get('reason'), (relative, name)
        assert hashlib.sha256((directory / name).read_bytes()).hexdigest() == record['sha256'], (relative, name)
        adaptations += 1
    for selection in directory.rglob('kolibrios.sources.json'):
        record = json.loads(selection.read_text())
        assert record['reason'], str(selection)
        if 'adapter_sha256' in record:
            assert len(record['replace']) == 1
            original, adapted = next(iter(record['replace'].items()))
            assert hashlib.sha256((selection.parent / original).read_bytes()).hexdigest() == record['upstream_sha256']
            assert hashlib.sha256((selection.parent / adapted).read_bytes()).hexdigest() == record['adapter_sha256']
            adaptations += 1
print(f'{files} upstream native dependency files verified; {adaptations} declared platform adaptations verified')

shared = root / 'third_party/kolibrios-compat'
manifest = json.loads((shared / 'ADAPTERS.json').read_text())
for name, record in manifest['adapters'].items():
    assert record['module'] in manifest['tested_modules'], name
    assert record['kind'] in ('upstream', 'platform', 'selection'), name
    assert hashlib.sha256((root / 'third_party' / record['origin']).read_bytes()).hexdigest() == record['sha256'], name
for name, expected in manifest['licenses'].items():
    assert hashlib.sha256((shared / name).read_bytes()).hexdigest() == expected, name
originals = sum(record['kind'] == 'upstream' for record in manifest['adapters'].values())
print(f"{len(manifest['adapters'])} shared compatibility files verified ({originals} unchanged upstream sources); {len(manifest['licenses'])} upstream licenses verified")
