# Third-Party Materials

Unless otherwise noted, the original code and documentation in this repository
are licensed under the MIT license in [LICENSE](LICENSE).

The following tracked files are third-party materials and keep their upstream
license status.

## KolibriOS System Call Reference

- [sysfuncs.txt](sysfuncs.txt)
  - Upstream notice in the file states:
    - `Copyright (C) KolibriOS team 2004-2021`
    - `Distributed under terms of the GNU General Public License`
  - This repository does not relicense that file under MIT.

## Local Upstream Caches And External Binaries

This workspace may contain locally downloaded or generated upstream artifacts
such as:

- `.cache/upstream-kolibrios/`
- local `.obj` binaries copied from KolibriOS images
- pruned or temporary KolibriOS disk images

Those artifacts are not the original MIT-licensed code of this repository and
retain the license terms of their respective upstream sources.

## GNU Binutils

- `tooling/bin/i386-elf-objcopy`
  - Built from GNU binutils 2.30
  - License: GNU GPL v3 or later
- `tooling/bin/objcopy`
- `tooling/bin/strip`
- `tooling/bin/ld`
- `tooling/bin/as`
  - Built from GNU binutils 2.42 (Ubuntu 24.04 build)
  - License: GNU GPL v3 or later

## NASM (Netwide Assembler)

- `tooling/bin/nasm`
  - Version: 2.16.01 (Ubuntu 24.04 build)
  - License: 2-clause BSD
  - Copyright (c) 1996-2010 The NASM Authors

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

- Redistributions of source code must retain the above copyright
  notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright
  notice, this list of conditions and the following disclaimer in the
  documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

## golang.org/x/image

- `third_party/golang.org/x/image/`
  - Upstream BSD-style license in `third_party/golang.org/x/image/LICENSE`
  - Additional patent grant in `third_party/golang.org/x/image/PATENTS`

## Go net/url

- `stdlib/net/url/url.go`
  - Upstream GCC 13.3.0 libgo implementation, Copyright 2009 The Go Authors
  - BSD-style license in `stdlib/net/url/LICENSE`

## Go Standard Library Packages For OpenCode

- Imported packages and additional dependencies are listed in
  `stdlib/OPENCODE_UPSTREAM.json`, with source versions and SHA-256 hashes.
  - Sources: GCC 13.3.0 libgo and Go 1.23.0
  - Original Go Authors copyright notices are retained
  - BSD-style license: `stdlib/LICENSE` and each imported package's `LICENSE`
  - Platform adaptations and extracted upstream support code are documented in
    `docs/OPENCODE_STDLIB.md` and the source manifest
- `platform/abi/runtime_unsafe_pointer.h`
  - Derived from GCC 13.3.0 libgo `runtime/go-unsafe-pointer.c`
  - Copyright 2009 The Go Authors; BSD-style license: `stdlib/LICENSE`
  - Native descriptor and equality type names are adapted to the SDK ABI

- `platform/abi/runtime_finalizer.h`, `stdlib/runtime/finalizer_kolibrios.go`
  - Adapted from GCC 13.3.0 libgo `runtime/mfinal.go` argument validation,
    finalizer dependency ordering and callback queueing
  - Copyright 2009 The Go Authors; BSD-style license: `stdlib/LICENSE`
  - Collector metadata and worker scheduling are adapted to the SDK runtime;
    callbacks use the existing upstream reflect/libffi backend
- `platform/abi/runtime_sync.h`
  - Adapted from GCC 13.3.0 libgo `runtime/sema.go` semaphore acquire/release,
    direct handoff and condition notification protocols
  - Copyright 2009 The Go Authors; BSD-style license: `stdlib/LICENSE`
  - Wait records and hashed queues use the SDK goroutine scheduler;
    profiling treaps are replaced by native linear queues

## GCC Go Frontend Patches

- `tooling/gccgo/patches/`
  - Patches for GCC 15.2.0 gofrontend; context retains the upstream license
  - GCC sources and locally built compiler binaries retain their upstream licenses

## libffi For Go Reflection

- `stdlib/reflect/ffi_*.c`, `ffi_sysv.S` and associated headers
  - libffi 3.4.2 bundled with GCC 13.3.0; original 32-bit x86 call and Go closure implementation
  - License and copyright notices: `stdlib/reflect/LIBFFI_LICENSE`
  - Go reflection bridges retain the Go BSD-style license in `stdlib/reflect/LICENSE`

## Go Generic Compiler Compatibility Stage

- `tooling/gccgo/compat/go2go/`
  - Derived from the Go project's experimental `dev.go2go` compiler, revision
    `55626ee50b284ae88e5341741b55fb2a6cd4c5d8` (2021-05-24)
  - Original Go Authors copyright notices are retained
  - BSD-style license: `tooling/gccgo/compat/go2go/LICENSE`
  - Adaptations use modern upstream `go/ast` and `go/types` APIs and actual gccgo
    target export data; generated compiler inputs are separate from application sources

## Native Terminal Compatibility

- `third_party/libvterm/libvterm-0.3.3/`
  - Unchanged libvterm 0.3.3 from Paul Evans' official release archive.
  - MIT license: `third_party/libvterm/libvterm-0.3.3/LICENSE`.
  - Archive provenance and per-file fingerprints: `third_party/libvterm/UPSTREAM.json`.
  - OS allocation, serialization, rendering and scrollback adapters are separate
    in `tooling/console/`; the OpenCode application sources are unchanged.
- `third_party/kolibri-printf/`
  - Unchanged embedded formatter from KolibriOS ktcc's libc.obj sources.
  - Marco Paland's MIT copyright/license remain in `format_print.c`.
  - Native builds disable unused floating-point and long-long formatting.
