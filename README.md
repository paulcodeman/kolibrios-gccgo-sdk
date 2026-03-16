# kolibrios-gccgo-sdk

[![build](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml/badge.svg)](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml)

Bootstrap Go SDK for building KolibriOS applications with `gccgo`.

## What This Repo Provides

- KolibriOS syscall ABI stubs and runtime glue (`platform/abi/`)
- Go bindings and higher-level wrappers (`platform/kos/`)
- Stdlib shims for the current bootstrap subset (`stdlib/`)
- Example apps and fuller utilities (`apps/`, `apps/examples/`)
- DLL-style libraries (`libs/`)
- Shared build logic and linker/startup templates (`tooling/`)
- Third-party sources vendored under `third_party/`

## Repository Layout

- `apps/` - fuller KolibriOS utilities built on the same bootstrap SDK
- `apps/cmm/` - ports from C-- sources (prefixes removed)
- `apps/examples/` - curated public demo applications
- `docs/` - bootstrap and build documentation
- `libs/` - DLL-style library implementations
- `templates/` - app templates for new projects
- `platform/abi/` - syscall assembly stubs and runtime glue used during linking
- `platform/kos/` - raw Go bindings and small higher-level wrappers
- `tooling/` - shared bootstrap make logic and linker templates
- `stdlib/` - bootstrap-compatible stdlib shim sources
- `stdlib/ui/` - minimal UI helpers built on top of `kos`
- `third_party/` - external dependency sources (for example `github.com` and `gopkg.in`)
- `build-app.sh` - build a single app
- `build-lib.sh` - build a DLL-style library
- `new-app.sh` - scaffold a new example app
- `new-lib.sh` - scaffold a new library
- `sysfuncs.txt` - KolibriOS system function specification
- `AGENTS.md` - repository instructions for future agent work

## Build Requirements

The current build is intended for Linux or WSL. The supported bootstrap host is
Ubuntu 24.04 or WSL Ubuntu 24.04.

Install the toolchain with your package manager:

- `gcc`
- `gccgo`
- `gcc-multilib`
- `gccgo-multilib`
- `make`
- `nasm`
- `binutils`
- `mtools`
- `qemu-system-x86`

To avoid system installs, you can drop prebuilt binaries into `tooling/bin`.
The build prefers these when present: `gccgo-15`/`gccgo`, `gcc`, `ld`, `strip`,
`nasm`, and `objcopy`.
The repo currently ships `tooling/bin/nasm`, `tooling/bin/objcopy`,
`tooling/bin/strip`, and `tooling/bin/ld` for Linux x86_64, plus
`tooling/bin/i386-elf-objcopy` for COFF conversion.

## Quick Start

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

The default build generates stdcall C stubs from `exports.txt`. Set
`EXPORTS_STUBS_MODE=bootstrap` in a library Makefile to build an entrypoint-style
library that calls `runtime_kolibri_start` for its exported function.

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

Create a new scaffolded app:

```sh
./new-app.sh demo "KolibriOS Demo"
```

Create a new scaffolded library:

```sh
./new-lib.sh mylib
```

The output `.kex` is written next to each target, for example
`apps/examples/uiwindow/uiwindow.kex`.
Library `.obj` outputs are written next to each library target.

## Notes

- `sysfuncs.txt` is the source of truth for syscall numbers and register
  conventions.
- `tooling/kolibri-app.mk` and `tooling/kolibri-lib.mk` accept ordered
  `PACKAGE_DIRS` and can resolve imports from `platform/` (first-party) and
  `third_party/`.
- The build defaults to `gccgo-15`; override with `GO=gccgo` if your binary name
  differs.
- Set `KEEP_PKG=1` to reuse `.pkg` artifacts across multiple builds.
- Set `KEEP_ABI=1` to reuse ABI objects across multiple builds.
- Set `FAST_PKG=1` to avoid package rebuild cascades in batch builds.
- Set `KPACK=1` to run `kpack` on the final `.kex` (override the binary with
  `KPACK_BIN=/path/to/kpack`).
- Library-only knobs (via `tooling/kolibri-lib.mk`): `OBJ_FORMAT=coff-i386`
  (default), `OBJ_WITH_LIBGCC=1`, `OBJ_EXTRA_OBJS=...`,
  `OBJ_REQUIRE_EXPORTS=0`.
- The bootstrap runtime now supports goroutines/channels and a multi-threaded
  scheduler. Use `runtime.GOMAXPROCS` to configure the runtime thread count.
- See `apps/examples/goroutines` for channel scheduling and `apps/examples/threads` for
  multi-thread slot sampling.
- CI uploads built `.kex` files directly as the `kex-artifacts` workflow
  artifact and runs `kpack` by default. Grab it from the latest run in the
  Actions build workflow:
  [Actions build workflow](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml).
- `.kex`, `.obj`, `.gccgo.o`, `.gox`, `.o`, and `.pkg/` build outputs are ignored by git.
- The bundled `tooling/bin/kpack` is a Linux x86_64 binary; override
  `KPACK_BIN` on other hosts.

## Docs

- `docs/BUILD.md`
- `docs/NATIVE_PORT.md`
- `docs/NATIVE_OWNERSHIP.md`
- `docs/NATIVE_GOTREE_PLAN.md`
- `docs/NATIVE_BRINGUP_CHECKLIST.md`

## License

Unless otherwise noted, the original code and documentation in this repository
are available under the MIT license in [LICENSE](LICENSE).

Third-party materials keep their upstream license status. In particular,
[sysfuncs.txt](sysfuncs.txt) is not relicensed under MIT; see
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
