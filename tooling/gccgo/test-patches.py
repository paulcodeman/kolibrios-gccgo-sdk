#!/usr/bin/env python3
"""Verify fresh, incremental and repeated patch application using a patched source tree."""

import argparse
import importlib.util
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True)
    args = parser.parse_args()
    spec = importlib.util.spec_from_file_location('gccgo_build', Path(__file__).with_name('build.py'))
    builder = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(builder)
    patches = sorted(Path(__file__).with_name('patches').glob('*.patch'))
    expected = {}
    for patch in patches:
        for line in patch.read_text().splitlines():
            if line.startswith('+++ b/'):
                relative = Path(line[6:].split('\t', 1)[0])
                expected[relative] = (args.source / relative).read_bytes()
    for prefix in range(len(patches) + 1):
        with tempfile.TemporaryDirectory(prefix='gccgo-patch-regression-') as directory:
            source = Path(directory)
            for relative, content in expected.items():
                path = source / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(content)
            for patch in reversed(patches[prefix:]):
                subprocess.run(['patch', '--batch', '--force', '--reverse', '-p1', '-i', str(patch.resolve())],
                               cwd=source, check=True, stdout=subprocess.DEVNULL)
            builder.apply_patches(source, patches)
            for relative, content in expected.items():
                if (source / relative).read_bytes() != content:
                    raise RuntimeError('patch output mismatch for prefix ' + str(prefix) + ': ' + str(relative))
    print('PASS: every applied prefix produces identical final compiler sources')


if __name__ == '__main__':
    main()
