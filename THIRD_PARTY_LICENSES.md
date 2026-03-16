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

## GNU Binutils (objcopy)

- `tooling/bin/i386-elf-objcopy`
  - Built from GNU binutils 2.30
  - License: GNU GPL v3 or later

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
