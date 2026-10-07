#!/usr/bin/env python3
"""Build a locally patched GCC 15.2.0 Go frontend without installing it."""

import argparse
import os
from pathlib import Path
import subprocess
import tempfile


def apply_patches(source, patches):
    """Validate the patch stack, including overlapping already-applied patches."""
    originals = {}
    for patch in patches:
        for line in patch.read_text().splitlines():
            if line.startswith('+++ b/'):
                relative = Path(line[6:].split('\t', 1)[0])
                if relative.is_absolute() or '..' in relative.parts:
                    raise RuntimeError('invalid patch path: ' + str(relative))
                originals[relative] = (source / relative).read_bytes()
    with tempfile.TemporaryDirectory(prefix='gccgo-patch-check-') as directory:
        staged = Path(directory)
        for applied in range(len(patches), -1, -1):
            for relative, content in originals.items():
                path = staged / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(content)
            commands = [(patch, '--reverse') for patch in reversed(patches[:applied])]
            commands.extend((patch, '--forward') for patch in patches)
            for patch, direction in commands:
                result = subprocess.run(['patch', '--batch', '--force', direction, '-p1', '-i', str(patch.resolve())],
                                        cwd=staged, capture_output=True)
                if result.returncode:
                    break
            else:
                for patch in patches[applied:]:
                    subprocess.run(['patch', '--batch', '--forward', '-p1', '-i', str(patch.resolve())],
                                   cwd=source, check=True)
                print('Verified patch stack: ' + str(applied) + ' already applied, ' +
                      str(len(patches) - applied) + ' added', flush=True)
                return
    raise RuntimeError('source does not match an unpatched tree or an applied prefix of the patch stack')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True, help='extracted GCC 15.2.0 source tree')
    parser.add_argument('--build', type=Path, required=True, help='separate build directory')
    parser.add_argument('--prereq-prefix', type=Path, help='optional prefix containing usr/include and usr/lib/x86_64-linux-gnu')
    parser.add_argument('--jobs', type=int, default=4)
    args = parser.parse_args()
    source, build = args.source.resolve(), args.build.resolve()
    if source == build or not (source / 'gcc/go/gofrontend/expressions.cc').exists():
        parser.error('provide a GCC source tree and a separate build directory')
    version = (source / 'gcc/BASE-VER').read_text().strip()
    if version != '15.2.0':
        parser.error('patches are tested against GCC 15.2.0, found ' + version)
    apply_patches(source, sorted(Path(__file__).with_name('patches').glob('*.patch')))
    build.mkdir(parents=True, exist_ok=True)
    if not (build / 'config.status').exists():
        options = ['--disable-bootstrap', '--enable-languages=c,go', '--disable-multilib',
                   '--disable-libsanitizer', '--disable-libquadmath', '--disable-libgomp',
                   '--disable-libatomic', '--without-isl', '--disable-nls']
        if args.prereq_prefix:
            prefix = args.prereq_prefix.resolve() / 'usr'
            for library in ('gmp', 'mpfr', 'mpc'):
                options.extend(['--with-' + library + '-include=' + str(prefix / 'include'),
                                '--with-' + library + '-lib=' + str(prefix / 'lib/x86_64-linux-gnu')])
        subprocess.run([str(source / 'configure')] + options, cwd=build, check=True)
    env = os.environ.copy()
    if args.prereq_prefix:
        lib_path = str(args.prereq_prefix.resolve() / 'usr/lib/x86_64-linux-gnu')
        env['LIBRARY_PATH'] = lib_path + (':' + env['LIBRARY_PATH'] if env.get('LIBRARY_PATH') else '')
    subprocess.run(['make', '-j' + str(args.jobs), 'all-gcc', 'MAKEINFO=true'], cwd=build, env=env, check=True)
    print('Use GCCGO_BUILD_DIR=' + str(build) + ' with tooling/gccgo/gccgo-kolibri')


if __name__ == '__main__':
    main()
