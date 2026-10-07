# Shared KolibriOS dependency adapters

These platform files support clipboard access, terminal input, filesystem
watching, SQLite locks, DNS messages, and character encodings. They can be used
by any SDK application with the dependency versions in `ADAPTERS.json`.
Original dependency sources remain in the application's vendor tree.

`ADAPTERS.json` is the shared registry. Its `origin` fields identify the single
maintained copy of each file under `third_party/`: terminal and cancellable-reader
adapters live beside their original shared library sources. Other adapters stay
in this overlay so existing SDK libraries with different versions are preserved.

The DNS message implementation and character encoding implementation/tables are
unchanged upstream sources from x/net v0.39.0 and x/text v0.24.0. They supply
packages omitted by the original application's vendor selection. Their
`_kolibrios.go` filenames select them for this platform; their algorithms are
byte-identical to the original files recorded in `ADAPTERS.json`. The remaining
files are ten platform adapters and six source-selection manifests. Upstream
licenses are retained under `licenses/` and fingerprinted in that manifest.

The golang.org/x/term adapter delegates to Charm's terminal backend, so raw-mode
and password handling share one implementation. File watching delegates to the
upstream polling watcher, SQLite retains its existing dot-lock VFS, and
cancelreader retains upstream's fallback reader for other input types.

The SDK's application and library Makefiles automatically apply this same tool
when the application/library has `vendor/modules.txt`. Use `VENDOR_DIR` for a
vendor tree elsewhere. The selected vendor tree takes priority during package
resolution, preserving the application's dependency versions. See
`docs/BUILD.md` for the shared build settings.

For manual staging:

```sh
python3 tooling/stage-kolibrios-compat.py --vendor path/to/vendor
python3 tooling/stage-kolibrios-compat.py --vendor path/to/vendor --check
```

Use repeated `--module` options to select particular libraries. When
`vendor/modules.txt` exists, the default selection includes only modules listed
there. A differing module version is rejected; adapters need validation before
use with that version. The tool verifies adapter fingerprints and refuses to
overwrite local changes that differ from its last recorded output.

`kolibrios.sources.json` selects platform implementations through the SDK's Go
source filter. This directory is an overlay, not a standalone package root:
apply it to the original dependency sources before building. Existing SDK
libraries under other `third_party` paths may contain different versions and
remain independently usable.

OpenCode uses this same staging tool. Its console launch adapter remains in
`apps/opencode/bootstrap`; the native shell is a separate application in
`apps/tools/gosh`.
