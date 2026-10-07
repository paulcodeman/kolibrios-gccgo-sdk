#!/usr/bin/env python3
"""Retain GCC symbol/source information when linking a flat KolibriOS app."""

import argparse
from pathlib import Path
import re
import shlex
import subprocess
import tempfile


def symbols(elf):
    result = {}
    output = subprocess.check_output(['nm', '-n', '-S', '--defined-only', str(elf)], text=True)
    for line in output.splitlines():
        fields = line.split()
        if len(fields) != 4 or fields[2] not in ('t', 'T'):
            continue
        start, size = int(fields[0], 16), int(fields[1], 16)
        if size:
            result[start] = (start, start + size, fields[3])
    return sorted(result.values())


def source_lines(elf):
    result = {}
    directory = ''
    output = subprocess.check_output(['readelf', '--debug-dump=decodedline', '--wide', str(elf)], text=True)
    for line in output.splitlines():
        if line.startswith('CU: '):
            directory = str(Path(line[4:].rstrip(':')).parent)
        elif line.endswith(':') and Path(line[:-1]).is_absolute():
            # readelf changes file tables within a multi-file compilation
            # unit. The CU's first directory is not always the current file's.
            directory = str(Path(line[:-1]).parent)
        match = re.match(r'^(.+?)\s+(\d+|-)\s+(0x[0-9a-fA-F]+)(?:\s|$)', line.strip())
        if match:
            file, number, address = match.groups()
            if number == '-':
                # End-of-sequence stops an earlier source location from
                # leaking into a gap or an assembly-only function.
                result[int(address, 16)] = ('', 0)
            else:
                file = str(Path(directory) / file) if not Path(file).is_absolute() else file
                result[int(address, 16)] = (file, int(number))
    return [(pc, file, line) for pc, (file, line) in sorted(result.items())]


def c_string(value):
    # Octal byte escapes are independent of source encoding and cannot merge
    # with following hexadecimal characters as C's hex escapes can.
    return '"' + ''.join('\\%03o' % byte for byte in value.encode('utf-8')) + '"'


def metadata(path, records, lines):
    # A large Go application has hundreds of thousands of source positions,
    # but only a small number of distinct filenames. Emit each string once
    # instead of making the C frontend parse its octal escapes for every row.
    # Keep symbol and source records unchanged, including empty filenames.
    strings = dict.fromkeys([name for _, _, name in records] +
                            [file for _, file, _ in lines])
    strings = {value: 'runtime_kolibri_text_%d' % index
               for index, value in enumerate(strings)}
    with path.open('w') as output:
        output.write('#include <stdint.h>\n#include <stddef.h>\n')
        output.write('typedef struct { uintptr_t start,end; const char* name; } symbol;\n')
        output.write('typedef struct { uintptr_t pc; const char* file; intptr_t line; } source_line;\n')
        for value, identifier in strings.items():
            output.write('static const char %s[] = %s;\n' % (identifier, c_string(value)))
        output.write('const symbol runtime_kolibri_symbols[] = {\n')
        for start, end, name in records:
            output.write('{0x%x,0x%x,%s},\n' % (start, end, strings[name]))
        if not records:
            output.write('{0,0,NULL},\n')
        output.write('};\nconst uint32_t runtime_kolibri_symbol_count = %d;\n' % len(records))
        output.write('const source_line runtime_kolibri_lines[] = {\n')
        for pc, file, line in lines:
            output.write('{0x%x,%s,%d},\n' % (pc, strings[file], line))
        if not lines:
            output.write('{0,NULL,0},\n')
        output.write('};\nconst uint32_t runtime_kolibri_line_count = %d;\n' % len(lines))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cc', required=True)
    parser.add_argument('link', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.link[1:] if args.link and args.link[0] == '--' else args.link
    if '-o' not in command:
        parser.error('a linker command with -o is required')
    elf = Path(command[command.index('-o') + 1])
    subprocess.run(command, check=True)
    if not any(name == 'runtime_kolibri_lookup_frame' for _, _, name in symbols(elf)):
        return 0
    with tempfile.TemporaryDirectory(prefix='kolibri-symbols-') as temporary:
        source = Path(temporary) / 'symbols.c'
        obj = Path(temporary) / 'symbols.o'
        for _ in range(3):
            records, lines = symbols(elf), source_lines(elf)
            metadata(source, records, lines)
            subprocess.run(shlex.split(args.cc) + ['-m32', '-c', '-Os', '-fno-pic', '-fno-pie',
                           '-fno-stack-protector', str(source), '-o', str(obj)], check=True)
            subprocess.run(command + [str(obj)], check=True)
            if symbols(elf) == records and source_lines(elf) == lines:
                print('Retained %d function symbols and %d source positions' % (len(records), len(lines)))
                return 0
        raise RuntimeError('symbol addresses did not stabilize after metadata linking')


if __name__ == '__main__':
    raise SystemExit(main())
