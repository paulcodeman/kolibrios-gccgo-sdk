#!/usr/bin/env python3

import argparse
import os
import shutil
import subprocess
import sys
import tempfile


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Shorten ELF symbol names for COFF conversion.")
    parser.add_argument("--input", required=True, help="Input ELF object path.")
    parser.add_argument("--output", required=True, help="Output ELF object path.")
    parser.add_argument("--objcopy", default="objcopy", help="objcopy binary to use.")
    parser.add_argument("--nm", default="nm", help="nm binary to use.")
    parser.add_argument(
        "--keep",
        action="append",
        default=["EXPORTS"],
        help="Symbol name to keep (repeatable).",
    )
    parser.add_argument("--prefix", default="s", help="Prefix for generated symbols.")
    parser.add_argument(
        "--width",
        type=int,
        default=7,
        help="Numeric width for generated symbols (prefix + width must be <= 8).",
    )
    return parser.parse_args()


def collect_symbols(nm_path: str, input_path: str):
    nm_out = subprocess.check_output(
        [nm_path, "-a", input_path], text=True, errors="ignore"
    )
    seen = set()
    symbols = []
    for raw in nm_out.splitlines():
        line = raw.strip()
        if not line:
            continue
        parts = line.split()
        name = ""
        if len(parts) == 2:
            name = parts[1]
        elif len(parts) >= 3:
            name = parts[2]
        if not name:
            continue
        if name in seen:
            continue
        seen.add(name)
        symbols.append(name)
    return symbols


def main() -> int:
    args = parse_args()
    prefix = args.prefix
    width = args.width
    if len(prefix) + width > 8:
        print("error: prefix + width must be <= 8 for COFF symbols", file=sys.stderr)
        return 2

    keep = set(args.keep or [])
    symbols = collect_symbols(args.nm, args.input)

    mapping = []
    index = 0
    for name in symbols:
        if name in keep:
            continue
        if name.startswith(".") or name.startswith("$"):
            continue
        short = f"{prefix}{index:0{width}d}"
        index += 1
        mapping.append((name, short))

    if not mapping:
        if args.input != args.output:
            shutil.copyfile(args.input, args.output)
        return 0

    with tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8") as handle:
        map_path = handle.name
        for old, new in mapping:
            handle.write(f"{old} {new}\n")

    try:
        subprocess.check_call(
            [args.objcopy, "--redefine-syms", map_path, args.input, args.output]
        )
    finally:
        try:
            os.unlink(map_path)
        except OSError:
            pass

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
