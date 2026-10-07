#!/usr/bin/env python3
"""Check GCC method table ownership using native 386 objects and real OTel exports."""

import argparse
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]


def symbols(path):
    result = {}
    for line in subprocess.check_output(['nm', str(path)], text=True).splitlines():
        fields = line.split()
        if len(fields) >= 2:
            result[fields[-1]] = fields[-2]
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler-prefix', type=Path,
                        default=ROOT / '.build-cache/gccgo15-kolibri/gcc')
    parser.add_argument('--packages', type=Path, default=ROOT / '.build-cache/opencode/pkg')
    args = parser.parse_args()
    if not (args.packages / 'go.opentelemetry.io/auto/sdk.gox').is_file():
        parser.error('compile the original OpenCode dependencies first; this checks their actual OTel export')
    base = ['gccgo-15', '-B' + str(args.compiler_prefix.resolve()) + '/',
            '-m32', '-c', '-Os', '-nostdinc', '-nostdlib', '-fno-split-stack', '-fexceptions',
            '-I' + str(args.packages.resolve())]
    with tempfile.TemporaryDirectory(prefix='gccgo-interface-tables-') as temporary:
        folder = Path(temporary)

        def compile(name, text, package='consumer', valid=True):
            source, obj = folder / (name + '.go'), folder / (name + '.o')
            source.write_text(text)
            result = subprocess.run(base + ['-I' + str(folder), '-fgo-pkgpath=' + package,
                                          '-o', str(obj), str(source)], capture_output=True, text=True)
            if valid and result.returncode:
                raise RuntimeError(result.stderr)
            if not valid:
                assert result.returncode and 'hidden' in result.stderr, result.stderr
            return obj

        obj = compile('otel', '''package consumer
import (
 "go.opentelemetry.io/auto/sdk"
 "go.opentelemetry.io/otel/trace"
)
func Tracer(name string) trace.Tracer { return sdk.TracerProvider().Tracer(name) }
''')
        tables = {name: kind for name, kind in symbols(obj).items()
                  if name.startswith('pimt..') and name.endswith('..go_0opentelemetry_0io_1auto_1sdk.tracerProvider')}
        assert all(kind != 'U' for kind in tables.values()), tables
        # A local immutable table may disappear after devirtualizing Tracer.
        assert any(name.startswith('go_0opentelemetry_0io_1auto_1sdk.') for name in symbols(obj)), symbols(obj)
        provider_symbols = symbols(args.packages / 'go.opentelemetry.io/auto/sdk.gccgo.o')
        for name, kind in symbols(obj).items():
            if kind == 'U' and name.startswith('go_0opentelemetry_0io_1auto_1sdk.'):
                definition = provider_symbols.get(name)
                assert definition and definition != 'U' and definition.isupper(), 'missing real provider method: ' + name

        owner = compile('owner', '''package owner
type Sealed interface { hidden(); Value() int }
type private struct{}
func (private) hidden() {}
func (private) Value() int { return 9 }
var instance = new(private)
func New() Sealed { return instance }
''', 'regression/owner')
        export = folder / 'regression/owner.gox'
        export.parent.mkdir()
        subprocess.run(['objcopy', '-j', '.go_export', str(owner), str(export)], check=True)
        client = compile('owned_client', '''package consumer
import "regression/owner"
func Value() int { return owner.New().Value() }
''')
        owner_tables = {name: kind for name, kind in symbols(owner).items() if name.startswith('pimt..')}
        assert owner_tables and any(kind == 'R' for kind in owner_tables.values()), owner_tables
        for name, kind in symbols(client).items():
            if name.startswith('pimt..') and kind == 'U':
                assert owner_tables.get(name) == 'R', 'owned public table missing: ' + name
        compile('invalid', '''package invalid
import "regression/owner"
type imitation struct{}
func (imitation) hidden() {}
func (imitation) Value() int { return 9 }
var _ owner.Sealed = imitation{}
''', 'invalid', valid=False)
    print('PASS: foreign hidden-method tables defined locally, own-package tables shared, private method ownership enforced')


if __name__ == '__main__':
    main()
