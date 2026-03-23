#!/usr/bin/env python3
import argparse
import os
import re
import sys


IMPORT_RE = re.compile(r"(?m)^\s*import\s*(\([^)]*\)|\"[^\"]+\"|`[^`]+`)")


def parse_imports(path: str):
    with open(path, "r", encoding="utf-8", errors="ignore") as f:
        data = f.read()
    data = re.sub(r"//.*", "", data)
    data = re.sub(r"/\*.*?\*/", "", data, flags=re.S)
    imports = []
    for block in IMPORT_RE.finditer(data):
        text = block.group(1)
        if text.startswith("("):
            for m in re.finditer(r"\"([^\"]+)\"", text):
                imports.append(m.group(1))
            for m in re.finditer(r"`([^`]+)`", text):
                imports.append(m.group(1))
        else:
            m = re.match(r"[\"`]([^\"`]+)[\"`]", text)
            if m:
                imports.append(m.group(1))
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


def list_go_files(pkg_dir: str):
    source_dirs = [pkg_dir]
    dirs_file = os.path.join(pkg_dir, "package_dirs.txt")
    if os.path.exists(dirs_file):
        with open(dirs_file, "r", encoding="utf-8", errors="ignore") as f:
            for raw in f:
                value = raw.split("#", 1)[0].strip()
                if not value:
                    continue
                source_dirs.append(os.path.join(pkg_dir, value))
    files = []
    for source_dir in source_dirs:
        if not os.path.isdir(source_dir):
            continue
        files.extend(
            os.path.join(source_dir, name)
            for name in os.listdir(source_dir)
            if name.endswith(".go") and not name.endswith("_test.go")
        )
    return files


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True)
    parser.add_argument("--stdlib", required=True)
    parser.add_argument("--app-dir", required=True)
    parser.add_argument("--packages", default="")
    parser.add_argument("--first-party", default="")
    parser.add_argument("--third-party", default="")
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
    for name in os.listdir(args.app_dir):
        if not name.endswith(".go"):
            continue
        if name.endswith("_test.go"):
            continue
        seeds.update(parse_imports(os.path.join(args.app_dir, name)))

    # add explicit packages
    for pkg in args.packages.split():
        seeds.add(pkg)

    builtin = {"unsafe"}

    visited = set()
    visiting = set()
    order = []

    def visit(pkg: str):
        if pkg in builtin:
            return
        if pkg in visited:
            return
        if pkg in visiting:
            return
        pkg_dir = find_pkg_dir(root, stdlib, first_party_roots, third_party_roots, pkg)
        if not pkg_dir:
            return
        visiting.add(pkg)
        for go_file in list_go_files(pkg_dir):
            for imp in parse_imports(go_file):
                if imp in builtin or imp == "C":
                    continue
                visit(imp)
        visiting.remove(pkg)
        visited.add(pkg)
        order.append(pkg)

    for pkg in sorted(seeds):
        visit(pkg)

    print(" ".join(order))


if __name__ == "__main__":
    main()
