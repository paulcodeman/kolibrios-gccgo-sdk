# libgo Runtime Staging

This directory stages an upstream `libgo` runtime slice and overlays the first
KolibriOS-specific files on top of it.

Why this exists:

- the current SDK still uses the bootstrap runtime in
  [platform/abi/runtime_gccgo.c](/mnt/c/Users/Paul/Desktop/kolibrios-gccgo-sdk/platform/abi/runtime_gccgo.c)
- `gccgo` already ships an upstream `libgo` runtime in the local GCC source
  tree
- porting against that runtime is a better starting point than growing the
  bootstrap runtime further

## Local Sources

The default expected upstream source tree is:

- `/usr/src/gcc-13/gcc-13.3.0/libgo`

That path exists on the current development host, but the sync script accepts a
custom path too.

## Sync

Run:

```sh
./tooling/sync-libgo-runtime.sh
```

Or with an explicit source tree and destination:

```sh
LIBGO_SRC=/path/to/libgo ./tooling/sync-libgo-runtime.sh /path/to/libgo /tmp/libgo-kolibri
```

The script stages:

- `go/runtime`
- `go/internal/abi`
- `go/internal/bytealg`
- `go/internal/cpu`
- `go/internal/goarch`
- `go/internal/goexperiment`
- `go/internal/goos`
- `runtime` (the C/asm side of `libgo`)

It then overlays the KolibriOS files from `native/libgo/overlay/`.
It also post-processes staged `go/runtime/os_gccgo.go` so the generic gccgo OS
layer is excluded when `GOOS=kolibrios`.

## Current Port Slice

The overlay added here is intentionally narrow:

- runtime OS layer: `os_kolibrios.go`
- lock/note path: `lock_kolibrios.go`
- no-op netpoll fallback: `netpoll_kolibrios.go`
- C timing/yield hooks:
  - `runtime/yield.c`
  - `runtime/go-now.c`
  - `runtime/go-nanotime.c`

This is enough to stop working from an empty directory and start porting
against real upstream runtime code.

## Known Blockers

This staging area is not wired into the SDK build yet. The next hard blockers
are:

- startup/argv:
  - upstream `runtime/go-main.c` assumes a normal hosted `main(argc, argv)`
  - KolibriOS needs loader-backed startup like the current
    [tooling/app-startup.c.in](/mnt/c/Users/Paul/Desktop/kolibrios-gccgo-sdk/tooling/app-startup.c.in)
- thread creation/context:
  - upstream `runtime/proc.c` uses `pthread` and `ucontext`
  - KolibriOS needs a path based on `CreateThreadRaw` and the existing
    context/trampoline work in
    [platform/abi/runtime_context_386.S](/mnt/c/Users/Paul/Desktop/kolibrios-gccgo-sdk/platform/abi/runtime_context_386.S)
- syscall/libc assumptions:
  - upstream `go-nosys.c` / `go-mmap.c` / stdio helpers assume POSIX libc
  - KolibriOS needs direct syscall-backed replacements

## Direction

The intended sequence is:

1. stage upstream `libgo`
2. replace the OS layer with KolibriOS files
3. replace startup and thread bring-up
4. replace libc-backed time/stdio/memory helpers
5. only then consider switching the SDK build to the staged runtime
