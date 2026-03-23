#!/usr/bin/env python3

import os
import re
from dataclasses import dataclass
from typing import Iterable, List, Optional, Sequence, Set


KNOWN_GOOS = {
    "aix",
    "android",
    "darwin",
    "dragonfly",
    "freebsd",
    "hurd",
    "illumos",
    "ios",
    "js",
    "kolibrios",
    "linux",
    "netbsd",
    "openbsd",
    "plan9",
    "solaris",
    "wasip1",
    "windows",
}

KNOWN_GOARCH = {
    "386",
    "amd64",
    "amd64p32",
    "arm",
    "arm64",
    "loong64",
    "mips",
    "mips64",
    "mips64le",
    "mipsle",
    "ppc64",
    "ppc64le",
    "riscv64",
    "s390x",
    "sparc64",
    "wasm",
}

UNIX_GOOS = {
    "aix",
    "android",
    "darwin",
    "dragonfly",
    "freebsd",
    "hurd",
    "illumos",
    "ios",
    "kolibrios",
    "linux",
    "netbsd",
    "openbsd",
    "solaris",
}

GOOS_ALIASES = {
    "android": {"linux"},
    "illumos": {"solaris"},
    "ios": {"darwin"},
}

TOKEN_RE = re.compile(r"\s*(&&|\|\||!|\(|\)|[A-Za-z0-9_\.]+)")


def default_tag_set(goos: str, goarch: str, extra_tags: Optional[Iterable[str]] = None) -> Set[str]:
    tags = {goos, goarch, "gccgo"}
    if goos in UNIX_GOOS:
        tags.add("unix")
    tags.update(GOOS_ALIASES.get(goos, set()))
    for major in range(1, 23):
        tags.add(f"go1.{major}")
    if extra_tags:
        tags.update(tag for tag in extra_tags if tag)
    return tags


@dataclass
class ExprNode:
    kind: str
    value: Optional[str] = None
    left: Optional["ExprNode"] = None
    right: Optional["ExprNode"] = None

    def eval(self, tags: Set[str]) -> bool:
        if self.kind == "tag":
            return self.value in tags
        if self.kind == "not":
            return not self.left.eval(tags)
        if self.kind == "and":
            return self.left.eval(tags) and self.right.eval(tags)
        if self.kind == "or":
            return self.left.eval(tags) or self.right.eval(tags)
        raise ValueError(f"unknown node kind: {self.kind}")


class ExprParser:
    def __init__(self, text: str):
        self.tokens = [m.group(1) for m in TOKEN_RE.finditer(text)]
        self.index = 0

    def peek(self) -> Optional[str]:
        if self.index >= len(self.tokens):
            return None
        return self.tokens[self.index]

    def take(self, want: Optional[str] = None) -> str:
        tok = self.peek()
        if tok is None:
            raise ValueError("unexpected end of build expression")
        if want is not None and tok != want:
            raise ValueError(f"expected {want}, got {tok}")
        self.index += 1
        return tok

    def parse(self) -> ExprNode:
        node = self.parse_or()
        if self.peek() is not None:
            raise ValueError(f"unexpected token {self.peek()}")
        return node

    def parse_or(self) -> ExprNode:
        node = self.parse_and()
        while self.peek() == "||":
            self.take("||")
            node = ExprNode("or", left=node, right=self.parse_and())
        return node

    def parse_and(self) -> ExprNode:
        node = self.parse_unary()
        while self.peek() == "&&":
            self.take("&&")
            node = ExprNode("and", left=node, right=self.parse_unary())
        return node

    def parse_unary(self) -> ExprNode:
        tok = self.peek()
        if tok == "!":
            self.take("!")
            return ExprNode("not", left=self.parse_unary())
        if tok == "(":
            self.take("(")
            node = self.parse_or()
            self.take(")")
            return node
        if tok is None:
            raise ValueError("unexpected end of build expression")
        self.take()
        return ExprNode("tag", value=tok)


def parse_go_build_expr(text: str) -> ExprNode:
    return ExprParser(text.strip()).parse()


def plus_build_matches(lines: Sequence[str], tags: Set[str]) -> bool:
    for raw in lines:
        line = raw.strip()
        if not line:
            continue
        any_term = False
        for term in line.split():
            if not term:
                continue
            all_factor = True
            for factor in term.split(","):
                if not factor:
                    continue
                negated = factor.startswith("!")
                tag = factor[1:] if negated else factor
                matched = tag in tags
                if negated:
                    matched = not matched
                if not matched:
                    all_factor = False
                    break
            if all_factor:
                any_term = True
                break
        if not any_term:
            return False
    return True


def read_build_constraints(path: str):
    go_build = None
    plus_build = []

    with open(path, "r", encoding="utf-8", errors="ignore") as f:
        lines = f.readlines()

    in_header = True
    for raw in lines:
        stripped = raw.strip()
        if not in_header:
            break
        if stripped == "":
            continue
        if stripped.startswith("//"):
            if stripped.startswith("//go:build"):
                go_build = stripped[len("//go:build") :].strip()
            elif stripped.startswith("// +build"):
                plus_build.append(stripped[len("// +build") :].strip())
            continue
        if stripped.startswith("/*"):
            continue
        in_header = False

    return go_build, plus_build


def filename_matches(name: str, goos: str, goarch: str, tags: Set[str]) -> bool:
    if not name.endswith(".go") or name.endswith("_test.go"):
        return False

    stem = name[:-3]
    if stem.endswith("_test"):
        return False

    parts = stem.split("_")
    if len(parts) < 2:
        return True

    tail = parts[-1]
    prev = parts[-2] if len(parts) >= 2 else None

    os_tag = None
    arch_tag = None

    if prev in KNOWN_GOOS.union({"unix"}) and tail in KNOWN_GOARCH:
        os_tag = prev
        arch_tag = tail
    elif tail in KNOWN_GOOS.union({"unix"}):
        os_tag = tail
    elif tail in KNOWN_GOARCH:
        arch_tag = tail

    if os_tag is not None and os_tag not in tags:
        return False
    if arch_tag is not None and arch_tag != goarch:
        return False
    return True


def file_matches(path: str, goos: str, goarch: str, tags: Set[str]) -> bool:
    name = os.path.basename(path)
    if not filename_matches(name, goos, goarch, tags):
        return False

    go_build, plus_build = read_build_constraints(path)
    if go_build:
        return parse_go_build_expr(go_build).eval(tags)
    if plus_build:
        return plus_build_matches(plus_build, tags)
    return True


def package_source_dirs(pkg_dir: str) -> List[str]:
    dirs = [pkg_dir]
    dirs_file = os.path.join(pkg_dir, "package_dirs.txt")
    if os.path.exists(dirs_file):
        with open(dirs_file, "r", encoding="utf-8", errors="ignore") as f:
            for raw in f:
                value = raw.split("#", 1)[0].strip()
                if value:
                    dirs.append(os.path.join(pkg_dir, value))
    return dirs


def list_package_go_files(
    pkg_dir: str, goos: str, goarch: str, extra_tags: Optional[Iterable[str]] = None
) -> List[str]:
    tags = default_tag_set(goos, goarch, extra_tags)
    files: List[str] = []
    for source_dir in package_source_dirs(pkg_dir):
        if not os.path.isdir(source_dir):
            continue
        for name in sorted(os.listdir(source_dir)):
            path = os.path.join(source_dir, name)
            if file_matches(path, goos, goarch, tags):
                files.append(path)
    return files
