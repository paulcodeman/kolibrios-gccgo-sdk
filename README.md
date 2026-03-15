# kolibrios-gccgo-sdk

[![build](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml/badge.svg)](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml)

Bootstrap Go SDK for building KolibriOS applications with `gccgo`.

## What This Repo Provides

- KolibriOS syscall ABI stubs and runtime glue (`platform/abi/`)
- Go bindings and higher-level wrappers (`platform/kos/`)
- Stdlib shims for the current bootstrap subset (`stdlib/`)
- Example apps and fuller utilities (`examples/`, `apps/`)
- Shared build logic and linker/startup templates (`tooling/`)
- Third-party sources vendored under `third_party/`

## Repository Layout

- `apps/` - fuller KolibriOS utilities built on the same bootstrap SDK
- `apps/cmm/` - ports from C-- sources (prefixes removed)
- `docs/` - bootstrap and build documentation
- `examples/` - curated public KolibriOS demo applications
- `templates/` - app templates for new projects
- `platform/abi/` - syscall assembly stubs and runtime glue used during linking
- `platform/kos/` - raw Go bindings and small higher-level wrappers
- `tooling/` - shared bootstrap make logic and linker templates
- `stdlib/` - bootstrap-compatible stdlib shim sources
- `stdlib/ui/` - minimal UI helpers built on top of `kos`
- `third_party/` - external dependency sources (for example `github.com` and `gopkg.in`)
- `build-app.sh` - build a single app or example
- `new-app.sh` - scaffold a new example app
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

## Quick Start

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

Create a new scaffolded app:

```sh
./new-app.sh demo "KolibriOS Demo"
```

The output `.kex` is written next to each target, for example
`examples/uiwindow/uiwindow.kex`.

## Notes

- `sysfuncs.txt` is the source of truth for syscall numbers and register
  conventions.
- `tooling/kolibri-app.mk` accepts ordered `PACKAGE_DIRS` and can resolve
  imports from `platform/` (first-party) and `third_party/`.
- The build defaults to `gccgo-15`; override with `GO=gccgo` if your binary name
  differs.
- Set `KEEP_PKG=1` to reuse `.pkg` artifacts across multiple builds.
- Set `KEEP_ABI=1` to reuse ABI objects across multiple builds.
- Set `FAST_PKG=1` to avoid package rebuild cascades in batch builds.
- Set `KPACK=1` to run `kpack` on the final `.kex` (override the binary with
  `KPACK_BIN=/path/to/kpack`).
- CI uploads built `.kex` files directly as the `kex-artifacts` workflow
  artifact and runs `kpack` by default. Grab it from the latest run in the
  Actions build workflow:
  [Actions build workflow](https://github.com/paulcodeman/kolibrios-gccgo-sdk/actions/workflows/build.yml).
- `.kex` build outputs are ignored by git.
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
