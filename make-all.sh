#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 [--kpack] [--kpack-bin <path>] [--jobs <count>] [--rebuild]" >&2
  echo "Build every app folder, including tests; preserve SDK/compiler/source caches." >&2
}

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=${SCRIPT_DIR}
KPACK=${KPACK:-0}
KPACK_BIN=${KPACK_BIN:-}
JOBS=${JOBS:-4}
REBUILD=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --kpack|--compress) KPACK=1; shift ;;
    --kpack-bin|--jobs)
      option=$1
      [[ $# -ge 2 ]] || { usage; exit 2; }
      if [[ "$option" == --jobs ]]; then JOBS=$2; else KPACK_BIN=$2; fi
      shift 2
      ;;
    --rebuild) REBUILD=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done
[[ "$JOBS" =~ ^[1-9][0-9]*$ ]] || { echo "--jobs must be positive" >&2; exit 2; }

cd "${REPO_ROOT}"
report_dir="${REPO_ROOT}/.build-cache/build-all/$(date -u +%Y%m%dT%H%M%SZ)-$$"
mkdir -p "$report_dir"
report="$report_dir/results.tsv"
printf 'target\tresult\tlog\n' > "$report"
mapfile -d '' -t targets < <(find "${REPO_ROOT}/apps" \
  -type d \( -name vendor -o -name .git -o -name .build \) -prune -o \
  -type f -name Makefile -printf '%h\0' | sort -zu)
echo "Found ${#targets[@]} targets; logs: $report_dir"
passed=0
failed=0
build_args=()
[[ "$KPACK" == 0 ]] || build_args+=(--kpack)
[[ -z "$KPACK_BIN" ]] || build_args+=(--kpack-bin "$KPACK_BIN")
export MAKEFLAGS="${MAKEFLAGS:-} -j${JOBS}"
export PARALLEL_PACKAGES=1
export GOMAXPROCS=${GOMAXPROCS:-2}

for dir in "${targets[@]}"; do
  rel="${dir#${REPO_ROOT}/}"
  log="$report_dir/${rel//\//_}.log"
  echo "==> Building ${rel}"
  clean_ok=1
  if [[ "$REBUILD" != 0 ]]; then
    # Only the selected app's generated output is cleaned. Compiler, upstream
    # sources, vendor trees, runtime and shared package caches stay intact.
    ./build-app.sh "$dir" clean > "$log" 2>&1 || clean_ok=0
  else
    : > "$log"
  fi
  if [[ "$clean_ok" == 1 ]] && FAST_PKG=1 ./build-app.sh "${build_args[@]}" "$dir" >> "$log" 2>&1; then
    passed=$((passed + 1))
    printf '%s\tPASS\t%s\n' "$rel" "$log" >> "$report"
    echo "PASS $rel"
  else
    failed=$((failed + 1))
    printf '%s\tFAIL\t%s\n' "$rel" "$log" >> "$report"
    echo "FAIL $rel"
    tail -n 15 "$log"
  fi
done
python3 "${REPO_ROOT}/tooling/build-all-report.py" "$report"
echo "Built $passed / ${#targets[@]} targets; failures: $failed"
echo "Report: $report"
[[ "$failed" == 0 ]]
