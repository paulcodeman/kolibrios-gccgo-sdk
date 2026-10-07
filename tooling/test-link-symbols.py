#!/usr/bin/env python3
"""Verify linked traceback metadata, including repeated and escaped source paths."""

import ctypes
import importlib.util
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location('link_symbols', ROOT / 'link-symbols.py')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class Symbol(ctypes.Structure):
    _fields_ = [('start', ctypes.c_size_t), ('end', ctypes.c_size_t),
                ('name', ctypes.c_char_p)]


class SourceLine(ctypes.Structure):
    _fields_ = [('pc', ctypes.c_size_t), ('file', ctypes.c_char_p),
                ('line', ctypes.c_ssize_t)]


def verify(folder, name, records, lines):
    source, library = folder / (name + '.c'), folder / (name + '.so')
    MODULE.metadata(source, records, lines)
    subprocess.run(['gcc', '-shared', '-fPIC', '-Os', str(source), '-o', str(library)], check=True)
    handle = ctypes.CDLL(str(library))
    assert ctypes.c_uint32.in_dll(handle, 'runtime_kolibri_symbol_count').value == len(records)
    assert ctypes.c_uint32.in_dll(handle, 'runtime_kolibri_line_count').value == len(lines)
    actual = (Symbol * max(1, len(records))).in_dll(handle, 'runtime_kolibri_symbols')
    assert [(s.start, s.end, s.name.decode('utf-8')) for s in actual[:len(records)]] == records
    actual_lines = (SourceLine * max(1, len(lines))).in_dll(handle, 'runtime_kolibri_lines')
    assert [(s.pc, s.file.decode('utf-8'), s.line) for s in actual_lines[:len(lines)]] == lines
    if not records:
        assert actual[0].start == actual[0].end == 0 and actual[0].name is None
    if not lines:
        assert actual_lines[0].pc == actual_lines[0].line == 0 and actual_lines[0].file is None
    return source.stat().st_size


def main():
    with tempfile.TemporaryDirectory(prefix='kolibri-metadata-test-') as temporary:
        folder = Path(temporary)
        verify(folder, 'empty', [], [])
        path = '/project/' + 'long-source-directory/' * 10 + 'кавычки"\\file.go'
        records = [(0x20, 0x50, 'pkg.Функция"\\'), (0x60, 0x80, 'other.Func')]
        lines = [(0x20 + 4 * i, path, i + 1) for i in range(10000)]
        lines += [(0x100000, '', 0), (0x100004, '/other/source.go', 7)]
        size = verify(folder, 'repeated', records, lines)
        # Source metadata must scale with positions, rather than path length
        # times positions. Verify a compiled C table, not only emitted text.
        assert size < 1000000, size
        print('PASS: compiled symbol/source tables preserve all records; repeated-path C source %d bytes' % size)


if __name__ == '__main__':
    main()
