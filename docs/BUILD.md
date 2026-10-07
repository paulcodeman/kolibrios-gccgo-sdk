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

The common application and library makefiles default to the patched
`tooling/gccgo/gccgo-kolibri` frontend. Its build belongs in
`.build-cache/gccgo15-kolibri` (or `GCCGO_BUILD_DIR`); see the compiler preparation
commands in `docs/OPENCODE_PORT.md`. This frontend handles the modern Go syntax
now used by the shared stdlib. A stock compiler can be selected explicitly for
sources it supports:

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

Build every application folder, including examples and test programs. The batch
continues after an error and returns failure if any target failed. Per-target
logs, output hashes and the summary are stored under `.build-cache/build-all/`;
`latest.json` points to the latest report. Compiler, upstream sources and shared
package caches are preserved:

```sh
./make-all.sh
./make-all.sh --rebuild --jobs 4
```

`--rebuild` removes each application's generated output before compiling and
linking it again, while retaining shared dependency objects. To rebuild a
target's dependencies too, use its `clean-cache` target; this only removes its
package and ABI objects, preserving prepared compiler and upstream sources.

OpenCode uses the same `tooling/kolibri-app.mk` as other applications:

```sh
./build-app.sh opencode
make -C apps/opencode
```

Both commands write `apps/opencode/opencode.kex`. Its Makefile prepares and
audits the pinned upstream sources and dependencies automatically; originals
stay in `.build-cache/opencode-upstream`, and build sources/vendor adapters in
`.build-cache/opencode`. `APP_SOURCE_DIR` lets the shared makefile compile a
staged entrypoint without copying application sources into the SDK stdlib.
OpenCode enables KPack by default because the complete CLI exceeds the native
loader's limit before compression. Packaging and native tests use this same
per-folder binary.

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

An application or library with its own `vendor/modules.txt` automatically uses
that vendor tree before the SDK's other external package roots. The common
`tooling/kolibri-compat.mk` applies the SDK adapters before Go imports are
resolved. The same setup serves both application and library builds.

For a vendor directory elsewhere, add `VENDOR_DIR = /path/to/vendor` to the
Makefile or pass it to make. Shared adapters, tested module versions, licenses
and fingerprints are recorded in `third_party/kolibrios-compat/ADAPTERS.json`.
Version mismatches and local edits produce an explicit build error. Validate and
update the shared adapter when a dependency API changes instead of adding an
application-specific copy. Original dependency sources remain unchanged.

`KOLIBRI_COMPAT_MODULES` optionally limits staging to a space-separated list of
module paths. `KOLIBRI_COMPAT=0` disables automatic staging while retaining the
selected vendor root; `KOLIBRI_COMPAT_RECORD` overrides its fingerprint record.
The standalone `tooling/stage-kolibrios-compat.py` provides the same staging and
read-only verification outside make.

Dependency discovery uses the same package-root order as compilation. A missing
import or a package containing only foreign-platform sources stops the build
before compilation and names the affected package. Cached exports cannot hide
that missing source dependency.

You can override these variables per target:

- `OPT_LEVEL` (default `-Os`) for size-optimized builds
- `PACKAGE_DIRS` to precompile additional shared packages
- `FIRST_PARTY_DIRS` to add extra in-repo package roots
- `THIRD_PARTY_DIRS` to add extra external package roots
- `VENDOR_DIR` to use an external vendor tree with shared KolibriOS adapters
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
- Application package and ABI artifacts are cached in `.build-cache/` by default;
  libraries use `.build-cache/libraries/` because their debug settings differ.
  `BUILD_CACHE_NAMESPACE` selects an explicit namespace when needed.
- `make clean` removes local target outputs, `make clean-cache` removes package
  and ABI artifacts in the selected namespace, and `make distclean` does both.
  Prepared compiler, original-source and vendor caches are retained.
- `.kex`, `.obj`, `.gccgo.o`, `.gox`, `.o`, `.pkg/`, and `.build-cache/` build outputs are ignored by git.
- The bundled `tooling/bin/kpack` is a Linux x86_64 binary; override
  `KPACK_BIN` if you are on another host.

## Portable native launch environment

KolibriOS executables can load a literal `<executable>.env` sidecar before
dependent Go packages initialize. Set `KOLIBRI_WORKING_DIRECTORY=.` to choose
the executable's directory as the initial working directory. Other relative
values of this setting are also resolved against the executable's directory.
Without this setting the inherited directory and literal environment behavior
are unchanged.

With this setting enabled, standard relative filesystem variables (`HOME`,
`XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `TMPDIR`,
`TMP`, `TEMP`, `SHELL`, `SSL_CERT_FILE`) are anchored to that initial directory.
Relative elements in `PATH`, `SSL_CERT_DIR`, `XDG_CONFIG_DIRS`, and
`XDG_DATA_DIRS` are anchored individually. Other values remain literal: no shell
expansion, trimming, or command evaluation occurs. Absolute paths remain intact.
SDK child processes retain their explicit environment and working directory.

For example:

```text
KOLIBRI_WORKING_DIRECTORY=.
HOME=.
XDG_CONFIG_HOME=./.config
SHELL=./gosh.kex
PATH=.:/sys:/sys/develop
SSL_CERT_FILE=./certs/ca-bundle.crt
```

Omit `TMPDIR` to use the SDK's existing `/tmp0/1` default. Standard ColibriOS
autorun creates this FAT RAM disk with `/SYS/TMPDISK A0`; a custom autorun must
create it as well. RAM-disk contents disappear on reboot. Applications needing
persistent SQLite databases should keep their data on a writable disk; ISO
volumes are read-only.
