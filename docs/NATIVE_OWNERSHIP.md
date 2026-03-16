# Bootstrap-To-Native Ownership Map

This document maps the current bootstrap implementation files to their future
owners in a native `GOOS=kolibrios GOARCH=386` port.

The goal is to stop Phase 6 work from drifting between three different kinds
of code:

- temporary bootstrap glue that should disappear
- behavior that must survive but move into the Go tree
- repo-local SDK code that can stay outside the Go tree even after native
  support exists

Read this together with `docs/NATIVE_PORT.md`.

## Ownership Classes

| Class | Meaning |
| --- | --- |
| native runtime | should move into `runtime` or runtime-owned arch files in the Go tree |
| native syscall | should move into `syscall` or runtime/syscall arch glue in the Go tree |
| native linker/build | should move into `cmd/link`, `cmd/dist`, `cmd/go`, or target tables |
| repo-local SDK | can stay in this repository as KolibriOS-specific wrappers/examples even after native support exists |
| bootstrap-only | useful for the current `gccgo` path, not intended to survive as part of the final target architecture |

## File-By-File Map

### `platform/abi/runtime_gccgo.c`

Current responsibilities:

- compiler-emitted `gccgo` runtime symbol glue
- allocation helpers
- string, slice, map, and interface helpers for the validated bootstrap subset
- bootstrap GC
- panic/bounds failure helpers
- DLL function-call trampolines and `lib_init` bridge
- console bridge helpers
- C-string / raw-memory utility helpers

Future ownership:

- native runtime:
  - allocation
  - string helpers
  - slice helpers
  - map helpers
  - interface/assertion helpers
  - panic/failure paths
  - collector implementation
- native syscall or runtime-internal arch glue:
  - raw low-level helper calls that must still exist below Go code
- bootstrap-only:
  - `gccgo`-specific symbol names and glue conventions
  - bootstrap console bridge state
  - bootstrap DLL callback plumbing as currently shaped

First native slice:

- only a small subset from this file should move first:
  - startup-adjacent runtime hooks
  - allocation
  - string basics
  - exit/failure path
  - time/sleep support

Not a first-slice blocker:

- broad map/interface parity
- bootstrap GC implementation details
- DLL helper surface for second-wave `.obj` wrappers

### `platform/abi/syscalls_i386.asm`

Current responsibilities:

- raw `int 0x40` entrypoints
- heap helpers via function `68`
- DLL loader entrypoints
- file-system entrypoints
- window/input/time/system entrypoints
- raw exports consumed by `platform/kos/raw.go`

Future ownership:

- native syscall:
  - public raw syscall-facing surface that should back `syscall` or runtime
    syscalls
- native runtime:
  - arch-specific low-level entrypoints needed internally by runtime startup or
    memory/time/process code
- bootstrap-only:
  - exported symbol names tied to the current `gccgo` package ABI

First native slice:

- keep only the stubs needed for:
  - exit
  - time/date/uptime
  - sleep
  - file-backed stdio or basic file opens if required by the first bring-up

Later native slices:

- windowing/input
- broader graphics/UI helpers
- DLL loading and higher-value process/UI syscalls

### `platform/kos/raw.go`

Current responsibilities:

- exported Go declarations matching `platform/abi/syscalls_i386.asm`
- runtime-only helper declarations for DLL trampolines, console bridge, loader
  buffers, and GC polling

Future ownership:

- native syscall:
  - public declarations whose semantics should survive as target primitives
- native runtime:
  - hidden startup/runtime hooks such as loader buffer access
- bootstrap-only:
  - direct exposure of helper hooks that are implementation details today
    rather than stable target API

Split rules:

- should remain public in some form:
  - stable syscall-backed primitives for time, files, process exit, and basic
    system services
- should become runtime/internal only:
  - `LoaderParametersRaw`
  - `LoaderPathRaw`
  - console bridge helpers
  - GC polling helpers
  - bootstrap DLL init/trampoline helpers

### `platform/kos/loader.go`

Current responsibilities:

- turns raw loader buffers into Go strings
- normalizes loader path prefixes

Future ownership:

- native runtime plus `os` startup path:
  - argv bootstrap and loader-path normalization should become internal target
    plumbing, not a public SDK requirement
