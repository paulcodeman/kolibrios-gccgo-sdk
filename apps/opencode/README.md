# OpenCode for KolibriOS

Build from the SDK root with `./build-app.sh opencode`, or run
`make -C apps/opencode`. The output is `apps/opencode/opencode.kex`.
Compilation, linking, the runtime and compression use `tooling/kolibri-app.mk`,
the same build rules as other applications. KPack is enabled by default.

The Makefile prepares the pinned original Go CLI and dependency adapters before
resolving imports. Original sources remain in `.build-cache/opencode-upstream`;
audited build sources and vendor dependencies live in `.build-cache/opencode`.
The bootstrap directory contains the native console entrypoint and optional
public Zen discovery. The one application UI correction is recorded in
`patches/status-width.patch`; `make -C apps/opencode audit` checks source parity.

The same build also prepares `assets/` with `gosh.kex`, root certificates
and dependency licenses. `opencode.kex.env` selects these relative paths
and enables public Zen models. Copy `opencode.kex`, `opencode.kex.env` and
the entire `assets` directory together to a writable KolibriOS folder.
The original application manages its configuration, cache and SQLite files
there. The embedded console font does not require an external font file.

Native tests and release packaging read this per-folder binary. Its debug ELF
is kept in `.build-cache/opencode`, outside the deployment directory.
See `docs/OPENCODE_PORT.md` for qualification and system requirements.
