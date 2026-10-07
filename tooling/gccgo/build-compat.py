#!/usr/bin/env python3
"""Build the Go-derived generic lowering stage with an upstream Go host compiler."""

import argparse
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--host-go', default=os.environ.get('HOST_GO', 'go'))
    root = Path(__file__).resolve().parents[2]
    parser.add_argument('--build', type=Path, default=root / '.build-cache/gccgo15-kolibri')
    args = parser.parse_args()
    output = args.build.resolve() / 'compat/go2go'
    output.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run([args.host_go, 'build', '-o', str(output), '.'],
                   cwd=Path(__file__).with_name('compat'), check=True)
    print('Built generic compiler stage: ' + str(output))


if __name__ == '__main__':
    main()
