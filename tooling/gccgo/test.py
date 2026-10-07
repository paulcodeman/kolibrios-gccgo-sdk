#!/usr/bin/env python3
"""Check locally patched Go builtins and integer ranges on the host and 386."""

import argparse
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    parser.add_argument('--reference-go', help='optional upstream Go compiler for the min/max execution fixture')
    parser.add_argument('--host-go-export-root', default='/usr/lib/x86_64-linux-gnu/go/15/x86_64-linux-gnu',
                        help='host libgo export files for runtime.Error in the unsafe fixture')
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='gccgo-regression-') as directory:
        directory = Path(directory)
        for name in ('clear', 'rangeint', 'minmax', 'generics', 'unsafe'):
            source = Path(__file__).with_name('tests') / (name + '.go')
            sources = [str(source)]
            flags = []
            if name == 'unsafe':
                sources.append(str(source.with_name('unsafe-runtime.go')))
                flags.append('-I' + args.host_go_export_root)
            exe = directory / name
            subprocess.run([args.compiler, '-fno-split-stack', '-fno-pie', '-no-pie'] + flags + sources + ['-o', str(exe)], check=True)
            subprocess.run([str(exe)], check=True)
            subprocess.run([args.compiler, '-m32', '-c', '-fno-split-stack'] + flags + sources + ['-o', str(directory / (name + '-386.o'))], check=True)
            if name in ('minmax', 'generics', 'unsafe'):
                subprocess.run([args.compiler, '-O2', '-fno-split-stack', '-fno-pie', '-no-pie'] + flags + sources + ['-o', str(exe)], check=True)
                subprocess.run([str(exe)], check=True)
                if args.reference_go:
                    subprocess.run([args.reference_go, 'run', str(source)], check=True)
        for expression in ['clear()', 'clear(1)', 'clear("x")', 'clear([1]int{})',
                           'clear(make(chan int))', 'clear(nil)', 'clear([]int{}, []int{})',
                           'clear([]int{}...)', '_ = clear([]int{})',
                           'for i, v := range 3 { _, _ = i, v }',
                           'for range 1.5 {}', 'var i int8; for i = range 128 {}',
                           'var i uint; for i = range int(3) {}',
                           '_ = min()', '_ = max()', '_ = min(true, false)',
                           '_ = max(1+2i, 2+1i)', '_ = min(nil)',
                           '_ = min([]int{1, 2}...)', '_ = max([]int{1}, []int{2})',
                           '_ = min(int32(1), int64(2))', '_ = max("a", 1)',
                           '_ = min(int8(1), 1000)', '_ = max(uint8(1), -1)',
                           '_ = min(int(1), 1.5)', 'min(1, 2)', 'max(1, 2)',
                           '_ = min', '_ = max', '_ = min(make(chan int))',
                           'var x int8 = max(1, 1000); _ = x',
                           '_ = min("a", int(1))', '_ = max(true, int(1))',
                           '_ = min(1, string("a"))',
                           '_ = max(int(1), 2+3i)', '_ = min(int(1), 2+0i)',
                           'type box[T ~int] struct { value T }; _ = box[string]{}',
                           'type box[T comparable] struct { value T }; _ = box[[]int]{}']:
            bad = directory / 'invalid.go'
            bad.write_text('package bad\nfunc F() { ' + expression + ' }\n')
            result = subprocess.run([args.compiler, '-c', str(bad), '-o', str(directory / 'bad.o')], capture_output=True, text=True)
            if result.returncode == 0 or 'internal compiler error' in result.stderr:
                raise RuntimeError('expected a clean diagnostic for ' + expression + ': ' + result.stderr)
        for expression in ['unsafe.StringData()', 'unsafe.StringData(1)', 'unsafe.StringData(nil)',
                           'unsafe.StringData("a", "b")', 'unsafe.SliceData(1)', 'unsafe.SliceData(nil)',
                           'unsafe.SliceData([1]int{})', 'unsafe.String(new(int), 1)',
                           'unsafe.String(new(byte), -1)', 'unsafe.String(nil, 1.5)',
                           'unsafe.String(nil, 1<<64)', 'unsafe.String(nil)',
                           'unsafe.String(nil, 0, 0)', 'unsafe.String', 'unsafe.SliceData', 'unsafe.StringData(text("x"))']:
            bad = directory / 'invalid-unsafe.go'
            bad.write_text('package bad\nimport "unsafe"\ntype text string\nfunc F() { _ = ' + expression + ' }\n')
            result = subprocess.run([args.compiler, '-c', str(bad), '-o', str(directory / 'bad.o')], capture_output=True, text=True)
            if result.returncode == 0 or 'internal compiler error' in result.stderr:
                raise RuntimeError('expected a clean diagnostic for ' + expression + ': ' + result.stderr)
        print('PASS: clear, integer range, min/max, local generics and unsafe; 386 objects; 54 negative cases')


if __name__ == '__main__':
    main()
