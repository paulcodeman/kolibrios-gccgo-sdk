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

## Fonts

- `apps/examples/uiwindow/assets/RobotoMono-Regular.ttf`
  - Source: Google Fonts (Roboto Mono)
  - License: Apache License 2.0
- `apps/examples/uiwindow/assets/OpenSans-Regular.ttf`
  - Source: Google Fonts (Open Sans)
  - License: Apache License 2.0
