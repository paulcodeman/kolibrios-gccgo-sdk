#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=${SCRIPT_DIR}

clean_artifacts() {
  rm -rf "${REPO_ROOT}/.pkg"
  find "${REPO_ROOT}/platform/abi" -maxdepth 1 -name '*.o' -delete
  find "${REPO_ROOT}" -type d -name '.build' -prune -exec rm -rf {} +
  find "${REPO_ROOT}" -type f \( -name '*.gccgo.o' -o -name '*.gox' \) -delete
}

clean_artifacts

mapfile -t targets < <(find "${REPO_ROOT}/examples" "${REPO_ROOT}/apps" -type f -name Makefile -printf '%h\n' | sort -u)
echo "Found ${#targets[@]} targets"

for dir in "${targets[@]}"; do
  rel="${dir#${REPO_ROOT}/}"
  echo "==> Building ${rel}"
  KEEP_PKG=1 KEEP_ABI=1 FAST_PKG=1 ./build-app.sh "${dir}"
done

clean_artifacts
