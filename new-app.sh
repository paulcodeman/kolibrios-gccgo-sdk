#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "$0")" && pwd)
template_dir="$repo_root/templates/basic-app"

usage() {
  echo "usage: bash ./new-app.sh <name> [window-title]" >&2
}

escape_sed_replacement() {
  printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 1
fi

name=$1
window_title=${2:-"KolibriOS $name"}

if [[ ! "$name" =~ ^[a-z_][a-z0-9_]*$ ]]; then
  echo "app name must match ^[a-z_][a-z0-9_]*$ so it is valid as a build target and example directory" >&2
  exit 1
fi

target_dir="$repo_root/examples/$name"
if [[ -e "$target_dir" ]]; then
  echo "target already exists: $target_dir" >&2
  exit 1
fi

mkdir -p "$target_dir"

name_replacement=$(escape_sed_replacement "$name")
title_replacement=$(escape_sed_replacement "$window_title")

render_template() {
  local source_name=$1
  local output_name=$2

  sed \
    -e "s|__PROGRAM__|$name_replacement|g" \
    -e "s|__PACKAGE_NAME__|$name_replacement|g" \
    -e "s|__WINDOW_TITLE__|$title_replacement|g" \
    "$template_dir/$source_name" >"$target_dir/$output_name"
}

render_template "Makefile.in" "Makefile"
render_template "main.go.in" "main.go"
render_template "app.go.in" "app.go"

printf 'created %s\n' "$target_dir"
printf 'build with: make -C examples/%s all\n' "$name"
