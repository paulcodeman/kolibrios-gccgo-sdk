#!/usr/bin/env python3
"""Check stable function type exports and a separately compiled consumer."""
import argparse
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', required=True)
    args = parser.parse_args()
    compiler = str(Path(args.compiler).resolve())
    source = Path(__file__).with_name('tests') / 'export-types/regression/exporttypes/export.go'
    with tempfile.TemporaryDirectory(prefix='gccgo-export-types-') as temporary:
        directory = Path(temporary)
        base = directory / 'regression/exporttypes'
        base.parent.mkdir(parents=True)
        flags = ['-m32', '-c', '-Os', '-fno-split-stack', '-I' + str(directory)]
        exports = []
        for iteration in range(2):
            obj = directory / ('owner' + str(iteration) + '.o')
            subprocess.run([compiler, *flags, '-fgo-pkgpath=regression/exporttypes',
                            str(source.resolve()), '-o', str(obj)], check=True)
            raw = directory / ('exports' + str(iteration))
            subprocess.run(['objcopy', '--dump-section', '.go_export=' + str(raw), str(obj)], check=True)
            exports.append(raw.read_bytes())
            subprocess.run(['objcopy', '-j', '.go_export', str(obj), str(base) + '.gox'], check=True)
        if exports[0] != exports[1]:
            raise RuntimeError('export data changed between identical builds')
        consumer = directory / 'consumer.go'
        consumer.write_text('''package consumer
import "regression/exporttypes"
var Callback func() = exporttypes.Callback
var Variadic func(...int) int = exporttypes.Variadic
var Slice func([]int) int = exporttypes.Slice
var Result = exporttypes.Clamp(8, 1, 4)
''')
        subprocess.run([compiler, *flags, str(consumer), '-o', str(directory / 'consumer.o')], check=True)
        consumer.write_text('''package consumer
import "regression/exporttypes"
var Invalid func([]int) int = exporttypes.Variadic
''')
        result = subprocess.run([compiler, *flags, str(consumer), '-o', str(directory / 'invalid.o')],
                                capture_output=True, text=True)
        if result.returncode == 0 or not any(word in result.stderr for word in ('incompatible', 'cannot use')):
            raise RuntimeError('variadic and slice signatures merged: ' + result.stderr)
    print('PASS: stable exports, consumer imports and distinct variadic signatures')


if __name__ == '__main__':
    main()
