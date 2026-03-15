# Native Bring-Up Checklist

This file defines the first executable success ladder for the future native
`GOOS=kolibrios GOARCH=386` port.

The purpose is simple: the first Go-tree fork should have a narrow list of
PASS / FAIL targets instead of attempting a broad port in one jump.

Reference inputs:

- `docs/NATIVE_PORT.md`
- `docs/NATIVE_OWNERSHIP.md`
- `docs/NATIVE_GOTREE_PLAN.md`

## Scope

This checklist covers only the first native bring-up slice:

- hello world
- process exit
- stdout / stderr
- argv
- wall clock
- sleep

It does **not** cover:

- maps
- interfaces
- files beyond what stdout/stderr needs
- goroutines (bootstrap runtime now supports them, but they are out of scope here)
- channels (bootstrap runtime now supports them, but they are out of scope here)
- broad stdlib parity

## Bring-Up Rungs

### 1. Image Boots And Reaches `main.main`

Goal:

- prove that the new linker head type, startup path, and runtime entry glue are
  coherent enough to start user code

PASS when:

- a native-built binary boots under KolibriOS
- `main.main` executes
- the program can reach a deliberate "I started" marker

Evidence source in the bootstrap repo:

- `tooling/app-startup.c.in`
- `tooling/static.lds.in`
- `examples/uiwindow`

Likely Go-tree files:

- `cmd/internal/objplatform/abi/head.go`
- `cmd/link/internal/x86/obj.go`
- `cmd/link/internal/ld/...`
- `runtime/rt0_kolibrios_386.s`

### 2. Program Returns And Exits Cleanly

Goal:

- prove that startup is not one-way only and that the process exit path is
  owned by the native runtime/syscall layer

PASS when:

- `main.main` returns
- the process terminates without hanging
- emulator automation can observe normal process end

Evidence source in the bootstrap repo:

- `tooling/app-startup.c.in`
- `platform/abi/syscalls_i386.asm`
- `sysfuncs.txt` for function `-1`

Likely Go-tree files:

- `runtime/rt0_kolibrios_386.s`
- `runtime/sys_kolibrios_386.s`
- `syscall/syscall_kolibrios.go`

### 3. `os.Args` Works

Goal:

- preserve the loader-backed argv contract already proven in the bootstrap path

PASS when:

- `len(os.Args) >= 1`
- `os.Args[0]` reflects the loader path
- additional loader parameters arrive as separate args

Evidence source in the bootstrap repo:

- `platform/kos/loader.go`
- `stdlib/os/os.go`

Likely Go-tree files:

- `runtime/os_kolibrios.go`
- startup/runtime argv bridge
- `os`

### 4. `stdout` / `stderr` Works

Goal:

- support ordinary entry-level Go programs before any GUI-specific work

PASS when:

- a native binary can write to stdout
- stderr can be routed distinctly or at least through the same first-stage sink
- emulator harness can capture output deterministically

First-stage note:

- file-backed or debug-console-backed output is enough
- console-object richness is not required for the first native rung

Evidence source in the bootstrap repo:

- `stdlib/fmt`
- `stdlib/log`
- `apps/diag`
- `examples/console`

Likely Go-tree files:

- `runtime`
- `syscall`
- `os`

### 5. Wall Clock Works

Goal:

- prove that the native runtime can provide the same narrow time contract as
  the bootstrap path

PASS when:

- `time.Now()` returns a coherent wall clock
- repeated reads are monotonic enough for ordinary log/output use

Evidence source in the bootstrap repo:

- `platform/kos/time.go`
- `stdlib/time`
- `sysfuncs.txt` for functions `3` and `29`

Likely Go-tree files:

- `runtime/os_kolibrios.go`
- `runtime/time`-adjacent target hooks
- `syscall`

### 6. Sleep Works

Goal:

- prove that timed delay and basic scheduler-independent waiting work

PASS when:

- `time.Sleep(...)` blocks approximately as expected
- a small wall-clock delta is visible before/after sleep

Evidence source in the bootstrap repo:

- `kos.Sleep`
- `platform/kos/time.go`
- `stdlib/time`
- `apps/diag`

Likely Go-tree files:

- `runtime/os_kolibrios.go`
- `runtime/sys_kolibrios_386.s`
- `syscall`

## First Native Test Programs

The first Go-tree fork should keep the test binaries intentionally tiny.

Program A:

- hello world
- writes one startup marker
- returns

Program B:

- prints `os.Args`
- returns

Program C:

- prints `time.Now()`
- sleeps briefly
- prints `time.Now()` again

If any of these fail, do not move on to broader stdlib work.

## Pass Order

Bring-up should be considered complete for the first native slice only if the
rungs pass in this order:

1. boot into `main.main`
2. clean exit
3. stdout / stderr
4. argv
5. wall clock
6. sleep

This order matters:

- `argv` depends on startup
- `stdout` depends on runtime/syscall basics
- `time` and `sleep` depend on working syscall plumbing

## Explicit Non-Goals For The First Fork

Do not expand scope to these before the checklist above is green:

- maps
- interfaces
- files and `os.Stat`
- `fmt` parity
- `net`
- `http`
- `.obj` wrappers
- GUI demos

Those are later native bring-up steps, not first-fork blockers.

## Next Checklist After This One

Once this file is fully green, the next checklist should extend to:

1. allocation and strings
2. files and minimal `os`
3. maps
4. interfaces
5. selected stdlib packages already proven by the bootstrap path
