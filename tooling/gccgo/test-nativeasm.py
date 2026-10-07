#!/usr/bin/env python3
"""Verify retained GCC native symbols in original Go/generic inputs."""

import argparse
import hashlib
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    parser.add_argument('--cc', default='gcc')
    args = parser.parse_args()
    fixture = Path(__file__).with_name('tests')
    source = fixture / 'nativeasm.go'
    original = hashlib.sha256(source.read_bytes()).digest()
    with tempfile.TemporaryDirectory(prefix='gccgo-nativeasm-') as directory:
        directory = Path(directory)
        native = directory / 'native.o'
        subprocess.run([args.cc, '-c', str(fixture / 'nativeasm.c'), '-o', str(native)], check=True)
        for optimize in ([], ['-O2']):
            executable = directory / 'nativeasm'
            subprocess.run([args.compiler, '-fno-split-stack', '-fno-pie', '-no-pie'] + optimize +
                           [str(source), str(native), '-o', str(executable)], check=True)
            subprocess.run([str(executable)], check=True)
        subprocess.run([args.compiler, '-m32', '-c', '-fno-split-stack', str(source),
                        '-o', str(directory / 'nativeasm-386.o')], check=True)
    if hashlib.sha256(source.read_bytes()).digest() != original:
        raise RuntimeError('compiler rewrote original native source')
    print('PASS: native GCC symbol annotations; generic calls; host/default/O2; 386 object; unchanged source')


if __name__ == '__main__':
    main()