- repo-local SDK:
  - a thin helper may still remain if direct loader inspection is useful for
    KolibriOS apps, but that is not required for the native target bring-up

First native slice:

- yes, because `os.Args` is part of the first target profile

### `tooling/app-startup.c.in`

Current responsibilities:

- bootstrap entrypoint symbol
- `kolibri_app_init()` then `kolibri_app_main()` handoff
- raw process exit after `main.main`
- stack-top registration for the bootstrap GC
- loader parameter/path buffers exported into the image

Future ownership:

- native runtime:
  - real `rt0` / startup assembly and init path
  - runtime-owned argv bootstrap
  - runtime-owned exit path after `main.main`
- bootstrap-only:
  - C template startup wrapper used by the current `gccgo` build

First native slice:

- yes, this file is one of the main direct inputs to the first `hello world`
  bring-up

### `tooling/static.lds.in`

Current responsibilities:

- emits the `MENUET01` loader header
- sets entrypoint
- exposes loader parameter/path buffer addresses in the image header
- shapes RX/RW segments
- reserves stack-space-aware memory top

Future ownership:

- native linker/build:
  - `cmd/link` must own the final executable layout and header emission
- runtime coordination:
  - startup/runtime still depend on the loader buffer contract and stack
    expectations, but should not own the final image format logic
- bootstrap-only:
  - local linker script templating as the long-term build story

First native slice:

- yes, because without this there is no valid native KolibriOS image to boot

### `tooling/kolibri-app.mk`

Current responsibilities:

- current `gccgo` app build orchestration
- package compilation ordering
- startup-template substitution
- linker-script substitution
- final `ld` invocation and binary conversion

Future ownership:

- native linker/build:
  - the "real" build path should move into `cmd/dist`, `cmd/link`, and
    `cmd/go`
- bootstrap-only:
  - repository-local `gccgo` orchestration

Carry-forward note:

- this file can remain useful as a fallback/bootstrap path even after the
  native target exists, but it should stop being the mandatory path for normal
  app builds

### `platform/kos/*.go` and `stdlib/ui/*.go` wrappers

Current responsibilities:

- typed KolibriOS wrappers on top of the raw ABI
- higher-level `.obj` integrations
- app-facing helper APIs

Future ownership:

- repo-local SDK

Why:

- these wrappers are useful application-layer affordances, but they are not
  blockers for getting `GOOS=kolibrios` accepted as a real Go target
- native Go support should make these wrappers easier to build, not absorb them
  wholesale into the standard library

### Validation Harness (`apps/diag`, `apps/examples/*`)

Current responsibilities:

- regression coverage for wrappers and stdlib shims
- smoke validation for the bootstrap runtime surface

Future ownership:

- repo-local SDK and validation harness

Carry-forward note:

- this regression discipline should survive Phase 6
- the bring-up strategy should stay the same: explicit PASS / FAIL checkpoints

## First Native Go-Tree Patch Set

The first external Go-tree patch set should focus only on these ownership
transfers:

1. runtime startup from `tooling/app-startup.c.in`
2. linker image/header rules from `tooling/static.lds.in`
3. minimal syscall/runtime arch stubs from `platform/abi/syscalls_i386.asm`
4. minimal runtime service surface from `platform/abi/runtime_gccgo.c`
5. target discovery/build integration that replaces the mandatory role of
   `tooling/kolibri-app.mk`

That first patch set should explicitly exclude:

- `.obj` wrapper convenience layers
- console bridge extras
- DLL callback trampoline breadth beyond what the runtime itself strictly needs
- broad stdlib expansion
- scheduler/goroutine ambitions

## What Should Stay Repo-Local

Even after native support exists, these are still good fits for this
repository:

- `kos` typed wrappers
- `ui` helpers
- `.obj` wrapper libraries
- app examples and templates
- QEMU regression scripts
- diagnostics utilities such as `apps/diag`

Native support should reduce the amount of custom bootstrap glue, not erase the
value of the repository as a KolibriOS Go SDK.

## Immediate Next Technical Slice

The next concrete implementation target after this ownership map should be a
minimal hello-world native checklist:

1. startup and `main.main`
2. exit
3. `os.Args`
4. `os.Stdout` / `os.Stderr`
5. wall clock and sleep

That keeps the first Go-tree work small and aligned with the Phase 6A contract.
