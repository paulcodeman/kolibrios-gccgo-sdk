#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "$0")" && pwd)
template_dir="$repo_root/templates/basic-lib"

usage() {
  echo "usage: bash ./new-lib.sh <name>" >&2
}

escape_sed_replacement() {
  printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

name=$1

if [[ ! "$name" =~ ^[a-z_][a-z0-9_]*$ ]]; then
  echo "library name must match ^[a-z_][a-z0-9_]*$ so it is valid as a build target and directory" >&2
  exit 1
fi

target_dir="$repo_root/libs/$name"
if [[ -e "$target_dir" ]]; then
  echo "target already exists: $target_dir" >&2
  exit 1
fi

mkdir -p "$target_dir"

name_replacement=$(escape_sed_replacement "$name")

render_template() {
  local source_name=$1
  local output_name=$2

  sed \
    -e "s|__PROGRAM__|$name_replacement|g" \
    -e "s|__PACKAGE_NAME__|$name_replacement|g" \
    "$template_dir/$source_name" >"$target_dir/$output_name"
}

render_template "Makefile.in" "Makefile"
render_template "lib.go.in" "lib.go"
render_template "exports.txt.in" "exports.txt"

printf 'created %s\n' "$target_dir"
printf 'build with: ./build-lib.sh libs/%s\n' "$name"
