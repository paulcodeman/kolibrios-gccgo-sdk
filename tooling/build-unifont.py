#!/usr/bin/env python3
"""Pack unchanged, verified GNU Unifont bitmaps for the native console."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parents[1]

def build(output):
    source = ROOT / 'third_party/unifont'
    manifest = json.loads((source / 'UPSTREAM.json').read_text())
    for name, record in manifest['files'].items():
        if hashlib.sha256((source / name).read_bytes()).hexdigest() != record['sha256']:
            raise RuntimeError('Unifont upstream fingerprint mismatch: ' + name)
    glyphs = []
    for line in gzip.decompress((source / 'unifont_all-16.0.04.hex.gz').read_bytes()).decode('ascii').splitlines():
        code, hex_bitmap = line.split(':')
        bitmap = bytes.fromhex(hex_bitmap)
        if len(bitmap) not in (16, 32):
            raise RuntimeError('unexpected Unifont glyph size')
        glyphs.append((int(code, 16), bitmap))
    glyphs.sort()
    if len({code for code, _ in glyphs}) != len(glyphs):
        raise RuntimeError('duplicate Unifont codepoint')
    offset = 8 + 12 * len(glyphs)
    entries = bytearray()
    pixels = bytearray()
    for code, bitmap in glyphs:
        entries += struct.pack('<III', code, offset, len(bitmap) // 16)
        pixels += bitmap
        offset += len(bitmap)
    packed = b'KBF1' + struct.pack('<I', len(glyphs)) + entries + pixels
    output.parent.mkdir(parents=True, exist_ok=True)
    if not output.exists() or output.read_bytes() != packed:
        output.write_bytes(packed)
    print('Packed %d unchanged Unifont glyphs: %d bytes' % (len(glyphs), len(packed)))

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    build(parser.parse_args().output)
