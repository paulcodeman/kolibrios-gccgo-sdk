# Repository Instructions

## KolibriOS API Source Of Truth

- When adding or changing Go bindings for KolibriOS system calls, always consult `sysfuncs.txt` in the repository root first.
- Do not invent syscall numbers, register layouts, subfunction codes, packed arguments, or return conventions from memory.
- Treat `sysfuncs.txt` as the source of truth for:
  - the function number placed into `eax`
  - input/output register usage
  - packed argument formats such as `x * 65536 + width`
  - preserved registers and return-value behavior

## Where To Apply Changes

- Keep low-level syscall entrypoints aligned with `platform/abi/syscalls_i386.asm`.
- Keep exported Go declarations aligned with `platform/kos/raw.go`.
- Keep higher-level Go wrappers and types aligned with the low-level ABI in `platform/kos/*.go` and `stdlib/ui/*.go` when relevant.

## Implementation Rule

- If a new Go function wraps a KolibriOS API call, verify the exact calling convention in `sysfuncs.txt` before writing the Go signature or the assembly stub.

## Stdlib And Runtime Strategy

- If stdlib functionality is missing, first copy the relevant non-platform-specific files from the official sources (Go stdlib or libgo).
- If a file is platform-dependent, adapt only the KolibriOS-specific parts and keep the rest aligned with upstream.
- If compilation fails due to missing runtime symbols, extend `platform/abi/runtime_gccgo.c` (and related ABI code) to provide the needed symbols.
