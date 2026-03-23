#!/usr/bin/env python3

import argparse
import os
import sys

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from go_file_filter import list_package_go_files


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--package-dir", required=True)
    parser.add_argument("--goos", default="kolibrios")
    parser.add_argument("--goarch", default="386")
    parser.add_argument("--tags", default="gccgo")
    args = parser.parse_args()

    files = list_package_go_files(
        args.package_dir,
        args.goos,
        args.goarch,
        [item for item in args.tags.split() if item],
    )
    for path in files:
        print(path)


if __name__ == "__main__":
    main()
