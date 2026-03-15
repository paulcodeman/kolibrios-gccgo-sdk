# Build Guide

## Supported Environment

The current bootstrap flow is supported on:

- Ubuntu 24.04
- WSL Ubuntu 24.04 on Windows

## Toolchain Installation

Install with your package manager:

- `gcc`
- `gccgo`
- `gcc-multilib`
- `gccgo-multilib`
- `make`
- `nasm`
- `binutils`
- `mtools`
- `qemu-system-x86`

By default the build expects `gccgo-15`. If your binary is named differently,
override it per build:

```sh
make -C examples/uiwindow GO=gccgo
```

## Build Commands

Build one target by path:

```sh
./build-app.sh examples/uiwindow
```

Build by short name (searched under `apps/` and `examples/`):

```sh
./build-app.sh uiwindow
```

Clean a target:

```sh
./build-app.sh uiwindow clean
```

Build all apps/examples in one pass (full rebuild, then clean artifacts except
`.kex`):

```sh
./make-all.sh
```

Build with KPack compression (uses the bundled `tooling/bin/kpack` by default):

```sh
./build-app.sh --kpack uiwindow
./make-all.sh --kpack
```

To use a different `kpack`, point `KPACK_BIN` at your preferred binary.

## New App Template

Create a new app from the shared template:

```sh
./new-app.sh demo "KolibriOS Demo"
```

This creates `examples/demo` with `package main`, a minimal window loop, and the
shared `tooling/kolibri-app.mk` build wiring in a single `main.go`.

## Makefile Knobs

You can override these variables per target:

- `OPT_LEVEL` (default `-Os`) for size-optimized builds
- `PACKAGE_DIRS` to precompile additional shared packages
- `FIRST_PARTY_DIRS` to add extra in-repo package roots
- `THIRD_PARTY_DIRS` to add extra external package roots
- `KEEP_PKG=1` to keep `.pkg` package artifacts between builds (useful when
  building many targets in a row)
- `KEEP_ABI=1` to keep ABI objects (`syscalls_i386.o`, `runtime_gccgo.o`,
  `go-unwind.o`) between builds in a batch
- `FAST_PKG=1` to treat package ordering as order-only and avoid rebuild
  cascades when compiling many targets in a row
- `KPACK=1` to run `kpack` on the final `.kex`
- `KPACK_BIN=/path/to/kpack` to override the `kpack` binary path
- `KPACK_FLAGS=--nologo` to pass flags to `kpack`

Example:

```sh
make -C examples/uiwindow OPT_LEVEL=-O0
```

## Notes

- The final `.kex` is written next to each target directory.
- Intermediate `.o` and `.gox` files are removed after a successful build.
- `.kex` build outputs are ignored by git.
- The bundled `tooling/bin/kpack` is a Linux x86_64 binary; override
  `KPACK_BIN` if you are on another host.
