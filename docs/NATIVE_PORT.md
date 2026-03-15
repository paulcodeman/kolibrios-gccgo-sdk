# Native Port Contract

This document turns the current bootstrap evidence into the target contract for
the future native Go port.

It does not claim that `GOOS=kolibrios` support already exists in the main Go
toolchain. It defines what that support is expected to mean when the port is
brought up.

Read this together with `docs/NATIVE_OWNERSHIP.md`.
For the first native execution gate, use `docs/NATIVE_BRINGUP_CHECKLIST.md`.

## Target Tuple

- `GOOS=kolibrios`
- `GOARCH=386`

Initial native-port profile:

- single-threaded first
- `CGO_ENABLED=0` first
- no goroutine or channel requirement in the first bring-up slice
- the first success bar is "real `go build` for a narrow supported package
  set", not full upstream parity

Note: the bootstrap runtime now supports goroutines/channels and a
multi-threaded scheduler, but that does not change the initial native-port
bring-up bar.

## Contract Summary

| Area | Native target contract | Current bootstrap evidence |
| --- | --- | --- |
| executable format | emit KolibriOS `.kex` binaries with the loader-visible `MENUET01` image contract and separate RX/RW load segments | `tooling/static.lds.in`, `tooling/app-startup.c.in`, `docs/BUILD.md` |
| startup | loader hands the process path and parameter buffers to the startup code before user `main.main`; runtime startup must own init ordering | `tooling/app-startup.c.in`, `platform/kos/loader.go`, `stdlib/os/os.go` |
| paths | slash-first paths, `/` separator, `:` list separator, no volume names, no Windows drive semantics | `stdlib/path`, `stdlib/path/filepath` |
| argv | `os.Args[0]` comes from the loader path, remaining args come from the loader parameter string | `platform/kos/loader.go`, `stdlib/os/os.go` |
| environment | process-local environment is acceptable for the first native slice; stronger global semantics can wait until there is real demand | `stdlib/os/os.go`, `examples/os`, `apps/diag` |
| files | ordinary Go file flows should preserve the current slash-first KolibriOS behavior for stat/open/read/write/seek/create/mkdir/rename/remove | `stdlib/os`, `examples/files`, `examples/os` |
| time | wall clock comes from the current date/time syscalls, monotonic timing from the uptime counter, and sleep remains explicit and syscall-backed | `platform/kos/time.go`, `stdlib/time` |
| stdio | ordinary stdout/stderr/stdin flows must work for pipe/file-backed cases first; console-backed integration remains a later integration layer on top | `stdlib/fmt`, `stdlib/log`, `examples/console`, `apps/diag` |
| process model | process exit is explicit; pid/ppid and child-start behavior can start with the current narrow contract before a broader process API exists | `stdlib/os/os.go`, `platform/kos/process.go` |
| scheduler | native port can start single-threaded; goroutines/channels and multi-threaded scheduling are supported in the bootstrap runtime, but are not a bring-up prerequisite | `platform/abi/runtime_gccgo.c`, `examples/goroutines`, `examples/threads` |

## Carry-Over Rules From The Bootstrap Path

The native port should intentionally preserve these already-proven contracts in
its first slice:

- slash-first path behavior
- loader-backed `os.Args`
- ordinary `fmt`, `log`, `os`, `path`, and `path/filepath` behavior where the
  bootstrap implementation already matches the intended KolibriOS semantics
- explicit wall-clock plus monotonic-time split
- headless emulator-backed PASS / FAIL regressions instead of ad-hoc manual
  testing

The native port should not blindly preserve these bootstrap implementation
choices:

- `gccgo`-specific runtime symbol glue in `platform/abi/runtime_gccgo.c`
- the custom bootstrap linker glue as the final linker story
- the bootstrap runtime envelope as the final language envelope
- process-local env behavior as an irreversible design choice

## First Native Bring-Up Envelope

The first native target should prove these capabilities in order:

1. process startup and `main.main`
2. `os.Args`
3. `os.Stdout` / `os.Stderr`
4. process exit
5. allocation and strings
6. `time.Now` and `time.Sleep`
7. files and `os`
8. maps and interfaces
9. selected stdlib packages already proven by the bootstrap path

This keeps the bring-up bar aligned with the current validated bootstrap
surface instead of inventing a new target profile from scratch.

## Not In Scope For The First Native Slice

- goroutines
- channels
- full `defer`
- general `panic` / `recover` parity
- `cgo`
- reflection-heavy stdlib expansion
- claiming compatibility with arbitrary external Go projects

## Ownership Map For The Future Port

The current bootstrap repo already tells us where responsibilities live today.
Phase 6 should convert that into native Go ownership:

- bootstrap startup and loader bridge
  - current: `tooling/app-startup.c.in`, `platform/kos/loader.go`
  - future owner: `runtime` startup code
- bootstrap memory/runtime helpers
  - current: `platform/abi/runtime_gccgo.c`
  - future owner: `runtime`
- raw syscall ABI
  - current: `platform/abi/syscalls_i386.asm`, `platform/kos/raw.go`
  - future owner: `runtime` / `syscall`
- linker image shape
  - current: `tooling/static.lds.in`
  - future owner: `cmd/link`
- target discovery and build integration
  - current: repository-local makefiles and scripts
  - future owner: `cmd/dist`, `cmd/go`, target lists in the Go tree

## Immediate Next Steps

1. Use `docs/NATIVE_OWNERSHIP.md` as the file-by-file bootstrap-to-native
   responsibility map.
2. Define the minimal hello-world native port checklist:
   - startup
   - stdout/stderr
   - argv
   - exit
   - time
3. Use that checklist to drive the first external Go-tree patch set instead of
   starting from broad stdlib ambitions.
