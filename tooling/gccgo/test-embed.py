#!/usr/bin/env python3
"""Compile original embedded resources with generics on the host and 386."""

import argparse
import hashlib
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    parser.add_argument('--host-go-export-root', default='/usr/lib/x86_64-linux-gnu/go/15/x86_64-linux-gnu')
    args = parser.parse_args()
    compiler = str(Path(args.compiler).resolve())
    source = Path(__file__).with_name('tests') / 'embed/main.go'
    before = {str(path): hashlib.sha256(path.read_bytes()).hexdigest()
              for path in source.parent.rglob('*') if path.is_file()}
    with tempfile.TemporaryDirectory(prefix='gccgo-embed-') as directory:
        directory = Path(directory)
        executable = directory / 'embed'
        flags = ['-I' + args.host_go_export_root, '-fno-split-stack', '-fno-pie', '-no-pie']
        for optimize in ([], ['-O2']):
            subprocess.run([compiler] + flags + optimize + [str(source), '-o', str(executable)], check=True)
            subprocess.run([str(executable)], check=True)
        output = directory / 'embed-386.o'
        subprocess.run([compiler, '-m32', '-c'] + flags + [str(source), '-o', str(output)], check=True)
        dependencies = Path(str(output) + '.embed.d').read_text()
        for name in ('binary.bin', 'space\\ file.txt', '.hidden', 'sub/a.txt'):
            if name not in dependencies:
                raise RuntimeError('missing original-resource build dependency: ' + name)
        copied = directory / 'original'
        shutil.copytree(source.parent, copied)
        empty = copied / 'resources/empty'
        empty.mkdir()
        rebuilt = directory / 'rebuilt.o'
        command = [compiler, '-m32', '-c'] + flags + [str(copied / 'main.go'), '-o', str(rebuilt)]
        makefile = directory / 'Makefile'
        makefile.write_text(f'.DEFAULT_GOAL := {rebuilt}\n-include {rebuilt}.embed.d\n'
                            f'{rebuilt}:\n\t{shlex.join(command)}\n')
        def make():
            subprocess.run(['make', '-s', '-f', str(makefile)], check=True)
            return hashlib.sha256(rebuilt.read_bytes()).digest()
        previous = make()
        for change in ('content', 'add', 'remove'):
            if change == 'content':
                (copied / 'resources/space file.txt').write_text('modified resource\n')
            elif change == 'add':
                (empty / 'new.txt').write_text('new matching file\n')
            else:
                (empty / 'new.txt').unlink()
            current = make()
            if current == previous:
                raise RuntimeError('make did not rebuild after embedded resource ' + change)
            previous = current
    after = {str(path): hashlib.sha256(path.read_bytes()).hexdigest()
             for path in source.parent.rglob('*') if path.is_file()}
    if before != after:
        raise RuntimeError('compiler changed original sources or resources')
    print('PASS: embed with generics; host/default/O2; 386 object; resource changes/additions/removals; unchanged sources')


if __name__ == '__main__':
    main()