- `third_party/kolibri-newlib-string/`
  - Unchanged memcpy, memmove, memset, strlen, strncpy, strncmp and abs from
    KolibriOS' newlib SDK tree; provenance is in `UPSTREAM.json`.
  - Upstream notices: `COPYING.NEWLIB`, copied from
    https://sourceware.org/newlib/COPYING.NEWLIB; per-file notices are retained.
- `tooling/console/utf8-decode.inc`
  - Original KolibriOS kernel UTF-8 decoder with namespaced assembly labels.
  - GPL-2.0; original copyright/license notice is retained.
- `tooling/console/cp866-unicode.inc`, `vterm-font-tables.h`
  - CP866 mapping extracted from Go x/text v0.23.0 (Go Authors BSD license,
    `tooling/console/LICENSE.charmap`).
  - Exact xterm palette from termenv v0.16.0 (MIT,
    `tooling/console/LICENSE.termenv`).

## Native Cryptographic Random Source

- `third_party/openssl-rand/`, `platform/abi/runtime_entropy_386.S`
  - Original OpenSSL 3.0.15 x86 generator/perlasm sources; fingerprints in
    `third_party/openssl-rand/UPSTREAM.json`.
  - Apache License 2.0: `third_party/openssl-rand/LICENSE.txt`.
  - The generated 386 RDRAND/RDSEED byte routines are extracted unchanged;
    `tooling/generate-cpu-random.py` reproduces them from verified originals.
  - `runtime_entropy.h` adapts CPU feature selection to the native runtime,
    using GCC's upstream CPUID helper. No clock/PID fallback is used.
- `stdlib/crypto/rand/rand_kolibrios.go`
  - Reader lifecycle and platform-call error handling start from Go 1.23.0
    `src/crypto/rand/rand_wasip1.go` (Go Authors BSD-style license).
  - The OS call is adapted to the native CPU random backend.

## System Certificate Store

- `third_party/go-root-certificates/`
  - Original Go 1.23.0 X.509 system-root loader, Go Authors BSD-style license.
  - `stdlib/crypto/x509/root_kolibrios.go` changes only the platform build tag;
    `root_paths_kolibrios.go` supplies native installation paths.
- `third_party/ca-certificates/`
  - Mozilla certificate store converted by curl, dated 2026-09-25, 121 roots.
  - MPL 2.0; original license and verified SHA-256 fingerprints are retained.

## Go Process Runtime

- `third_party/go-process/`
  - Unchanged Go 1.23.0 process and command sources for native porting.
  - Go Authors BSD-style license; source URLs and fingerprints are retained.
  - The production `stdlib/os/exec/exec.go` is unchanged upstream code. Native
    loader attributes and kernel socketpair streams are adapted in separate
    KolibriOS files; source fingerprints are recorded in the stdlib manifest.

## Native Shell And MCP Server

- `third_party/kolibrios-compat/`
  - Shared platform adapters and source selections for eleven dependency modules.
  - DNS messages and character encodings retain byte-identical upstream sources.
  - Tested versions and file fingerprints: `ADAPTERS.json`; original dependency
    licenses are retained under `licenses/`. Application builds stage these
    files alongside their original dependencies through the SDK's shared tool.
- `third_party/github.com/charmbracelet/x/term/`,
  `third_party/github.com/muesli/cancelreader/`, `third_party/golang.org/x/term/`,
  and `third_party/golang.org/x/sync/errgroup/`
  - Original source snapshots at v0.2.1, v0.2.2, v0.31.0 and v0.13.0 respectively.
  - Original licenses and module checksums are retained in each directory;
    `UPSTREAM.json` records file fingerprints. Platform adapters share the
    terminal backend and upstream fallback implementations.

- `third_party/mvdan.cc/sh/v3/`
  - Unchanged mvdan/sh v3.10.0 parser, expansion and interpreter sources.
  - BSD 3-Clause license: `LICENSE`; revision and fingerprints: `UPSTREAM.json`.
  - `apps/tools/gosh` supplies a native entrypoint for OpenCode's configured
    shell. The upstream interpreter supports some Bash features, not all GNU Bash.
- `third_party/github.com/mark3labs/mcp-go/`
  - Unchanged MCP server implementation at v0.17.0, matching OpenCode's client.
  - MIT license: `LICENSE`; version and fingerprints: `UPSTREAM.json`.
  - Native stdio integration fixtures use the original client/server protocol.
- `third_party/github.com/radovskyb/watcher/`
  - Upstream v1.0.7 filesystem polling implementation, MIT license in `LICENSE`.
  - Original sources, module checksums and declared FAT adaptation: `UPSTREAM.json`.
  - The separate fsnotify backend retains its original API. Content fingerprints
    detect same-size edits hidden by FAT's coarse modification timestamps.
    A registration generation protects concurrent watch additions from stale
    snapshots; native content reads yield between chunks.

## Fonts

- `apps/examples/uiwindow/assets/RobotoMono-Regular.ttf`
  - Source: Google Fonts (Roboto Mono)
  - License: Apache License 2.0
- `apps/examples/uiwindow/assets/OpenSans-Regular.ttf`
  - Source: Google Fonts (Open Sans)
  - License: Apache License 2.0
- `third_party/unifont/`
  - Unchanged GNU Unifont 16.0.04 all-plane bitmap glyphs, from Unifoundry.
  - Font license: SIL Open Font License 1.1 (dual licensed with GPL-2.0-or-later
    plus the font embedding exception); original `LICENSE.txt` and
    `OFL-1.1.txt` are retained.
  - Source URLs and SHA-256 fingerprints: `third_party/unifont/UPSTREAM.json`.
  - `tooling/build-unifont.py` packages original bitmap rows without editing
    glyphs; the native adapter performs lookup and framebuffer conversion.
