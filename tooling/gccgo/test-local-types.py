#!/usr/bin/env python3
"""Check lexical type identity through native generic specialization."""

import argparse
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    parser.add_argument('--host-go-export-root', default='/usr/lib/x86_64-linux-gnu/go/15/x86_64-linux-gnu')
    args = parser.parse_args()
    sources = Path(__file__).with_name('tests') / 'local-types'
    with tempfile.TemporaryDirectory(prefix='gccgo-local-types-') as directory:
        directory = Path(directory)
        base = directory / 'regression' / 'localowner'
        base.parent.mkdir(parents=True)
        obj = Path(str(base) + '.gccgo.go.o')
        includes = ['-I' + str(directory), '-I' + args.host_go_export_root]
        subprocess.run([args.compiler, '-c', '-fno-split-stack', '-fno-pie', *includes,
                        '-fgo-pkgpath=regression/localowner',
                        str(sources / 'regression/localowner/owner.go'), '-o', str(obj)], check=True)
        subprocess.run(['objcopy', '-j', '.go_export', str(obj), str(base) + '.gox'], check=True)
        exe = directory / 'test'
        for optimization in ([], ['-O2']):
            subprocess.run([args.compiler, *optimization, '-fno-split-stack', '-fno-pie', '-no-pie',
                            *includes, str(sources / 'main.go'), str(obj), '-o', str(exe)], check=True)
            subprocess.run([str(exe)], check=True)
        bad = directory / 'scope.go'
        bad.write_text('package scope\nfunc id[T any](v T) T { return v }\n'
                       'func f() { type hidden int; _ = id(hidden(1)) }\n'
                       'func g() { _ = id(hidden(2)) }\n')
        result = subprocess.run([args.compiler, '-c', *includes, str(bad), '-o', str(directory / 'bad.o')],
                                text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if result.returncode == 0 or 'undefined: hidden' not in result.stdout:
            raise AssertionError('lifting bypassed original lexical visibility: ' + result.stdout)
        print('PASS: original local type visibility remains enforced')


if __name__ == '__main__':
    main()
