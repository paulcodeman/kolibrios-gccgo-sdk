#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <runtime.inc.raw> <runtime.inc>" >&2
  exit 1
fi

raw_input=$1
output=$2

tmpdir=$(mktemp -d)
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

cp "${raw_input}" "${tmpdir}/runtime.inc.raw"
cd "${tmpdir}"

grep -v "#define _" runtime.inc.raw | \
  grep -v "#define [cm][012345] " | \
  grep -v "#define empty " | \
  grep -v "#define \\$" | \
  grep -v "#define mSpanInUse " > runtime.inc.tmp2

for pattern in '_[GP][a-z]' _Max _Lock _Sig _Trace _MHeap _Num; do
  grep "#define ${pattern}" runtime.inc.raw >> runtime.inc.tmp2 || true
done

for type_name in _Complex_lock _Reader_lock semt boundsError _FILE; do
  sed -e "/struct ${type_name} {/,/^}/s/^.*$//" runtime.inc.tmp2 > runtime.inc.tmp3
  mv runtime.inc.tmp3 runtime.inc.tmp2
done

sed -e 's/sigset/sigset_go/' runtime.inc.tmp2 > runtime.inc.tmp3
mv runtime.inc.tmp3 runtime.inc.tmp2

sed -e '/struct .*type {/,/^}/ s/\t\(.*;\)/\tconst \1/' \
  < runtime.inc.tmp2 > tmp-runtime.inc

mkdir -p "$(dirname "${output}")"
cp tmp-runtime.inc "${output}"
