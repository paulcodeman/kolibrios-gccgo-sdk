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
make -C apps/examples/uiwindow GO=gccgo
```

If you want to use bundled tools instead of system installs, drop prebuilt
binaries into `tooling/bin`. The build will prefer these when present:
`gccgo-15`/`gccgo`, `gcc`, `ld`, `strip`, `nasm`, and `objcopy`.
The repo currently ships `tooling/bin/nasm`, `tooling/bin/as`,
`tooling/bin/objcopy`, `tooling/bin/strip`, and `tooling/bin/ld` for
Linux x86_64, plus
`tooling/bin/i386-elf-objcopy` for COFF conversion.

## Build Commands

Use `build-app.sh` for apps and `build-lib.sh` for DLL-style `.obj` libraries.

Build one target by path:

```sh
./build-app.sh apps/examples/uiwindow
```

Build by short name (searched under `apps/` and `apps/examples/`):

```sh
./build-app.sh uiwindow
```

Clean a target:

```sh
./build-app.sh uiwindow clean
```

Build a KolibriOS DLL-style `.obj` library:

```sh
./build-lib.sh mylib
```

Library targets should include `tooling/kolibri-lib.mk` in their Makefile and
provide an `exports.txt` file. Example:

```
# export_name(argcount) -> GoFunc
hello(0) -> Hello
version(0) -> Version
```

Each line maps an export name to a Go function. `->` and `=` are both accepted
as separators. The optional `(argcount)` suffix is required when C stubs are
generated (default). If the right-hand side is omitted, the export name is used
as the Go function name. Prefix with `@` to use a raw symbol name (no Go
prefixing).

By default the build generates stdcall C stubs (`EXPORTS_STUBS=1`) that forward
to the Go symbols. Use `EXPORTS_STUBS_MODE=bootstrap` for entrypoint-style
libraries that should call `runtime_kolibri_start` for their exported function.

Library builds now default to `OBJ_FORMAT=coff-i386` and require an `objcopy`
that supports `coff-i386`. The repo ships `tooling/bin/i386-elf-objcopy` (built
from GNU binutils with COFF enabled), and the library makefile prefers it
automatically. If you want to use a different `objcopy`, set `OBJCOPY=...`.
If your toolchain lacks `coff-i386`, you can still set
`OBJ_REQUIRE_COFF=0 OBJ_FORMAT=pei-i386` (PEI output may not be loadable by
Kolibri's DLL loader).

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

This creates `apps/examples/demo` with `package main`, a minimal window loop, and the
shared `tooling/kolibri-app.mk` build wiring in a single `main.go`.

## New Library Template

Create a new library from the shared template:

```sh
./new-lib.sh mylib
```

This creates `libs/mylib` with a minimal export, `exports.txt`, and
`tooling/kolibri-lib.mk` build wiring.

## Makefile Knobs

You can override these variables per target:

- `OPT_LEVEL` (default `-Os`) for size-optimized builds
- `PACKAGE_DIRS` to precompile additional shared packages
- `FIRST_PARTY_DIRS` to add extra in-repo package roots
- `THIRD_PARTY_DIRS` to add extra external package roots
- `KEEP_PKG=0` to disable reuse of cached package artifacts for a build
- `KEEP_ABI=0` to disable reuse of cached ABI objects for a build
- `FAST_PKG=1` to treat package ordering as order-only and avoid rebuild
  cascades when compiling many targets in a row
- `KPACK=1` to run `kpack` on the final `.kex`
- `KPACK_BIN=/path/to/kpack` to override the `kpack` binary path
- `KPACK_FLAGS=--nologo` to pass flags to `kpack`
- Library-only knobs (via `tooling/kolibri-lib.mk`): `OBJ_FORMAT=coff-i386` (or
  `OBJ_FORMAT=pei-i386` if your `objcopy` lacks `coff-i386` support),
  `OBJ_WITH_LIBGCC=1`, `OBJ_EXTRA_OBJS=...`, `OBJ_REQUIRE_EXPORTS=0`,
  `DEBUG=1` (keep debug info), `OBJ_STRIP=0` (skip stripping debug info),
  `OBJ_GC_SECTIONS=0` (disable section GC during `.obj` link),
  `OBJ_GC_ROOT=SYMBOL` (override GC root when `--gc-sections` is enabled),
  `EXPORTS_STUBS=0` (disable auto-generated C stubs),
  `EXPORTS_STUBS_MODE=direct|bootstrap` (stub behavior),
  `EXPORTS_STUBS_STRICT=0` (allow missing arg counts).

Example:

```sh
make -C apps/examples/uiwindow OPT_LEVEL=-O0
```

## Notes

- The final `.kex` is written next to each target directory.
- `make` (or `make obj`) in a library target writes `$(PROGRAM).obj`.
- Shared package and ABI artifacts are cached in `.build-cache/` by default.
- `make clean` removes local target outputs, `make clean-cache` drops the shared cache, and `make distclean` does both.
- `.kex`, `.obj`, `.gccgo.o`, `.gox`, `.o`, `.pkg/`, and `.build-cache/` build outputs are ignored by git.
- The bundled `tooling/bin/kpack` is a Linux x86_64 binary; override
  `KPACK_BIN` if you are on another host.
