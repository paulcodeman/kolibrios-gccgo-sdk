# Native Go-Tree Patch Plan

This document turns Phase 6 from a general roadmap into a concrete patch plan
for a Go-tree fork.

Reference tree used for this plan:

- Go `1.22.2`
- local source tree observed at `/usr/lib/go-1.22/src`

This file does not implement the native port by itself. It defines the first
real patch sets and the exact file classes they should touch.

Read this together with:

- `docs/NATIVE_PORT.md`
- `docs/NATIVE_OWNERSHIP.md`
- `docs/NATIVE_BRINGUP_CHECKLIST.md`

## Core Decision

KolibriOS should be treated as its own OS target, not squeezed into an
existing head type.

Implications:

- add `GOOS=kolibrios`
- keep `GOARCH=386`
- add a dedicated linker head type instead of pretending KolibriOS is:
  - `Hplan9`
  - `Hlinux`
  - `Hwindows`

Why:

- the executable image contract is `MENUET01`, not ELF, PE, or Plan 9 a.out
- the loader exposes process path and parameter buffers directly in the image
  header
- the entry/startup contract is specific enough that overloading another head
  type would hide real differences and make the port harder to maintain

## Patch Set 0 - Target Registration

Goal: make the Go tree recognize `kolibrios/386` as a real target tuple before
trying to boot programs.

Primary files in Go 1.22:

- `go/build/syslist.go`
- `internal/goos/goos.go`
- generated `internal/goos/zgoos_kolibrios.go`
- `cmd/dist/build.go`
- `cmd/go/internal/imports/build.go`
- `internal/platform/supported.go`

Expected edits:

- add `"kolibrios"` to `knownOS` in `go/build/syslist.go`
- decide whether `kolibrios` belongs in `unixOS`
  - current recommendation: do **not** add it to `unixOS` in the first slice
- regenerate/add `internal/goos/zgoos_kolibrios.go`
- add `kolibrios` to `cmd/dist/build.go` target recognition (`okgoos`)
- update the `cmd/go/internal/imports/build.go` copy logic so file selection
  understands `_kolibrios.go`
- add a `distInfo` entry in `internal/platform/supported.go` for
  `OSArch{"kolibrios","386"}`

First-slice policy values:

- `CgoSupported`: false
- `FirstClass`: false
- `Broken`: false
- race / msan / asan / fuzz: false
- internal linking only in the first slice
- only `buildmode=exe` in the first slice

Acceptance bar:

- `GOOS=kolibrios GOARCH=386` is accepted as a target tuple by the build
  system
- filename matching for `_kolibrios.go` works
- the target is recognized as narrow but valid, not as a broken fake target

## Patch Set 1 - Linker Head Type And Image Shape

Goal: let the linker emit a real KolibriOS executable image.

Primary files in Go 1.22:

- `cmd/internal/objplatform/abi/head.go`
- `cmd/internal/obj/x86/obj6.go`
- `cmd/link/internal/x86/obj.go`
- `cmd/link/internal/ld/asmb.go`
- `cmd/link/internal/ld/data.go`
- `cmd/link/internal/ld/target.go`
- likely one new linker source such as:
  - `cmd/link/internal/ld/kolibri.go`

Expected edits:

- add `Hkolibrios` in `cmd/internal/objplatform/abi/head.go`
- teach head-type parsing and formatting about `kolibrios`
- add x86 assembler/linker handling for the new head type
- define how `HEADR`, text start, and segment layout are chosen for KolibriOS
- emit the `MENUET01` header instead of ELF/PE/Plan9 output
- preserve the current bootstrap contract:
  - separate RX/RW load regions
  - loader parameter/path pointers in the image header
  - explicit entry symbol

Design note:

- do not try to shoehorn KolibriOS into the ELF path in `ld/elf.go`
- the current bootstrap linker script already proves that the format is its
  own thing

Acceptance bar:

- the linker can emit a `.kex`-class image for `kolibrios/386`
- the result matches the bootstrap image invariants closely enough to boot a
  hello-world-class program

## Patch Set 2 - Runtime Startup Slice

Goal: replace `tooling/app-startup.c.in` with native Go runtime startup.

Primary files in Go 1.22:

- `runtime/rt0_kolibrios_386.s` (new)
- `runtime/os_kolibrios.go` (new)
- `runtime/sys_kolibrios_386.s` (new)
- possibly:
  - `runtime/mem_kolibrios.go`
  - `runtime/stubs_kolibrios.go`
  - `runtime/defs_kolibrios_386.go`

Bootstrap behavior to carry over:

- startup reaches `main.main`
- process exits cleanly after `main.main` returns
- runtime sees loader path/parameter buffers
- `os.Args` can be built from those buffers
- wall clock and sleep stay syscall-backed

First-slice runtime scope:

- no goroutines requirement (bootstrap runtime already supports them, but they are not required for the first native bring-up slice)
- no channel requirement
- no broad signal integration requirement
- no `cgo`

