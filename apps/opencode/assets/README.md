# OpenCode resources

`make -C apps/opencode` stages `gosh.kex`, Mozilla root certificates and
dependency licenses here. The source copies remain in `apps/tools/gosh`,
`third_party` and the audited dependency checkout; `manifest.json` records
their hashes. These generated copies are deployment resources.

Copy this entire directory with `../opencode.kex` and `../opencode.kex.env`
to a writable KolibriOS folder. The sidecar resolves these resources relative
to the executable. Public Zen models are enabled by default.
