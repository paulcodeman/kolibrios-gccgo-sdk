#!/usr/bin/env python3
"""Stage pinned OpenCode with audited adapters and build it for KolibriOS."""

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

from go_file_filter import list_package_go_files

_spec = importlib.util.spec_from_file_location('resolve_packages', Path(__file__).with_name('resolve-packages.py'))
_resolver = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_resolver)
find_pkg_dir = _resolver.find_pkg_dir
parse_imports = _resolver.parse_imports

_compat_spec = importlib.util.spec_from_file_location('kolibrios_compat', Path(__file__).with_name('stage-kolibrios-compat.py'))
_compat = importlib.util.module_from_spec(_compat_spec)
_compat_spec.loader.exec_module(_compat)


ROOT = Path(__file__).resolve().parent.parent
CACHE = ROOT / '.build-cache'
UPSTREAM = CACHE / 'opencode-upstream'
STAGE = CACHE / 'opencode'
REVISION = '73ee493265acf15fcd8caab2bc8cd3bd375b63cb'
MODULE = 'github.com/opencode-ai/opencode'

# One portable UI correction, deliberately outside the SDK/runtime. The pinned
# checkout remains pristine; all changes to the staged application are audited.
STATUS_SOURCE = Path('internal/tui/components/core/status.go')
STATUS_EDITS = (
    ('\t"github.com/charmbracelet/lipgloss"\n',
     '\t"github.com/charmbracelet/lipgloss"\n\t"github.com/charmbracelet/x/ansi"\n'),
    ('\t\t\tWidth(availableWidht)\n\n\t\tswitch',
     '\t\t\tWidth(availableWidht).\n\t\t\tMaxHeight(1)\n\n\t\tswitch'),
    ('\t\tinfoWidth := availableWidht - 10\n'
     '\t\t// Truncate message if it\'s longer than available width\n'
     '\t\tmsg := m.info.Msg\n'
     '\t\tif len(msg) > infoWidth && infoWidth > 0 {\n'
     '\t\t\tmsg = msg[:infoWidth] + "..."\n\t\t}\n',
     '\t\tinfoWidth := max(0, availableWidht-infoStyle.GetHorizontalFrameSize())\n'
     '\t\tmsg := ansi.Truncate(m.info.Msg, infoWidth, "…")\n'),
    ('\treturn status\n}\n', '\treturn ansi.Truncate(status, max(0, m.width), "")\n}\n'),
)


def application_source_bytes(relative, original):
    if relative != STATUS_SOURCE:
        return original
    text = original.decode('utf-8')
    for before, after in STATUS_EDITS:
        if text.count(before) != 1:
            raise RuntimeError('status-width patch no longer matches pinned upstream')
        text = text.replace(before, after)
    return text.encode('utf-8')


def stage_application_sources():
    original = (UPSTREAM / STATUS_SOURCE).read_bytes()
    expected = application_source_bytes(STATUS_SOURCE, original)
    destination = STAGE / 'third_party' / MODULE / STATUS_SOURCE
    if destination.read_bytes() not in (original, expected):
        raise RuntimeError('refusing to overwrite modified application source: ' + str(STATUS_SOURCE))
    if destination.read_bytes() != expected:
        destination.write_bytes(expected)
    (STAGE / 'application-source-compat.json').write_text(json.dumps({str(STATUS_SOURCE): {
        'upstream_sha256': hashlib.sha256(original).hexdigest(),
        'patched_sha256': hashlib.sha256(expected).hexdigest(),
        'reason': 'keep narrow-terminal status notifications on one row; preserve ANSI and Unicode',
    }}, indent=2) + '\n')


def run(args, **kwargs):
    print('+', ' '.join(map(str, args)), flush=True)
    return subprocess.run(list(map(str, args)), check=True, **kwargs)


def verify_upstream():
    revision = subprocess.check_output(['git', '-C', str(UPSTREAM), 'rev-parse', 'HEAD'], text=True).strip()
    dirty = subprocess.check_output(['git', '-C', str(UPSTREAM), 'status', '--porcelain', '--untracked-files=no'], text=True).strip()
    if revision != REVISION or dirty:
        raise RuntimeError('OpenCode checkout must be clean at pinned revision ' + REVISION)


def verify_staged_sources():
    destination = STAGE / 'third_party' / MODULE
    sources = {p.relative_to(UPSTREAM): p for p in UPSTREAM.rglob('*.go')
               if '.git' not in p.relative_to(UPSTREAM).parts
               and 'vendor' not in p.relative_to(UPSTREAM).parts}
    staged = {p.relative_to(destination): p for p in destination.rglob('*.go')}
    if sources.keys() != staged.keys():
        raise RuntimeError('staged OpenCode Go file inventory differs from upstream')
    for relative, source in sources.items():
        if application_source_bytes(relative, source.read_bytes()) != staged[relative].read_bytes():
            raise RuntimeError('staged OpenCode source was changed: ' + str(relative))
    if (STAGE / 'app/main.go').read_bytes() != (UPSTREAM / 'main.go').read_bytes():
        raise RuntimeError('OpenCode entrypoint differs from upstream')
    print('Verified ' + str(len(sources)-1) + ' unchanged OpenCode Go files and one audited status-width patch', flush=True)


