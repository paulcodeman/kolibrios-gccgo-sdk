#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 [--kpack] [--kpack-bin <path>]" >&2
  echo "examples:" >&2
  echo "  $0" >&2
  echo "  $0 --kpack" >&2
}

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=${SCRIPT_DIR}

KPACK=${KPACK:-0}
KPACK_BIN=${KPACK_BIN:-}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --kpack|--compress)
      KPACK=1
      shift
      ;;
    --kpack-bin)
      if [[ $# -lt 2 ]]; then
        echo "missing argument for --kpack-bin" >&2
        usage
        exit 2
      fi
      KPACK_BIN=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      echo "unknown option: $1" >&2
      usage
      exit 2
      ;;
    *)
      echo "unexpected argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

clean_artifacts() {
  rm -rf "${REPO_ROOT}/.pkg"
  find "${REPO_ROOT}/platform/abi" -maxdepth 1 -name '*.o' -delete
  find "${REPO_ROOT}" -type d -name '.build' -prune -exec rm -rf {} +
  find "${REPO_ROOT}" -type f \( -name '*.gccgo.o' -o -name '*.gox' \) -delete
}

clean_artifacts

find_roots=()
if [[ -d "${REPO_ROOT}/apps" ]]; then
  find_roots+=("${REPO_ROOT}/apps")
fi

mapfile -t targets < <(find "${find_roots[@]}" -type f -name Makefile -printf '%h\n' | sort -u)
echo "Found ${#targets[@]} targets"

for dir in "${targets[@]}"; do
  rel="${dir#${REPO_ROOT}/}"
  echo "==> Building ${rel}"
  build_args=()
  if [[ "${KPACK}" != "0" ]]; then
    build_args+=(--kpack)
  fi
  if [[ -n "${KPACK_BIN}" ]]; then
    build_args+=(--kpack-bin "${KPACK_BIN}")
  fi
  KEEP_PKG=1 KEEP_ABI=1 FAST_PKG=1 ./build-app.sh "${build_args[@]}" "${dir}"
done

clean_artifacts
