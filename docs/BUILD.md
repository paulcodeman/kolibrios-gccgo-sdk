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

## Build Commands

Build one target by path:

```sh
./build-app.sh examples/window
```

Build by short name (searched under `apps/` and `examples/`):

```sh
./build-app.sh window
```

Clean a target:

```sh
./build-app.sh window clean
```

## New App Template

Create a new app from the shared template:

```sh
./new-app.sh demo "KolibriOS Demo"
```

This creates `examples/demo` with `package main`, a minimal window loop, and the
shared `tooling/kolibri-app.mk` build wiring.

## Makefile Knobs

You can override these variables per target:

- `OPT_LEVEL` (default `-Os`) for size-optimized builds
- `PACKAGE_DIRS` to precompile additional shared packages
- `FIRST_PARTY_DIRS` to add extra in-repo package roots
- `THIRD_PARTY_DIRS` to add extra external package roots

Example:

```sh
make -C examples/window OPT_LEVEL=-O0
```

## Notes

- The final `.kex` is written next to each target directory.
- Intermediate `.o` and `.gox` files are removed after a successful build.