Acceptance bar:

- a native hello world reaches `main.main`
- `os.Args` is visible
- the process exits normally

## Patch Set 3 - Minimal Syscall Surface

Goal: provide only the syscall-backed primitives needed by the first native
bring-up ladder.

Primary files in Go 1.22:

- `syscall/syscall_kolibrios.go` (new)
- `syscall/asm_kolibrios_386.s` (new)
- likely narrow generated/handwritten support files such as:
  - `syscall/zerrors_kolibrios_386.go`
  - `syscall/zsysnum_kolibrios_386.go`

First-slice syscall needs:

- exit
- sleep
- date/time
- uptime/high-precision time if required by runtime
- minimal file/stdout support
- enough process startup support for argv

Source of truth:

- `sysfuncs.txt`

Acceptance bar:

- runtime startup does not depend on bootstrap-only syscall glue
- first native hello world plus stdout/stderr and time paths can link

## Patch Set 4 - Minimal `os` / stdio Bring-Up

Goal: move beyond "program starts" to "ordinary Go entry-level programs run".

Primary areas:

- `runtime`
- `syscall`
- `os`

First-slice behaviors:

- `os.Args`
- `os.Stdout`
- `os.Stderr`
- `time.Now`
- `time.Sleep`

Carry-over rule:

- file-backed or pipe-backed stdio is enough first
- console bridge behavior from the bootstrap repo is not a blocker for the
  first native patch set

Acceptance bar:

- a native binary can print to stdout/stderr
- a native binary can report argv
- a native binary can sleep and read wall clock

## Patch Set 5 - Dist And `go build` Integration

Goal: make the port visible to the ordinary Go build flow.

Primary files in Go 1.22:

- `cmd/dist/build.go`
- `cmd/dist/supported_test.go`
- `cmd/link/internal/...` touched earlier
- target lists and generated `internal/goos` data
- `internal/platform/supported.go`

Expected results:

- `cmd/dist` can build the target toolchain
- `cmd/go` accepts the target tuple
- the linker path resolves to the new KolibriOS head type

First-slice policy:

- `go build` for a narrow supported package set
- not full stdlib parity
- not full host test parity

Acceptance bar:

- `GOOS=kolibrios GOARCH=386 go build` works for hello world and the first
  narrow runtime/os/time slice

## First Bring-Up Ladder

The first native programs should be brought up in this order:

1. hello world with clean exit
2. stdout / stderr
3. argv dump
4. wall clock
5. sleep
6. small allocation/string program
7. minimal `os` file open/stat flow

Do not start with:

- maps
- interfaces
- net/http
- `.obj` wrappers
- GUI-heavy samples

Those come after the native startup/linker/syscall story is proven.

## Explicit Go 1.22 Touch List

This is the smallest honest list of Go-tree areas that the first native port
will need to touch.

Target registration:

- `go/build/syslist.go`
- `internal/goos/goos.go`
- generated `internal/goos/zgoos_kolibrios.go`
- `cmd/go/internal/imports/build.go`
- `cmd/dist/build.go`
- `internal/platform/supported.go`

Target identity and head type:

- `cmd/internal/objplatform/abi/head.go`

386 assembler/linker OS handling:

- `cmd/internal/obj/x86/obj6.go`
- `cmd/link/internal/x86/obj.go`

Linker core:

- `cmd/link/internal/ld/asmb.go`
- `cmd/link/internal/ld/data.go`
- `cmd/link/internal/ld/target.go`
- one new KolibriOS-specific linker file

Runtime:

- `runtime/rt0_kolibrios_386.s`
- `runtime/os_kolibrios.go`
- `runtime/sys_kolibrios_386.s`
- likely one or more small KolibriOS-specific runtime support files

Syscall:

- `syscall/syscall_kolibrios.go`
- `syscall/asm_kolibrios_386.s`
- narrow z* support files as needed

## What This Repository Can Do Before The Go-Tree Fork Exists

Useful preparatory work that still belongs here:

- keep refining the target contract and ownership map
- keep `sysfuncs.txt`-driven syscall evidence exact
- preserve emulator-backed PASS / FAIL discipline
- keep the bootstrap runtime/examples as the executable specification for the
  later native port

What should wait for the Go-tree fork:

- `Hkolibrios` implementation
- native runtime startup files
- native linker output support
- real `GOOS=kolibrios GOARCH=386 go build`

## Immediate Next Step After This Plan

The next concrete Phase 6 implementation task should now be to use
`docs/NATIVE_BRINGUP_CHECKLIST.md` as the execution gate for the first Go-tree
fork instead of attempting a broad port in one jump.

## Current Native Port Tooling

The actual Go-tree patch work is intentionally tracked outside this bootstrap
repository. This document stays focused on the contract and sequencing, not the
mechanics of any particular fork script.
