#!/usr/bin/env python3
"""Exercise generic type identity across separately compiled package boundaries."""

import argparse
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    parser.add_argument('--host-go-export-root', default='/usr/lib/x86_64-linux-gnu/go/15/x86_64-linux-gnu')
    args = parser.parse_args()
    sources = Path(__file__).with_name('tests') / 'cross-package'
    with tempfile.TemporaryDirectory(prefix='gccgo-cross-package-') as directory:
        directory = Path(directory)
        objects = []
        for name in ('legacy', 'owner', 'consumer'):
            base = directory / 'regression' / name
            base.parent.mkdir(parents=True, exist_ok=True)
            obj = Path(str(base) + '.gccgo.go.o')
            subprocess.run([args.compiler, '-c', '-fno-split-stack', '-fno-pie',
                            '-I' + str(directory), '-I' + args.host_go_export_root, '-fgo-pkgpath=regression/' + name,
                            *map(str, sorted((sources / 'regression' / name).glob('*.go'))), '-o', str(obj)], check=True)
            subprocess.run(['objcopy', '-j', '.go_export', str(obj), str(base) + '.gox'], check=True)
            objects.append(str(obj))
        exe = directory / 'regression-test'
        subprocess.run([args.compiler, '-fno-split-stack', '-fno-pie', '-no-pie',
                        '-I' + str(directory), '-I' + args.host_go_export_root,
                        str(sources / 'main.go')] + objects + ['-o', str(exe)], check=True)
        subprocess.run([str(exe)], check=True)
        invalid = directory / 'private-access.go'
        for source in (
            'package main\nimport "regression/owner"\nvar _ = owner.Nested[int]{inner: owner.Make(1)}\n',
            'package main\nimport "regression/owner"\nvar _ = owner.NewNested(1).inner\n',
        ):
            invalid.write_text(source)
            result = subprocess.run([args.compiler, '-c', '-fno-split-stack',
                                     '-I' + str(directory), '-I' + args.host_go_export_root,
                                     str(invalid), '-o', str(directory / 'invalid.o')],
                                    capture_output=True, text=True)
            if result.returncode == 0 or 'unexported' not in result.stderr:
                raise RuntimeError('private field access was not rejected: ' + result.stderr)
        print('PASS: original checking rejects private generic fields in caller source')
        subprocess.run([args.compiler, '-O2', '-fno-split-stack', '-fno-pie', '-no-pie',
                        '-I' + str(directory), '-I' + args.host_go_export_root,
                        str(sources / 'main.go')] + objects + ['-o', str(exe)], check=True)
        subprocess.run([str(exe)], check=True)


if __name__ == '__main__':
    main()
