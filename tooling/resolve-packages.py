#!/usr/bin/env python3
import argparse
import ast
import hashlib
import json
import os
import re
import sys
from pathlib import Path

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from go_file_filter import list_package_go_files


GO_TOKEN = re.compile(r'\s+|//[^\n]*|/\*.*?\*/|"(?:\\.|[^"\\])*"|`[^`]*`|\w+|[^\s]', re.S)


def parse_imports(path: str):
    with open(path, "r", encoding="utf-8", errors="ignore") as f:
        data = f.read()
    # Imports precede all other declarations. Read lexical tokens in that
    # header, so comments and Go source templates inside strings do not add
    # dependencies or invent import cycles.
    tokens = (m.group() for m in GO_TOKEN.finditer(data.lstrip('\ufeff'))
              if not m.group().isspace() and not m.group().startswith(('//', '/*')))
    token = next(tokens, '')
    if token != 'package':
        return []
    next(tokens, '')  # package name
    token = next(tokens, '')
    imports = []

    def add_path(value):
        if value.startswith('`'):
            imports.append(value[1:-1].replace('\r', ''))
        elif value.startswith('"'):
            imports.append(ast.literal_eval(value))

    while token in (';', 'import'):
        if token == ';':
            token = next(tokens, '')
            continue
        token = next(tokens, '')
        if token == '(':
            token = next(tokens, '')
            while token not in ('', ')'):
                add_path(token)
                token = next(tokens, '')
        else:
            if not token.startswith(('"', '`')):
                token = next(tokens, '')  # alias or dot
            add_path(token)
        token = next(tokens, '')
    return imports


def find_pkg_dir(
    root: str, stdlib: str, first_party_roots, third_party_roots, import_path: str
):
    if not import_path:
        return None
    if import_path == "C":
        return None
    candidates = [root]
    candidates.extend(first_party_roots)
    candidates.extend(third_party_roots)
    candidates.append(stdlib)
    for base in candidates:
        if not base:
            continue
        abs_path = os.path.join(base, import_path)
        if os.path.isdir(abs_path):
            return abs_path
    return None


def source_selection(artifacts, package, sources, target):
    """Invalidate an object when its selected files change, even if older."""
    prefix = Path(artifacts) / package
    selection = Path(str(prefix) + '.sources.json')
    content = json.dumps({'target': target, 'sources': sources}, sort_keys=True) + '\n'
    if selection.exists() and selection.read_text() == content:
        return selection
    # Upgrade existing verified compiler records without rebuilding coherent
    # packages. A different vendor root or file set cannot pass this check.
    previous_time = None
    if not selection.exists():
        metadata = Path(str(prefix) + '.generics.json')
        obj = Path(str(prefix) + '.gccgo.go.o')
        if metadata.exists() and obj.exists():
            try:
                record = json.loads(metadata.read_text())
                if record.get('sources') == sources and all(
                    record.get('source_sha256', {}).get(name) == hashlib.sha256(Path(name).read_bytes()).hexdigest()
                    for name in sources
                ):
                    previous_time = obj.stat().st_mtime_ns
            except (OSError, ValueError):
                pass
    selection.parent.mkdir(parents=True, exist_ok=True)
    selection.write_text(content)
    if previous_time is not None:
        os.utime(selection, ns=(previous_time, previous_time))
    return selection


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True)
    parser.add_argument("--stdlib", required=True)
    parser.add_argument("--app-dir", required=True)
    parser.add_argument("--packages", default="")
    parser.add_argument("--first-party", default="")
    parser.add_argument("--third-party", default="")
    parser.add_argument("--goos", default="kolibrios")
    parser.add_argument("--goarch", default="386")
    parser.add_argument("--tags", default="gccgo")
    parser.add_argument("--make-deps", help="write selected package imports as Make variables")
    parser.add_argument("--artifact-root", help="track selected source identities in the shared package cache")
    args = parser.parse_args()

    root = args.root
    stdlib = args.stdlib
    first_party = [item for item in args.first_party.split() if item]
    third_party = [item for item in args.third_party.split() if item]

    def normalize_roots(items):
        roots = []
        for item in items:
            if os.path.isabs(item):
                roots.append(item)
            else:
                roots.append(os.path.join(root, item))
        return roots

    first_party_roots = normalize_roots(first_party)
    third_party_roots = normalize_roots(third_party)

    # seed imports from app sources
    seeds = set()
    for go_file in list_package_go_files(
        args.app_dir, args.goos, args.goarch,
        [item for item in args.tags.split() if item],
    ):
        seeds.update(parse_imports(go_file))

    # add explicit packages
    for pkg in args.packages.split():
        seeds.add(pkg)

    builtin = {"unsafe"}

    visited = set()
    visiting = set()
    order = []
    package_imports = {}
    package_sources = {}
    missing = {}
    empty = set()

    def visit(pkg: str, parent='application or explicit PACKAGE_DIRS'):
        if pkg in builtin:
            return
        if pkg in visited:
            return
        if pkg in visiting:
            return
        pkg_dir = find_pkg_dir(root, stdlib, first_party_roots, third_party_roots, pkg)
        if not pkg_dir:
            missing.setdefault(pkg, set()).add(parent)
            return
        visiting.add(pkg)
        imports = set()
        sources = list_package_go_files(
            pkg_dir,
            args.goos,
            args.goarch,
            [item for item in args.tags.split() if item],
        )
        package_sources[pkg] = sources
        if not sources:
            empty.add(pkg)
        for go_file in sources:
            for imp in parse_imports(go_file):
                if imp in builtin or imp == "C":
                    continue
                imports.add(imp)
                visit(imp, pkg)
        package_imports[pkg] = imports
        visiting.remove(pkg)
        visited.add(pkg)
        order.append(pkg)

    for pkg in sorted(seeds):
        visit(pkg)

    if missing or empty:
        for pkg, parents in sorted(missing.items()):
            print('missing Go package ' + pkg + ' (imported by ' + ', '.join(sorted(parents)) + ')', file=sys.stderr)
        for pkg in sorted(empty):
            print('Go package has no sources for ' + args.goos + '/' + args.goarch + ': ' + pkg, file=sys.stderr)
        return 1

    if args.make_deps:
        path = Path(args.make_deps)
        path.parent.mkdir(parents=True, exist_ok=True)
        content = "# Generated from selected target sources; do not edit.\n"
        for pkg in order:
            imports = sorted(package_imports[pkg] & visited)
            content += "PACKAGE_IMPORTS_" + pkg + " := " + " ".join(imports) + "\n"
            # The resolver already selected these exact target files. Reuse
            # that result instead of launching another Python process for
            # each of OpenCode's hundreds of packages while parsing Make.
            content += "PACKAGE_RESOLVED_SOURCES_" + pkg + " := " + " ".join(package_sources[pkg]) + "\n"
            if args.artifact_root:
                selection = source_selection(args.artifact_root, pkg, package_sources[pkg],
                                             [args.goos, args.goarch, sorted(args.tags.split())])
                content += "PACKAGE_SOURCE_SELECTION_" + pkg + " := " + str(selection) + "\n"
        if not path.exists() or path.read_text() != content:
            path.write_text(content)

    print(" ".join(order))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