def stage_dependency_compat():
    """Apply the SDK's shared adapters while keeping OpenCode sources unchanged."""
    _compat.stage(STAGE / 'vendor', STAGE / 'dependency-compat.json')


def stage_application_compat():
    """Add the native launch wrapper while preserving the upstream entrypoint."""
    record_path = STAGE / 'application-compat.json'
    previous = json.loads(record_path.read_text()) if record_path.exists() else {}
    font = CACHE / 'console-unifont.kbf'
    run([sys.executable, ROOT / 'tooling/build-unifont.py', font])
    records = {}
    for source in (*sorted((ROOT / 'apps/opencode/bootstrap').glob('*.go')), font):
        destination = STAGE / 'app' / source.name
        data = source.read_bytes()
        if destination.exists() and destination.read_bytes() != data:
            actual = hashlib.sha256(destination.read_bytes()).hexdigest()
            if previous.get(destination.name) != actual:
                raise RuntimeError('refusing to overwrite a modified application adapter: ' + destination.name)
        destination.parent.mkdir(parents=True, exist_ok=True)
        if not destination.exists() or destination.read_bytes() != data:
            destination.write_bytes(data)
        records[destination.name] = hashlib.sha256(data).hexdigest()
    record_path.write_text(json.dumps(records, indent=2) + '\n')


def prepare(host_go):
    CACHE.mkdir(exist_ok=True)
    if not UPSTREAM.exists():
        run(['git', 'init', UPSTREAM])
        run(['git', '-C', UPSTREAM, 'remote', 'add', 'origin', 'https://github.com/opencode-ai/opencode.git'])
        run(['git', '-C', UPSTREAM, 'fetch', '--depth=1', 'origin', REVISION])
        run(['git', '-C', UPSTREAM, 'checkout', '--detach', 'FETCH_HEAD'])
    verify_upstream()
    destination = STAGE / 'third_party' / MODULE
    shutil.copytree(UPSTREAM, destination, dirs_exist_ok=True,
                    ignore=shutil.ignore_patterns('.git', 'vendor'))
    run([host_go, 'mod', 'vendor', '-o', STAGE / 'vendor'], cwd=UPSTREAM)
    stage_dependency_compat()
    app = STAGE / 'app'
    app.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(UPSTREAM / 'main.go', app / 'main.go')
    stage_application_sources()
    stage_application_compat()
    stage_makefile()
    print('Prepared OpenCode at ' + REVISION + ' with one audited status-width correction')


def stage_makefile():
    # Compatibility entrypoint for old commands; compilation/output live in
    # apps/opencode just like the other SDK applications.
    path = STAGE / 'app/Makefile'
    data = ('.PHONY: all build clean clean-cache distclean print-output\n'
            'all build clean clean-cache distclean print-output:\n'
            '\t+$(MAKE) -C ' + str(ROOT / 'apps/opencode') + ' $@\n').encode()
    if not path.exists() or path.read_bytes() != data:
        path.write_bytes(data)


def stage(host_go):
    if not all(path.exists() for path in
               (UPSTREAM / 'go.mod', STAGE / 'app/main.go', STAGE / 'vendor/modules.txt')):
        prepare(host_go)
    verify_upstream()
    stage_application_sources()
    verify_staged_sources()
    stage_application_compat()
    stage_makefile()


