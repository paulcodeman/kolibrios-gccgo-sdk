# Repository Instructions

## Syscall ABI (Source Of Truth)

- Always consult `sysfuncs.txt` before adding or changing any KolibriOS syscall binding.
- Do not invent syscall numbers, register layouts, packed args, or return rules.
- Verify calling conventions in `sysfuncs.txt` before writing Go signatures or assembly stubs.
- Keep low-level entrypoints aligned across `platform/abi/syscalls_i386.asm` and `platform/kos/raw.go`.

## ABI Surface Alignment

- Keep higher-level wrappers/types in `platform/kos/*.go` and `stdlib/ui/*.go` aligned with the raw ABI.

## Stdlib And Runtime Porting

- When stdlib/runtime functionality is missing, start from upstream sources (Go stdlib, libgo, or runtime) and port them.
- Only change KolibriOS-specific parts; keep the rest aligned with upstream.
- Do not reimplement from scratch unless there is no upstream source to start from.
- For missing runtime symbols in the bootstrap build, extend `platform/abi/runtime_gccgo.c` (and related ABI code).

## App Layout Convention

- For small apps/examples, keep all logic in a single `main.go`.
- Avoid splitting trivial entrypoints into `main.go` + `app.go` unless there is a clear complexity or reuse need.
