#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

LIBGO_SRC=${1:-${LIBGO_SRC:-/usr/src/gcc-13/gcc-13.3.0/libgo}}
DEST=${2:-${DEST:-${ROOT}/native/libgo/staging}}

SYNC_DIRS=(
  go/runtime
  go/internal/abi
  go/internal/bytealg
  go/internal/cpu
  go/internal/goarch
  go/internal/goexperiment
  go/internal/goos
  runtime
)

if [[ ! -d "${LIBGO_SRC}" ]]; then
  echo "libgo source tree not found: ${LIBGO_SRC}" >&2
  echo "set LIBGO_SRC=/path/to/gcc/libgo or pass it as the first argument" >&2
  exit 1
fi

mkdir -p "${DEST}"

for rel in "${SYNC_DIRS[@]}"; do
  src="${LIBGO_SRC}/${rel}"
  dst="${DEST}/${rel}"

  if [[ ! -d "${src}" ]]; then
    echo "missing upstream directory: ${src}" >&2
    exit 1
  fi

  rm -rf "${dst}"
  mkdir -p "$(dirname "${dst}")"
  cp -a "${src}" "${dst}"
done

find "${DEST}" -type f \( -name '*_test.go' -o -name '*_test.c' -o -name '*_test.cc' \) -delete
find "${DEST}" -type d \( -name testdata -o -name .git \) -prune -exec rm -rf {} +

cp -a "${ROOT}/native/libgo/overlay/." "${DEST}/"

OS_GCCGO_FILE="${DEST}/go/runtime/os_gccgo.go"
if [[ -f "${OS_GCCGO_FILE}" ]]; then
  python3 - "${OS_GCCGO_FILE}" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = path.read_text(encoding="utf-8")
if "//go:build !kolibrios" not in data:
    path.write_text("//go:build !kolibrios\n\n" + data, encoding="utf-8")
PY
fi

RUNTIME_TIME_FILE="${DEST}/go/runtime/time.go"
if [[ -f "${RUNTIME_TIME_FILE}" ]]; then
  python3 - "${RUNTIME_TIME_FILE}" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
data = path.read_text(encoding="utf-8")
updated = data.replace("//go:linkname timeSleep time.Sleep", "//go:linkname timeSleep runtime.timeSleep")
if updated != data:
    path.write_text(updated, encoding="utf-8")
PY
fi

cat > "${DEST}/.source-stamp" <<EOF
source=${LIBGO_SRC}
synced_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF

echo "staged libgo runtime slice into ${DEST}"
