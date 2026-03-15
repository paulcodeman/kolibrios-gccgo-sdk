#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <name|path> [make-target]" >&2
  echo "examples:" >&2
  echo "  $0 uiwindow" >&2
  echo "  $0 examples/uiwindow" >&2
  echo "  $0 /abs/path/to/examples/uiwindow" >&2
  echo "  $0 uiwindow clean" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=${SCRIPT_DIR}

INPUT=$1
MAKE_TARGET=${2:-all}

target_dir=""

if [[ -d "${INPUT}" ]]; then
  target_dir=$(cd "${INPUT}" && pwd)
elif [[ "${INPUT}" == */* ]]; then
  if [[ -d "${REPO_ROOT}/${INPUT}" ]]; then
    target_dir=$(cd "${REPO_ROOT}/${INPUT}" && pwd)
  fi
else
  matches=()
  for base in examples apps; do
    if [[ -d "${REPO_ROOT}/${base}/${INPUT}" ]]; then
      matches+=("${REPO_ROOT}/${base}/${INPUT}")
      continue
    fi

    if [[ "${base}" == "apps" ]]; then
      for group in "${REPO_ROOT}/${base}"/*; do
        if [[ -d "${group}/${INPUT}" ]]; then
          matches+=("${group}/${INPUT}")
        fi
      done
    fi
  done

  if [[ ${#matches[@]} -eq 1 ]]; then
    target_dir=$(cd "${matches[0]}" && pwd)
  elif [[ ${#matches[@]} -gt 1 ]]; then
    echo "ambiguous target name: ${INPUT}" >&2
    for match in "${matches[@]}"; do
      rel="${match#${REPO_ROOT}/}"
      echo "  - ${rel}" >&2
    done
    exit 1
  fi
fi

if [[ -z "${target_dir}" ]]; then
  echo "target directory not found: ${INPUT}" >&2
  exit 1
fi

if [[ ! -f "${target_dir}/Makefile" ]]; then
  echo "target does not provide a Makefile: ${target_dir}" >&2
  exit 1
fi

make -C "${target_dir}" "${MAKE_TARGET}"

if [[ "${MAKE_TARGET}" == "clean" ]]; then
  exit 0
fi

target_base=$(basename "${target_dir}")
output_path="${target_dir}/${target_base}.kex"

if [[ ! -f "${output_path}" ]]; then
  echo "expected output not found: ${output_path}" >&2
  exit 1
fi

echo "${output_path}"