def audit():
    verify_upstream()
    stage_application_sources()
    verify_staged_sources()
    stage_dependency_compat()
    first = [str(ROOT / 'platform')]
    third = [str(STAGE / 'third_party'), str(STAGE / 'vendor'), str(ROOT / 'third_party')]
    missing, empty, visited = {}, {}, {}

    def visit(pkg, parent):
        if pkg == 'unsafe' or pkg in visited:
            return
        directory = find_pkg_dir(str(ROOT), str(ROOT / 'stdlib'), first, third, pkg)
        if directory is None:
            missing.setdefault(pkg, set()).add(parent)
            return
        files = list_package_go_files(directory, 'kolibrios', '386', ['gccgo'])
        visited[pkg] = files
        if not files:
            empty.setdefault(pkg, set()).add(parent)
        for source in files:
            for imp in parse_imports(source):
                if imp == 'C':
                    missing.setdefault('C (cgo unsupported by bootstrap)', set()).add(pkg)
                else:
                    visit(imp, pkg)

    for source in list_package_go_files(str(STAGE / 'app'), 'kolibrios', '386', ['gccgo']):
        for pkg in parse_imports(source):
            visit(pkg, 'main')
    report = {
        'upstream_revision': REVISION,
        'target': 'kolibrios/386',
        'selected_packages': len(visited),
        'selected_go_files': sum(map(len, visited.values())),
        'missing_packages': {pkg: sorted(parents) for pkg, parents in sorted(missing.items())},
        'packages_without_target_sources': {pkg: sorted(parents) for pkg, parents in sorted(empty.items())},
        'upstream_main_sha256': hashlib.sha256((UPSTREAM / 'main.go').read_bytes()).hexdigest(),
        'staged_main_sha256': hashlib.sha256((STAGE / 'app/main.go').read_bytes()).hexdigest(),
        'unchanged_application_files': 139,
        'application_patches': [STATUS_SOURCE.as_posix()],
    }
    (STAGE / 'audit.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report, indent=2))
    return int(bool(missing or empty))


PROBES = {
    'any': 'package probe\nfunc F(v any) any { return v }\n',
    'generics': 'package probe\nfunc F[T any](v T) T { return v }\nvar V = F[int](1)\n',
    'minmax': 'package probe\nfunc F(a, b int) int { return max(a, min(b, 10)) }\n',
    'clear': 'package probe\nfunc F(m map[string]int, s []int) { clear(m); clear(s) }\n',
    'rangeint': 'package probe\nfunc F(n int) { for range n {} }\n',
    'rangefunc': 'package probe\nfunc Each(yield func(int) bool) { yield(1) }\nfunc F() int { n := 0; for v := range Each { n += v }; return n }\n',
}


def probe(compiler):
    directory = STAGE / 'probes'
    directory.mkdir(parents=True, exist_ok=True)
    report = {}
    for name, source in PROBES.items():
        path = directory / (name + '.go')
        path.write_text(source)
        result = subprocess.run([compiler, '-m32', '-c', '-fno-split-stack', str(path), '-o', str(directory / (name + '.o'))], capture_output=True, text=True)
        report[name] = {'supported': result.returncode == 0, 'diagnostic': result.stderr}
        print(name + ': ' + ('PASS' if result.returncode == 0 else 'BLOCKED'))
    (directory / 'results.json').write_text(json.dumps(report, indent=2) + '\n')
    return int(any(not result['supported'] for result in report.values()))


def build(compiler, smoke=False, jobs=4):
    verify_upstream()
    stage_application_sources()
    verify_staged_sources()
    stage_dependency_compat()
    target = ROOT / 'apps/tests/opencode/smoke' if smoke else ROOT / 'apps/opencode'
    if not smoke:
        stage_application_compat()
    log_path = STAGE / ('smoke-build.log' if smoke else 'build.log')
    with log_path.open('w') as log:
        command = ['make', '-j' + str(jobs), '-C', str(target),
                   'PARALLEL_PACKAGES=1', 'GO=' + compiler,
                   'THIRD_PARTY_DIRS=' + ' '.join(str(path) for path in
                       (STAGE / 'third_party', STAGE / 'vendor', ROOT / 'third_party'))]
        if smoke:
            command.append('KPACK=1')
        else:
            # Upstream's loader refuses raw files larger than 16 MiB. KPACK
            # compresses the complete CLI without trimming original packages.
            command.append('KPACK=1')
            command.append('APP_MAIN_SYMBOL=main.kolibriMain')
            command.append('DEBUG_ELF=opencode.kex.debug.elf')
        environment = os.environ.copy()
        # Each parallel host compiler helper needs only a small worker pool.
        # Respect explicit host tuning; this never changes target runtime settings.
        environment.setdefault('GOMAXPROCS', '2')
        result = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT,
                                env=environment)
    print('Build log: ' + str(log_path))
    if result.returncode:
        print('\n'.join(log_path.read_text().splitlines()[-35:]))
    else:
        print('Built: ' + str(target / ('opencode-smoke.kex' if smoke else 'opencode.kex')))
    return result.returncode


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['prepare', 'stage', 'audit', 'probe', 'build', 'smoke'])
    parser.add_argument('--host-go', default=os.environ.get('HOST_GO', 'go'))
    parser.add_argument('--compiler', default=os.environ.get('GCCGO', str(ROOT / 'tooling/gccgo/gccgo-kolibri')))
    parser.add_argument('--jobs', type=int, default=4)
    args = parser.parse_args()
    if args.jobs < 1:
        parser.error('--jobs must be positive')
    if args.action not in ('prepare', 'stage', 'probe') and not (STAGE / 'app/main.go').is_file():
        parser.error('run prepare first to stage pinned OpenCode sources and dependencies')
    if args.action == 'prepare':
        prepare(args.host_go)
        return 0
    if args.action == 'stage':
        stage(args.host_go)
        return 0
    if args.action == 'audit':
        return audit()
    if args.action == 'probe':
        return probe(args.compiler)
    return build(args.compiler, args.action == 'smoke', args.jobs)


if __name__ == '__main__':
    sys.exit(main())
