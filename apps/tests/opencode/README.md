# OpenCode port tests

Smoke tests, native test services, the QEMU test launcher, and fixture data for
the Go OpenCode port live here. They are development tools and are not part of
the OpenCode release package.

The production build entrypoint and launch adapter remain in `apps/opencode/`.
Shared dependency adapters are registered in
`third_party/kolibrios-compat/ADAPTERS.json`; the reusable native shell is in
`apps/tools/gosh/`.

Build a fixture from the repository root in WSL:

```sh
make -C apps/tests/opencode/smoke GO="$PWD/tooling/gccgo/gccgo-kolibri" FAST_PKG=1
```

`build-app.sh` also resolves fixture names in this directory. QEMU checks use
`tooling/run-opencode-smoke.py`, with native CLI scenarios in `tooling/tests/`.
See `docs/OPENCODE_PORT.md` and `docs/OPENCODE_STDLIB.md` for check commands.
