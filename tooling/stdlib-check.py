#!/usr/bin/env python3
"""Compile imported OpenCode stdlib packages against the KolibriOS SDK.

This checks target package objects, not linked applications or runtime behavior.
Each root is checked independently so one blocker cannot hide the other results.
"""

import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess

from go_file_filter import list_package_go_files


ROOT = Path(__file__).resolve().parent.parent
CACHE = ROOT / '.build-cache/opencode-stdlib'


def invalidate_changed_packages(packages, compiler):
    """Use source/import fingerprints before make's FAST_PKG order-only mode."""
    spec = importlib.util.spec_from_file_location('resolver', ROOT / 'tooling/resolve-packages.py')
    resolver = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(resolver)
    stamps_path = CACHE / 'fingerprints.json'
    previous = json.loads(stamps_path.read_text()) if stamps_path.exists() else {}
    stamps, visiting = {}, set()
    compiler_id = compiler + subprocess.check_output([compiler, '--version'], text=True)
    frontend = ROOT / '.build-cache/gccgo15-kolibri/gcc/go1'
    if frontend.exists():
        compiler_id += str(frontend.stat().st_mtime_ns)
    compat = ROOT / 'tooling/gccgo/compat'
    for source in sorted(compat.rglob('*.go')):
        compiler_id += str(source.relative_to(ROOT)) + hashlib.sha256(source.read_bytes()).hexdigest()
    compiler_id += hashlib.sha256((ROOT / 'tooling/gccgo/compiler-driver.py').read_bytes()).hexdigest()

    def visit(pkg):
        if pkg == 'unsafe':
            return 'builtin'
        if pkg in stamps:
            return stamps[pkg]
        if pkg in visiting:
            raise RuntimeError('import cycle at ' + pkg)
        directory = resolver.find_pkg_dir(str(ROOT), str(ROOT / 'stdlib'),
                                          [str(ROOT / 'platform')], [str(ROOT / 'third_party')], pkg)
        if directory is None:
            return 'missing:' + pkg
        visiting.add(pkg)
        digest = hashlib.sha256((compiler_id + ':kolibrios:386:gccgo').encode())
        imports = set()
        for file in list_package_go_files(directory, 'kolibrios', '386', ['gccgo']):
            digest.update(Path(file).name.encode())
            digest.update(Path(file).read_bytes())
            imports.update(resolver.parse_imports(file))
        for file in sorted(Path(directory).iterdir()):
            if file.suffix in ('.c', '.h', '.S'):
                digest.update(file.name.encode())
                digest.update(file.read_bytes())
        for imp in sorted(imports):
            digest.update((imp + visit(imp)).encode())
        visiting.remove(pkg)
        stamps[pkg] = digest.hexdigest()
        if previous.get(pkg) != stamps[pkg]:
            for extension in ('.gox', '.gccgo.o', '.gccgo.go.o', '.generics.json'):
                artifact = CACHE / 'pkg' / (pkg + extension)
                if artifact.is_relative_to(CACHE) and artifact.is_file():
                    artifact.unlink()
        return stamps[pkg]

    for pkg in packages:
        visit(pkg)
    previous.update(stamps)
    stamps_path.write_text(json.dumps(previous, indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', default=str(ROOT / 'tooling/gccgo/gccgo-kolibri'))
    parser.add_argument('--smoke', action='store_true', help='compile and link the target execution test')
    parser.add_argument('packages', nargs='*')
    args = parser.parse_args()
    manifest = json.loads((ROOT / 'stdlib/OPENCODE_UPSTREAM.json').read_text())
    smoke_packages = ['encoding/gob', 'flag', 'mime/multipart', 'net/http/httptrace',
                      'net/netip', 'net/textproto', 'os/signal', 'runtime/debug', 'text/template']
    packages = args.packages or (smoke_packages if args.smoke else manifest['requested_packages'])
    driver = CACHE / 'driver'
    driver.mkdir(parents=True, exist_ok=True)
    invalidate_changed_packages(packages, args.compiler)
    (driver / 'main.go').write_text('package main\nfunc main() {}\n')
    (driver / 'Makefile').write_text(
        'PROGRAM = stdlib-check\nGO_PACKAGE = main\nROOT = ' + str(ROOT) + '\n'
        'BUILD_CACHE_ROOT = ' + str(ROOT / '.build-cache') + '\n'
        'BUILD_CACHE_NAMESPACE = opencode-stdlib\n'
        'include $(ROOT)/tooling/kolibri-app.mk\n')
    results = {}
    for pkg in packages:
        log_path = CACHE / (pkg.replace('/', '_') + '.log')
        with log_path.open('w') as log:
            result = subprocess.run(['make', '-C', str(driver), 'PACKAGE_DIRS=' + pkg, 'FAST_PKG=1',
                                     'GO=' + args.compiler, str(CACHE / 'pkg' / (pkg + '.gox'))],
                                    stdout=log, stderr=subprocess.STDOUT)
        results[pkg] = {'compiled': result.returncode == 0, 'log': str(log_path.relative_to(ROOT))}
        print(pkg + ': ' + ('COMPILED' if result.returncode == 0 else 'BLOCKED'), flush=True)
        if result.returncode:
            errors = [line for line in log_path.read_text().splitlines() if 'error:' in line or 'undefined' in line]
            print('\n'.join(errors[:6]), flush=True)
    report_path = CACHE / 'results.json'
    if report_path.exists():
        previous = json.loads(report_path.read_text())
        previous['packages'].update(results)
    else:
        previous = {'target': 'kolibrios/386', 'validation': 'package objects only', 'packages': results}
    previous['compiler'] = args.compiler
    report_path.write_text(json.dumps(previous, indent=2) + '\n')
    failed = any(not result['compiled'] for result in results.values())
    if args.smoke and not failed:
        log_path = CACHE / 'smoke-build.log'
        with log_path.open('w') as log:
            result = subprocess.run(['make', '-C', str(ROOT / 'apps/tests/opencode/stdlib-smoke'),
                                     'GO=' + args.compiler, 'FAST_PKG=1', 'KPACK=1'],
                                    stdout=log, stderr=subprocess.STDOUT)
        failed = result.returncode != 0
        print('Smoke link: ' + ('BLOCKED' if failed else 'PASS'))
        if failed:
            print('\n'.join(log_path.read_text().splitlines()[-25:]))
    return int(failed)


if __name__ == '__main__':
    raise SystemExit(main())
