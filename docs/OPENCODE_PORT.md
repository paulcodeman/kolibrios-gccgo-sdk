# OpenCode Go CLI on KolibriOS

## Common per-folder build

Build with `./build-app.sh opencode` from the SDK root or
`make -C apps/opencode`. The canonical output is
`apps/opencode/opencode.kex`, beside its Makefile. OpenCode now includes
`tooling/kolibri-app.mk` for dependency compilation, the runtime, linking and
KPack, just like other application folders. Its only extra preparation step
stages and audits the pinned original sources and dependency adapters.
`make -C apps/opencode audit` checks the source inventory.

The build also stages deployment resources in `apps/opencode/assets`: the
canonical `apps/tools/gosh/gosh.kex`, Mozilla root certificates and licenses.
`opencode.kex.env` sits beside the executable and uses `./assets/...` paths,
anchored to the executable directory by the SDK. Public Zen discovery is
enabled by default. Copy the executable, sidecar and entire assets directory
together to a writable KolibriOS folder; configuration, cache and SQLite data
are created there by the original CLI. The Unicode console font is embedded.
The debug ELF stays in `.build-cache/opencode/opencode.kex.debug.elf`.

The adjacent-assets installation passed the native Zen test with the original
model picker, a real remote reply, SQLite integrity and clean exit. Its resource
hashes and relative-path settings are recorded in the native validation JSON.
The staged, packed `gosh.kex` also passed loops, command substitutions,
pipelines, external process status and the original persistent-shell protocol;
evidence is `.build-cache/opencode/assets-shell-native-validation.json`.

The common-build executable is 11,558,649 bytes, SHA-256
`e9de7030c7d5aab7e0d4c86ef367a7e43cb0592801fdb9a5287360b6be47aca9`.
139 of 140 original application Go files remain unchanged; the existing
audited status-width patch is retained. Native tests and portable packaging
consume this per-folder executable, rather than a separate cache binary.
The native Zen test verified an actual Space Bunny Free response, model
selection, SQLite integrity, one-row footer and orderly shutdown with this
hash; evidence is `.build-cache/opencode/zen-public/native/validation.json`.

`./make-all.sh --kpack --jobs 4` builds every application/example/test Makefile
folder and records actual output paths, sizes and SHA-256 hashes in
`.build-cache/build-all/latest.json`. Compiler, upstream and vendor caches
are retained. `--rebuild` cleans only each application's generated output.
Existing portable archives and ISO images contain their historical qualified
executables; building the source tree does not replace those distributions.

The complete batch in `.build-cache/build-all/20261006T230852Z-23903/`
successfully built all 125 application, example and test folders with zero
failures. Every recorded output hash and native executable header was checked.
`libs/otto/otto.obj` was also rebuilt using the separate `libraries` cache
namespace. Libraries may omit DWARF, so they must not replace application
package objects used for native stack traces.

This pass exposed a compiler compatibility bug in merged SDK packages: files
with the same basename in different source directories overwrote each other.
The lowering helper now retains distinct filenames and function-local type
identities, without changes to the UI sources. `tooling/gccgo/test-multidir.py`
verifies host execution at O0/O2, 386 compilation, unchanged input sources and
distinct methods/local types. The general, cross-package and local-type
compiler regression scripts also passed; the build resolver's seven tests
include actual compilation from an external staged source directory.

The old Goro port now reads the local HTTP address from the standard
`http.LocalAddrContextKey`; the browser's duplicate `option` switch arm was
removed. Two stale fixtures were updated for the implemented process streams
and the shared watcher dependency. The console-startup fixture supplies its
own test-only optional model-discovery callback; the full CLI uses the real
bootstrap implementation verified by the native Zen test.

Fresh per-folder binaries were checked in private copies of the current fork
image: `goroutines` closed both channels and completed, `threads` completed
across three native slots, and `boxlib`/`uiwindow` rendered their controls and
nested elements. Provider API and original reflection/configuration fixtures
displayed their PASS markers. Current binary hashes, captures and the full
CLI's actual remote reply qualification are recorded together in
`.build-cache/build-all/native-validation.json`.

## Status notification width correction

The corrected executable is 11,545,205 bytes, SHA-256
`18c83439083bbf06d216a851b9250072686bcc05b1512f1d7bff762f48adc49e`.
139 of the original 140 Go files are unchanged. One small, explicit patch to
`internal/tui/components/core/status.go` keeps the footer on one row.
The pristine pinned checkout is retained. `apps/opencode/patches/status-width.patch`
contains the reviewable diff, and `application-source-compat.json` records
the original and patched hashes. No SDK, runtime or console semantics change.

The original footer computed `availableWidht - 10`, but skipped truncation when
that result was zero or negative. Lipgloss then wrapped the notification into
many rows. With `Zen: Big Pickle` and a conversation at 28.8K tokens, ordinary
upstream Go reproduces an eleven-row footer at 80 columns. Its byte slicing
also corrupts UTF-8 for some Russian notifications. The correction subtracts
actual style padding, uses upstream ANSI-aware truncation, limits the message
to one row, and bounds the complete footer by the terminal width.

`tooling/tests/test-opencode-status-width.py` runs the actual original and
patched component with three model names, English/Russian messages and
40/60/80/120-column terminals. All 24 corrected cases stay within the terminal
width, on one row, with valid Unicode. Evidence is in
`.build-cache/opencode/status-host/validation.json`.
The native Zen test also changes to Big Pickle after a Space Bunny reply;
its 210-second screenshot verifies the one-row footer after confirming the
selection with populated token counters. That selection check does not claim
Big Pickle admission.

## Previous public Zen release and provider support

The public-Zen portable executable is 11,558,560 bytes, SHA-256
`ead84d57579fc022ae01b73982c0f8e3256ed4e78684137d6e46d9ffdb5049a0`.
All 140 pinned upstream application Go files remain unchanged. The optional
launch adapter `apps/opencode/bootstrap/zen_public.go` uses the original
OpenAI-compatible client, model picker, tools and persistence. It belongs to
the application bootstrap, rather than the general-purpose SDK stdlib.

Set `OPENCODE_ZEN_PUBLIC=1` in the executable's environment sidecar and omit
`LOCAL_ENDPOINT`. The adapter retrieves the live Zen model list and the
models.dev catalog, intersects them, and requires explicit zero input and
output prices, tool support, valid context/output limits, a non-deprecated
entry and the Chat Completions protocol. The current intersection has nine
models; this is catalog availability, not a guarantee of server admission.
Models that require Responses or Anthropic requests are excluded.

Space Bunny Free is the default and has been verified without a personal API
key. The original compatible provider has the internal name `local`, so its
picker retains the title `Select Local Model`; the entries `Zen: ...` use the
remote HTTPS endpoint `https://opencode.ai/zen/v1`. Public authentication uses
`public`, as in current upstream OpenCode. The adapter identifies the actual
Go port as `opencode/go-73ee493-kolibrios` and preserves server access refusals.
For example, Big Pickle returned `FreeTierError` during the host probe.

Native Windows QEMU verification opened Ctrl+O, moved the selection with
arrows, confirmed with Enter, sent `hi`, received an actual Space Bunny reply,
persisted the selected model and checked SQLite integrity and TUI cleanup.
The correct catalog context window prevented the former immediate summary.
Evidence: `.build-cache/opencode/zen-public/native/validation.json`, the
105-second model-picker capture and the 150-second response capture.
Reproduce with `python tooling/tests/run-opencode-native-zen.py` on Windows
using the prepared QEMU runtime; the test uses private copies of the images.

Package the verified binary with `python tooling/package-opencode-zen.py
--base QUALIFIED_PORTABLE_DIRECTORY --desktop`. The existing local package
and the user's running VirtualBox VM are not modified. For local inference,
remove `OPENCODE_ZEN_PUBLIC=1` and restore the local endpoint sidecar setting.

## Previous local portable release and CPU requirement

The portable executable is 11,552,580 bytes, SHA-256
`4bc3253a2a67c7b27fd48d7a9f64ca16b62bdda843472b4c1b21b829c64e470f`.
All 140 original application Go files remain unchanged. The adjacent
`opencode.kex.env` resolves relative HOME, configuration, shell and certificate
paths against the executable directory, including launches from `/tmp0/1`.

The current secure random reader requires RDSEED or RDRAND. A CPU without both
returns `crypto/rand: secure CPU random source unavailable`; upstream UUID
creation then panics when the original TUI creates its first session. This
affects both sending the first message and Init Project. Native `qemu32`
execution reproduces the failure. Pentium 3 in Qemu Manager/QEMU 0.11.1 cannot
satisfy this requirement. Increasing guest RAM does not provide CPU features.
There is no clock/PID fallback for cryptographic randomness.

Use a recent QEMU with TCG and `-cpu max`. The full original CLI launched from
RAM with this CPU completes Enter, a real local Qwen response, automatic
summarization, SQLite integrity checks and orderly shutdown. Evidence is
`.build-cache/opencode/enter-ram-chat-files/validation.json`; the independent
unsupported-CPU entropy diagnostic is
`.build-cache/opencode/enter-entropy-files/ENTROPY.txt`.

`tooling/start-opencode-qemu.ps1` starts the separate Windows runtime in
`Desktop/opencode-qemu`, using its own boot and persistent disk copies, RTL8139
user networking and the cached local Qwen server. It does not update Qemu
Manager or the source images. `-CheckOnly` checks runtime files and prints the
selected configuration without booting the guest. Guest OpenCode uses
`http://10.0.2.2:18081/v1` to reach the host server.

Windows QEMU 11.1.0 qualification passes the complete original TUI Enter,
actual host Qwen response, SQLite integrity and orderly exit in
`.build-cache/qemu-windows/validation.json`. The isolated Qemu Manager 7.0
profile is separately inspected through its actual controls and its green
Start button launches the modern Windows backend; the resulting process and
command line are recorded in `manager-launch-validation.json` in the same
directory. The profile uses `Advanced/OptionalString` and `OverrideAll=Yes`;
the unused legacy `OverrideString` key does not supply additional parameters.
This bypasses removed legacy QEMU options such as `-enable-kqemu`, `-soundhw`
and `-net ...,vlan=...`. The modern binary also has the `qemu.exe` alias
expected by the old manager. Embedded display is disabled; SDL provides the
normal interactive QEMU window. `Desktop/opencode-qemu/Start-QemuManager.cmd`
starts a separate copy of the original manager with the prepared profile.

## Native ColibriOS fork qualification

The sibling `../colibrios` repository now contains the native kernel and console
changes and a complete 128 MiB disk image at `build-ru_RU/data/colibrios.img`.
The selected upstream compiler profile builds 2770 rules. The image boots to the
original desktop; OpenCode remains a separate application to copy into it.

The rebuilt original CLI is 11,542,444 bytes, SHA-256
`25e30dab0af9ed8a4ad073f20dfdd653e8c7975fb429847aa9c65cbc6ab8eec9`.
All 140 application Go files remain unchanged. Native semantic checks on the
fork pass verified-TLS inference, Unicode interactive TUI and automatic summary,
file tools, shell/search, MCP and LSP. Installing the release layout into a
temporary copy of the complete disk image also passes actual HTTPS inference.
Evidence prefixes start with `colibrios-` and preserve the older reports.

New binaries use atomic FAT operations 14/15, preserving upstream symlink
operations 11/12 and volume information 13. The earlier archived qualification
below uses its own older kernel with private operations 11/12; keep that archive
paired with its supplied kernel. The new fork automatically packs `console.obj`.

## Earlier archived SDK qualification

Archived qualification (2026-10-06): the complete original CLI runs natively
and has passed actual AI requests over verified TLS, SQLite persistence,
original file tools, native MCP discovery/execution, persistent shell commands
and grep/glob/ls. All 140 application Go files remain byte-identical at revision
`73ee493265acf15fcd8caab2bc8cd3bd375b63cb`. The current source closure is
723 packages / 3064 Go files, without missing imports or empty target packages.
Full original CLI LSP integration, interactive AI through verified TLS, native
Unicode clipboard input, automatic session summarization and orderly TUI exit
also pass on this binary. Qualification checks completed messages and SQLite
integrity; a successful build alone is not sufficient.

Qualified executable: `app/opencode.kex`, 11,553,013 bytes, SHA-256
`9805b7e743d4a9d3f55f5307ac71e9b4adf690cee6032876ce59175603bda9c6`.
Installation layout qualification and a second complete Unicode TUI session
(including automatic summarization and clean shutdown) also pass. Their reports
are `original-cli-installed-package-tls-files/validation.json` and
`original-cli-tui-verified-tls-repeat-qualified-files/validation.json` in the
OpenCode build cache. The normal runtime object rebuilt from current sources
matches the qualified object exactly; optional scheduler tracing is excluded.

The distribution is built by `tooling/package-opencode.py` into
`.build-cache/opencode/release/`. It requires six semantic native reports with
matching SHA-256 hashes for the CLI, kernel and console. The archive includes
the native shell, public Mozilla CA bundle, installation instructions, licenses
and corresponding kernel/console source with reproducible builders. The
distributed kernel and console have been rebuilt from those bundled sources
with exact hash matches. `tooling/tests/verify-opencode-package.py` verifies the
archive and all manifest entries.

`tooling/tests/run-opencode-native-install.py` exercises the actual installation
layout `/hd0/1/opencode/opencode.kex`, a separate project directory and the
distributed HOME/shell/public certificate configuration. The loopback model's
private test CA is added in a separate test-only directory; it is not distributed.
Use `tooling/tests/validate-opencode-native-install.py` to verify its completed
request and database after capture.

The test inference peer is actual host-side Qwen, not an inference engine port
to KolibriOS. Its small 0.5B model can invent repeated tool calls, and a cold
evaluation of the original 9K-token coder prompt can exceed a short test timeout.
Host stream timing and scheduler diagnostics distinguish those conditions from
native execution failures. Current interactive/install fixtures allow five
minutes for the host peer. Paid cloud providers and physical hardware are not
qualified by these loopback QEMU tests. External Git/editors and additional
LSP/MCP servers still need native executables; gosh is not complete GNU Bash.
This development snapshot retains upstream's `unknown` build-time version label;
the distribution records the exact revision and file hashes instead.

`original-cli-native-mcp-env` validates the full original MCP tool pipeline;
`original-cli-native-shell-search-case` validates shell state, stderr, exit 7
and all three search/list tools. Their validators check exact tool results,
SQLite integrity and clean original CLI completion. These controlled provider
fixtures are separate from the actual Qwen inference over verified TLS.

Native filesystem watching uses upstream radovskyb/watcher v1.0.7 through a
separate fsnotify backend. The original poller and license are preserved;
the declared FAT adaptation additionally captures content fingerprints because
timestamps alone can lose same-size edits. A registration generation prevents
an in-flight directory snapshot from falsely removing files added concurrently;
native content reads yield to other goroutines. `upstream-fsnotify-native` verifies
Unicode create, same-size/same-time modification, delete and watcher lifecycle.

`original-lsp-bash-parser-native` validates OpenCode's unchanged LSP client
against actual upstream mvdan Bash parsing, including a real missing-`fi`
diagnostic, a corrected document and clean server shutdown.
`original-cli-native-lsp-ready` additionally verifies the original CLI's
view/write/view pipeline: the real diagnostic is displayed, the missing `fi`
is corrected and a final view confirms cleared diagnostics. Test-image FAT
timestamps are staged in UTC to match QEMU's RTC. This server does not provide
Go/TypeScript analysis.

Native Go child termination now uses exact runtime worker PID/TID records in
documented named memory, then documented native termination calls. The
`process-worker2-kill-before-native` fixture reproduced a leaked worker with
the previous backend. `process-worker2-control-native` passes cancellation
with distinct main and worker OS threads, alongside byte-stream/file tests.
Releasing a child handle retains a detached reaper for SDK resources.

The compatible kernel needs `--exclusive-create --local-streams --fat-name-case`.
`original-fat-case-native` checks filename spelling, case-insensitive lookup,
rename, glob and removal. The fix uses upstream FAT name/LFN routines and no
new syscall ABI. See [KERNEL_FILESYSTEM_EXTENSIONS.md](KERNEL_FILESYSTEM_EXTENSIONS.md).

The upstream libvterm mouse encoder now receives native button and wheel events.
`upstream-mouse-terminal-native` verifies real QMP mouse input delivered through
the console as SGR press/release/wheel sequences. Both upstream terminal package
APIs (`charmbracelet/x/term` and `golang.org/x/term`) share native console state.

The detailed bring-up notes below retain earlier failures and their fixes;
their older audit counts describe those intermediate builds.

Native child streams now pass with the original Go 1.23 `os/exec.Cmd`: stdin,
stdout, stderr, CombinedOutput with a shared descriptor, explicit stderr close,
131073-byte backpressure, streaming pipes, regular-file redirection, preserved
caller file ownership, child exit status and CommandContext cancellation.
Evidence is `upstream-command-streams-native-kernel.log`. Native loader
attributes remain in SDK metadata because KolibriOS's loader does not inherit
a Unix descriptor table. The separate adapter retains Cmd's unchanged original
copying and pipe logic and uses the kernel's existing AF_LOCAL buffers.

The isolated kernel builder's `--local-streams` option fixes AF_LOCAL close:
the peer drains buffered bytes and receives EOF, and writing to a closed peer
returns an error. `local-socket-eof-clean-native` passes with no invalid-peer
diagnostic; `local-streams-http-regression` also passes the network regression.
This kernel is required for cross-process pipes; the stock kernel lacks that
EOF notification. Raw syscall-77 file handles and syscall-75 socket identifiers
remain distinct namespaces.

Unchanged upstream mcp-go v0.17.0 client/server code passes native initialize,
listTools, a 108 KiB Unicode tool response and child shutdown in
`original-mcp-stdio-native-kernel.log`. This is evidence for the protocol and
process backend; full CLI MCP execution is checked separately. `gosh` imports
the original mvdan/sh v3.10.0 interpreter and uses OpenCode's supported shell
configuration; it must not be represented as complete GNU Bash compatibility.

Bring-up began with the unchanged OpenCode LSP protocol package, which
builds into a KolibriOS application and has passed URI and JSON checks in QEMU.
Ten GCC Go frontend patches add `clear`, integer `range`, `min`/`max`,
Go 1.20 unsafe string/backing-store operations, and generic type identity and
private member ownership, embedded generic fields across packages and the
64-bit atomic alignment marker, stable variadic function type export, and
correct interface method-table ownership after re-export.
A separate compiler stage
specializes supported generic declarations without changing input sources.

The earlier source audit selected 719 packages and 3037 Go files with no missing
imports. A check of that complete original-source closure now reports no type
errors. All 719 dependency packages and the original main.go have also compiled
to native objects. The complete CLI links to a 61,518,556-byte MENUET01
executable (build-full-link-complete.log), with its debug ELF retained.
The original CLI now runs natively: `-v`, `--help`, and a complete
noninteractive request through a fixed integration provider have passed.
The collected original SQLite database passes integrity_check and contains
both the user prompt and the completed assistant response. This fixture
checks the actual CLI pipeline; it does not prove real AI inference.

The complete CLI exceeds upstream's 16 MiB raw-file loader limit. Builds now
retain a debug ELF and use original KPACK compression; the first complete
packed artifact is 10,608,254 bytes. The runner accepts both MENUET01 images
and recognized KPACK containers while rejecting incomplete/debug ELF files.
Its first native startup exposed the bootstrap's 16,384-record unwind scan
limit: this CLI has 49,396 FDEs. The native runtime now uses libgcc 15.2's
find_fde_tail sorted-header search, adapted to the single flat image, and
the fallback scan is bounded by its section rather than a record-count cap.
Caller PCs retain libgcc's own adjustment. That fix reached the next real
startup failure: the old custom log package left log/internal.DefaultOutput
nil for original slog. Complete, byte-identical Go 1.23 log/log.go replaces
the custom logger and the old extracted termination methods. Its initialization,
both log/slog output bridges, source formatting and concurrent records pass
in log-bridge-native-kernel.log. The next startup failure was a stack overflow
inside original regexp compilation. libgo's non-split-stack StackMin is now
used: 2 MiB per goroutine on i386, rather than the SDK's old 256 KiB.
A 700-level nested regexp compiled before main passes natively in
regexp-stack-native-kernel.log. The full CLI then passed version and help
checks in original-cli-version-upstream-stack-native and original-cli-help-native.

original-cli-fixed-provider-native exercises the original CLI's model discovery,
title request, streaming coder response, JSON output and SQLite persistence.
It displays the expected response and finishes. Its debug log reports both
Request completed and Non-interactive run completed; the original database
contains one session and two completed messages.

The QEMU user-network option attaches original NETCFG/RTL8139 and configures
IPv4 through the documented syscall 76 API. The native upstream net/http GET
to a host-only test endpoint passes in host-network-autoload-native.
A real Qwen2.5-0.5B model is available on a loopback llama.cpp endpoint; direct
host inference has passed. The original CLI's first native real-provider run
reached the title request but triggered a kernel TCP persist-timer page fault.
FASM symbols map EIP 800370EF to tcp_set_persist: EBX holds a timeout, but
upstream used it as the socket pointer. The isolated kernel builder now uses
EAX, the actual socket pointer; the generated kernel differs by one instruction
byte. original-cli-real-qwen-persist-native then transferred the complete
request without a kernel fault; its server rejected the 9291-token prompt
because the test context was only 8192 tokens. With a 32768-token context,
original-cli-real-qwen-32768-native exposed the original OpenAI SDK's handling
of an SSE comment ping as an empty JSON event. The same exact SDK reproduces
unexpected end of JSON input under the host Go toolchain. The test server now
uses its documented --sse-ping-interval -1 option; OpenCode and its SDK are
unchanged.

original-cli-real-qwen-no-ping-native completes a real model exchange in QEMU.
The screen displays a JSON response and OpenCode [Finished]. The original
SQLite database passes integrity_check and holds the exact user prompt and a
completed local.kolibri-test-model assistant message; the debug log reports
Non-interactive run completed without errors. The tiny model answered 9 to
4 + 7, while a host replay of the same coder messages returned 11. This is
proof of real native request/response and persistence, not a passed arithmetic
or deterministic response-equivalence test.

The separate KolibriOS bootstrap now connects the unchanged CLI to native
libvterm 0.3.3. Original libvterm, newlib string primitives and the KolibriOS
embedded printf sources are fingerprinted and unchanged; only the console's
OS boundaries are adapted. The original console ABI and opt-in CP866/UTF-8
clients continue to work. `console-vterm-plain-coff-native` and
`console-vterm-history-native` verify split UTF-8, actual Unicode codepoints,
exact ANSI256 foreground/background colors, wide-character cell geometry,
cursor query responses and alternate-screen restoration in KolibriOS.
`console-vterm-legacy-utf8-native` also verifies the older UTF-8 mode after
linking the C backend. Host checks verify scrollback and alternate-screen
history restoration; the adapter retains up to 1000 lines.

`original-cli-tui-vterm-confirm-native` displays the original colored project
dialog, accepts the original keyboard selection and Ctrl-C confirmation,
logs `TUI message channel closed` and `TUI exited with result`, and finishes
without a runtime/kernel fault. Screens at 24s and 60s show the original dialog
and finished window. The Unicode renderer now imports all 126086 unchanged
GNU Unifont 16.0.04 glyphs and embeds their bitmaps in the separate bootstrap.
`console-unifont-native` verifies actual emoji, CJK, Cyrillic and combining
glyph rendering in KolibriOS. Host framebuffer checks compare every rendered
pixel of representative glyphs with the imported bitmaps, including the second
cell of wide characters. Complex-script shaping and color emoji are not added.
The complete CLI with the embedded font passes the same original TUI selection
and confirmed exit again (`original-cli-tui-unifont.log`).
`console-unicode-keyboard-native` verifies physical QMP key events translated
through the native CP866 layout into UTF-8 `аБ`, using libvterm's unchanged
encoder. All 256 keyboard-byte mappings also pass host comparison. Resize,
clipboard and broader TUI controls still need native verification.

`original-cli-tui-real-vterm-native` also passes actual interactive AI usage:
QMP types `say hello` through the native keyboard; the unchanged TUI sends it
to the loopback llama.cpp/Qwen endpoint, displays `Hello, how can I assist you
today?`, performs its original automatic summary flow, and exits through
Ctrl-C and the original confirmation. The collected SQLite database passes
integrity_check and contains the typed user prompt and completed real-model
assistant response. `native-tui-real-ai-db-check.json` records these messages.
The known original file picker logs an empty-directory cursor error on the
blank FAT fixture, but there are no other ERROR records or native faults.
This proves the original interactive request/display/persistence pipeline;
the inference engine still runs on the host.

`original-cli-native-tools` also executes the original `view`, `write`, `view`,
`edit`, `view` sequence against native FAT files. A controlled native HTTP peer
requests each tool and checks its result; it is not an inference engine.
The final file contains `Original edit PASS`, all five persisted tool results
have `is_error=false`, and the original SQLite database passes integrity_check.
`tooling/tests/validate-opencode-native-tools.py` verifies the collected evidence.

`original-cli-real-ai-verified-tls-native` runs the complete unchanged CLI
against the actual host Qwen model through a verified HTTPS endpoint. The
explicit test CA is loaded through `SSL_CERT_FILE`; certificate verification
remains enabled. The command returns `Hello! How can I assist you today?`,
exits normally and persists the response in an intact SQLite database.
`tooling/tests/validate-opencode-native-tls-ai.py` checks these results.
The earlier 144-second attempt was stopped during cold model prompt evaluation;
the 300-second fixture completes the full exchange, with finished output
already visible in its 72-second capture.

The SDK system-root loader now uses Go 1.23.0 `root_unix.go`, changing only
the platform build tag and supplying native certificate paths separately.
The verified, unchanged Mozilla/curl bundle dated 2026-09-25 supplies 121 roots.
Install it as `/sys/certs/ca-bundle.crt` or
`/hd0/1/opencode/certs/ca-bundle.crt`; both directories are also scanned for
additional PEM roots. Original `SSL_CERT_FILE` and `SSL_CERT_DIR` overrides
are supported. `system-roots-verified-native` loads the default bundle and a
separate test CA from the standard directory and verifies actual HTTPS.
`system-roots-untrusted-native` loads the default bundle alone and rejects
the same endpoint with the original `x509.UnknownAuthorityError`.

The entire Go 1.23.0 `os/exec/exec.go` now replaces the partial command wrapper
without changing its bytes. The native lookup sidecar retains the original
PATH-search algorithm and checks MENUET01/KPACK headers in place of Unix
executable permissions. Portable `os.StartProcess`, `ProcAttr`, `Process.Wait`
and ProcessState API wrappers are imported from Go; the native startup bridge
preserves exact argv, environment and cwd before dependent package initialization.
Normal main return and explicit os.Exit report actual codes to the parent.
`process-native-attributes` verifies Unicode, empty/quoted/backslash arguments,
isolated child environment, cwd, exit 7 and normal return 0 in separate native
processes. `upstream-command-native` also verifies original Cmd.Run,
ExitError.ExitCode, CommandContext cancellation and the original GODEBUG
last-value parser with live environment updates. Standard null endpoints work;
actual child pipe streams now pass the original Cmd and full CLI checks above.

`local-socket-buffered-eof-native` verifies the documented syscall 75/10
socketpair with 65537 exact bytes, buffer draining before EOF and an error on
writing after peer close. The isolated kernel builder's `--local-streams`
option breaks AF_LOCAL peer links and notifies the receiver on close.
These transport checks are supplemented by the full original CLI MCP and
shell integration checks described above.

The console builder's native COFF conversion handles ELF i386 absolute/PC-relative
relocations explicitly, including their different addends, BSS sizes and plain
symbol records. KolibriOS' loader does not skip auxiliary COFF symbols, and
cross-format objcopy retained unsupported relocation numbers/local-symbol
addends; the first failed captures are preserved. Native fixture and original
TUI success, rather than an object-file conversion alone, validate this path.

The SDK metadata generator now emits repeated filenames once instead of
copying escaped paths for every PC. For this CLI its C source shrinks from
about 777 MB to 84 MB without dropping records (49,396 function symbols and
1,441,424 source positions). test-link-symbols.py verifies compiled table
contents, escaped Unicode paths, empty tables and repeated-path source size.
Native startup failures and their ELF files are retained for diagnosis; an
image capture without successful command output is not a passed CLI check.

The interrupted Azure public-package build is repaired in the compiler's
private-function forwarding bridges. Result names are omitted and generated
parameter names cannot shadow the forwarded function. A host regression
executes multiple-result and variadic bridges, and the unchanged MSAL public
package compiles for native 386. Full builds now use four workers and direct
import dependencies; the independent-package build path was also verified
with a clean-cache HTTP application and its native QEMU checks.

The compiler emits go/types' resolved length for inferred array literals,
including keyed constants from later source files. The unchanged x/image/ccitt
package compiles. Slice-to-array value conversions use GCC's existing checked
array-pointer conversion and copy; the zero-length case evaluates its source
once without dereferencing a nil pointer. The unchanged OpenTelemetry attribute
package compiles. Host tests cover sparse initialization order, aliasing,
named types and short-slice panics. `slice-array-native-kernel.log` verifies
value/pointer aliasing, nil zero-length arrays and a recoverable runtime.Error
with upstream boundsConvert text. The bootstrap delegates this panic to
libgo's boundsError implementation; other bounds panic entrypoints still need
their equivalent runtime integration.

The compatibility compiler now qualifies dot-imported types in anonymous
struct fields using the type use recorded by go/types, rather than the field
definition. Unchanged Chroma lexers compile with this fix. Direct ranges over
the original `strings.SplitSeq` and `SplitAfterSeq` are lowered to ordinary
slice ranges while preserving enclosing control flow and defer. Host tests
compare original and lowered execution, including labeled continue, early
return, assignment and Unicode. This intrinsic allocates the complete split
slice. General function ranges now use Go 1.23's callback, nonlocal-return
and runtime state-check protocol, adapted to go/ast. Zero, one and two yielded
values, nested ranges, assignment, per-iteration closure capture, ordinary
break/continue and valued returns pass against upstream host execution and in
KolibriOS (`rangefunc-mixed-native-kernel.log`). Incorrect yield-after-stop,
yield-after-exit and swallowed loop panics produce upstream runtime.Error
values. Labeled transfers and defer inside a general function-range body are
explicitly rejected pending enclosing-frame support; those forms still work
for the string slice intrinsic. This is not complete Go 1.23 compiler support.
The unchanged google.golang.org/genai and original OpenCode internal/llm/provider
packages compile for native 386 with this rewrite, including Gemini streaming.
An inferred local SplitSeq/SplitAfterSeq variable is also lowered when its
only use is the immediately following range in the same block, with no goto.
Reused, reassigned, escaping and explicitly typed iterators retain their
original representation. Execution comparisons cover stateful repeated use;
the unchanged OpenCode internal/diff package compiles with this restriction.

The complete unchanged libgo os/env.go and internal/testlog/log.go now replace
the old environment API and extracted Expand helpers. syscall/env_unix.go's
protected table and mutation algorithms are retained with a KolibriOS loader
backend. A literal `<executable path>.env` sidecar is loaded before dependent
package initialization, including original local-model discovery. Empty values,
first duplicate wins, literal/UTF-8 values, deletion, independent snapshots and
concurrent mutation on two OS threads pass in
`environment-native-kernel.log`; the fixture captures provider configuration
before main. The test runner accepts `--environment-file`. No cgo environment
mirror or child-process environment inheritance is claimed.

The runner can start a native test service with `--service-binary`, waiting for
SERVICE.READY before starting the application, and copy requested FAT files
with `--collect-file`. The controlled provider test peer passes model discovery,
ordinary completion and SSE between two native processes in
`provider-service-native-kernel.log`. Its replies are fixed integration fixtures;
these peer-only tests do not establish original CLI execution or AI inference.
The subsequent original-cli-fixed-provider-native check above exercises the
complete original CLI against this peer; real AI execution remains separate.

The native launch adapter in `apps/opencode/bootstrap/console_kolibrios.go`
opens CONSOLE.OBJ and invokes the unchanged upstream main function. A completed
command leaves its console output visible until the window is closed. This
adapter is staged as a separate platform file, with a recorded fingerprint;
the 140 original OpenCode files and original main.go remain untouched.
`arguments-native-kernel.log` verifies loader argument handling, including
spaces, an empty argument and UTF-8 text.
`console-startup-native-kernel.log` and its 36-second screen capture verify the
same launch adapter with native stdout: the command returns, the output remains
visible, and CONSOLE.OBJ marks the window as Finished. This fixture is not the
full CLI; original CLI startup has since passed the version/help checks above.
The unchanged auto/sdk tracer also passes native interface dispatch, span
creation and End in otel-method-table-corrected-native-kernel.log. Its default
span is recording until End; the test follows that original behavior.

Native checks in `cli-clipboard-cleanup-native-kernel.log` verify original
provider parameter handling, private-field reflection access, buffer growth,
relative paths, scatter buffers and UTF-8/CP866/CP1251 clipboard round trips.
`resolver-native-kernel.log` verifies real TCP DNS A/AAAA, SRV and TXT replies,
NXDOMAIN, SERVFAIL, StrictErrors and canceled contexts. DNS message validation
and record algorithms use original libgo sources, with unchanged x/net
message and x/text charmap sources at the application's pinned versions.
UDP DNS and search-domain configuration remain outside the native backend.

The SDK kernel builder now offers atomic FAT file and directory creation via
`--exclusive-create`; its ABI is documented in
[KERNEL_FILESYSTEM_EXTENSIONS.md](KERNEL_FILESYSTEM_EXTENSIONS.md). This enables
the upstream temporary-file/directory algorithms without a destructive
check-then-create fallback. Working-directory changes use an application-wide
path because kernel function 30 is per thread. Native collector snapshots
provide actual heap/allocation/GC counters; other memory-accounting fields
remain untracked. Tracing and profile sampling requests explicitly fail.

`exclusive-cwd-memstats-native-kernel.log` verifies these changes in KolibriOS:
eight competing exclusive creates have exactly one winner, existing contents
survive, duplicate directories return ErrExist, and the original CreateTemp,
MkdirTemp and MkdirAll algorithms work. It also checks shared working-directory
changes across Go threads, relative files opened before a directory change,
child directory inheritance, and real allocation/collection counter changes.
The child and parent complete without a kernel fault or runtime panic.

The native terminal adapters now connect x/term raw-mode state and cancelreader
to os.Stdin's nonblocking CONSOLE.OBJ input. `terminal-adapter-native-kernel.log`
verifies actual size queries, cancellation of a waiting read, arrow escape
sequences split into one-byte reads, mode restoration, and password editing
without echo. The visible console capture contains no test password. The size
query requires the SDK console library described in
[CONSOLE_EXTENSIONS.md](CONSOLE_EXTENSIONS.md).
The native signal backend now uses the original libgo handler, NotifyContext
and Stop algorithms with console Ctrl-C capture and actual size polling.
`signal-delivery-native-kernel.log` verifies two subscribers, context
cancellation, raw Ctrl-C as input, Ignore/Reset, and no delivery after Stop.
The visible console also reports PASS. Bubble Tea's resize listener now uses
the unchanged body of upstream signals_unix.go. Resize event delivery still
needs native validation, as do the original TUI and Unicode/color behavior.
Other signal numbers have no native event source; closing the console retains
its upstream immediate parent-thread termination. Default Ctrl-C restoration
returns to ordinary native input, rather than POSIX process termination.

The compiler also removes a semantically universal interface union such as
`crypto.PublicKey | []byte` after go/types proves it is an ordinary method-set
interface. It preserves its named type, explicit methods and embedded ordinary
interfaces, while restricted constraints remain untouched. The unchanged
golang-jwt/jwt/v5 package now compiles; a host test checks assignability and
method sets before and after rewriting.

## Source and scope

- Repository: https://github.com/opencode-ai/opencode
- Pinned revision: `73ee493265acf15fcd8caab2bc8cd3bd375b63cb`
- Module requires Go 1.24.0.
- Target: KolibriOS, 32-bit x86, bootstrap SDK runtime.
- OpenCode source edits: zero; sources and dependencies are staged under
  `.build-cache/opencode/`.

This is the Go implementation of OpenCode. `opencode-smoke.kex` is a test of
its LSP package and compiler/runtime support; it is not an AI agent executable.

## Prepare and attempt the build

Run from the SDK root in Linux or WSL. `HOST_GO` is the ordinary Go toolchain
used to download module dependencies, independently of the target compiler.
Use Go 1.24 or later, or a Go installation that can download the required
toolchain automatically.

```sh
make -C apps/opencode prepare HOST_GO=/path/to/go
make -C apps/opencode audit
make -C apps/opencode probe
make -C apps/opencode build
```

`audit` fails if source discovery or original-source verification fails;
`probe` still returns nonzero for unsupported language features.
The SDK build attempt records the actual compiler error in
`.build-cache/opencode/build.log`. The audit records missing packages and their
importers in `.build-cache/opencode/audit.json`.

Preparation pins and checks the upstream revision, copies sources without
rewriting them, and vendors dependencies from its original `go.mod`/`go.sum`.
Audits and builds also compare all 140 staged OpenCode Go files and the
application entrypoint with the clean upstream checkout before compiling.
Do not manually downgrade libraries or rewrite generic declarations in that
checkout to make the probes pass.

## GCC Go changes

The patches in `tooling/gccgo/patches/` target GCC 15.2.0:

- `0001-go-clear.patch`: predeclared `clear`, type checking, escape handling,
  export support, and lowering to existing `runtime.mapclear` and
  `runtime.memclrHasPointers` entrypoints. Slice clearing uses length, preserves
  capacity and the backing tail, and evaluates its operand once. Map clearing
  also removes NaN keys.
- `0002-integer-range.patch`: integer bounds and index inference, one bound
  evaluation, hidden loop counter, `break`/`continue`, and an iteration-local
  declaration for `:=` so escaped addresses and closures retain distinct values.
- `0003-min-max.patch`: ordered operand type inference, constant folding,
  diagnostics, escape/export handling, and native comparisons. Every argument
  is evaluated once; floats propagate NaN and preserve the required signed
  zero, and equal strings select the first operand. No new runtime ABI is
  required. Semantics follow the [Go specification](https://go.dev/ref/spec#Min_and_max).

- `0004-unsafe-string-data.patch`: `unsafe.String`, `StringData`, and `SliceData`,
  operand validation, escape propagation, and export support. String construction
  evaluates pointer and length once in source order and calls original Go runtime
  length/overflow checks imported into the SDK.

- `0005-generic-type-identity.patch`: canonical identities and shared runtime
  descriptors for generic specializations emitted in different packages,
  retaining original package paths and reflection names.
- `0006-generic-private-members.patch`: preserve original package ownership of
  private fields and methods in checked generic templates. Compiler aliases
  expose private ordinary functions and pointers to private package variables
  without copying their state.
- `0007-generic-embedded-fields.patch`: restore the original field name and
  visibility of embedded generic specializations, including forward type
  declarations. Keyed literals, promoted methods and reflection keep Go's
  original embedding semantics.
- `0008-atomic-align64.patch`: give the original private atomic `align64`
  markers eight-byte alignment on 386, including imported type definitions.
  Empty structs in unrelated packages retain their ordinary alignment. The
  native heap metadata also preserves eight-byte payload alignment.
- `0009-export-function-varargs.patch`: include variadicness when sorting
  function export types with identical backend names. Exported builtin min/max
  inline bodies and ordinary callbacks no longer crash the comparator.
  The unchanged bubbles/viewport compiles. `test-export-types.py` checks
  repeated export bytes, a separate consumer, and rejection of a variadic
  function assigned to a slice-function signature. Native invocation passes
  in `export-types-native-kernel.log`.
- `0010-interface-method-table-owner.patch`: use the original private method's
  package identity when deciding whether a re-exported interface table belongs
  to its concrete type's package. The unchanged auto/sdk and otel packages
  exposed a consumer requesting a global table whose owner defined it locally.
  `test-interface-tables.py` reproduces that native GCC failure without the
  compatibility rewrite, checks local and owned tables, and retains rejection
  of a private-method imitation in another package. All eleven prefixes of
  the ten-patch stack produce identical final compiler sources.

These are narrow extensions, not complete Go 1.24 support. The wrapper also uses
`tooling/gccgo/compat/`, derived from Go's pinned experimental `dev.go2go` compiler
(ref `55626ee50b284ae88e5341741b55fb2a6cd4c5d8`). Its modern parser and type checker
validate original sources and specialize generic functions/types into separate
compiler cache inputs. Ordinary imports use actual target `.gox` data; generic
imports retain source metadata with checked hashes. No original Go files are
rewritten. Private constants and non-generic type references use compiler aliases.
Aliases are normalized, and specialization identities retain struct tags and
the owning packages of anonymous private fields and methods. Build constraints
are omitted only from generated compiler inputs after source selection, avoiding
corrupt `SourcePos` output for separately grouped legacy/modern constraints.

Local constraints, inference, function values, generic struct methods and
recursive types have regression coverage. `test-cross-package.py` passes
assignment, interface assertions, aliases, struct tags, reflection names, private
methods, private ordinary structs/functions, and shared private variable state
across separately compiled packages, including optimized caller code. Embedded
generic pointer/value fields, interface method sets, field names/tags, private
field visibility and one generic type embedding another have host coverage.
Range over iterator functions remains. Upstream iter.Pull/Pull2 now execute
on the target through a scheduler coroutine adapter, described below.

Obtain and extract GCC 15.2.0 sources. Build prerequisites are the normal GCC
requirements, including a C++ compiler, make, and GMP/MPFR/MPC development files.
The helper uses the bundled zlib on a fresh build and does not install over the
system compiler:

```sh
python3 tooling/gccgo/build.py \
  --source /path/to/gcc-15.2.0 \
  --build /path/to/gccgo-build --jobs 4
export GCCGO_BUILD_DIR=/path/to/gccgo-build
python3 tooling/gccgo/build-compat.py --host-go /path/to/go
chmod +x tooling/gccgo/gccgo-kolibri
python3 tooling/gccgo/test.py --compiler "$PWD/tooling/gccgo/gccgo-kolibri"
python3 tooling/gccgo/test-patches.py --source /path/to/gcc-15.2.0
python3 tooling/gccgo/test-cross-package.py --compiler "$PWD/tooling/gccgo/gccgo-kolibri"
python3 tooling/gccgo/test-multidir.py --compiler "$PWD/tooling/gccgo/gccgo-kolibri"
make -C apps/opencode smoke GO="$PWD/tooling/gccgo/gccgo-kolibri"
./build-app.sh opencode
```

`gccgo-kolibri` uses the system `gccgo-15` driver with the patched frontend.
The locally saved, stripped frontend at
`.build-cache/gccgo15-kolibri/gcc/go1` is used by default when no
`GCCGO_BUILD_DIR` is set. It is a host executable, not a KolibriOS compiler.
The wrapper is intended for SDK builds with their explicit target flags;
hosted regression executables explicitly disable PIE.

The regression runner executes builtins and local generic tests on the host,
compiles all five fixtures to 386 objects, and checks 54 invalid programs fail with
diagnostics rather than internal compiler errors. Tests cover nil slices/maps,
backing tails, pointers, structs, zero-sized elements, NaN keys, deferred
clearing, side effects, typed indices, bound mutation, nested labeled loops,
captured addresses and closures, and reuse of an existing index variable.
`min`/`max`, local generics and unsafe operations also run with optimization enabled. Tests cover ordered
named types, rune/float constant promotion, large constants, result overflow,
string ties, evaluation order, NaN, infinities and both zero signs. Optional
`--reference-go /path/to/go` runs the same fixture with the upstream compiler.
The patch-stack test checks every already-applied prefix, so overlapping
patches work on fresh, partially patched and fully patched compiler sources.

## SDK changes

`stdlib/net/url/url.go` is copied unchanged from GCC 13.3.0 upstream libgo.
It supplies `ParseRequestURI`, decoded URL paths, and the other upstream URL
methods that the old bootstrap subset lacked. The previous bootstrap
`errors.As` hooks remain in `bootstrap_errors.go`. The upstream BSD license
is included alongside the code.

The package resolver now recognizes standalone named, dot, and blank imports,
and applies platform/build constraints to the entrypoint files before seeding
dependencies. Its focused tests are in `tooling/tests/test-resolve-packages.py`.

## Validation and remaining blockers

The initial target inventory selected 677 packages and 2728 Go files, with 20
missing packages. All twenty packages and their selected additional dependencies
have since been imported from upstream. The latest audit selects 719 packages
and 3037 Go files with no missing imports;
this is source discovery, not proof that all packages compile or run. See
[OPENCODE_STDLIB.md](OPENCODE_STDLIB.md) for per-package compilation and target
execution results, including the remaining compiler and runtime limitations.

The unchanged `github.com/rivo/uniseg` package now compiles. Its original host
suite passes against specialized compiler inputs, and KolibriOS QEMU checks
combining characters, a family emoji, string width and grapheme iterator methods.
The full CLI build has progressed through termenv, colorprofile, cellbuf and
lipgloss to Bubble Tea. Compiler fixes address private template
objects and imported generic function values, and generate unique import aliases
to avoid collisions with local declarations such as `hash`. Unused transitive
imports are blank imports that retain initialization without keeping arbitrary
function descriptors alive. The next missing APIs were `time.Ticker` and signal
identifiers; these have been added. Bubble Tea now compiles, and the full build
has advanced through cryptographic packages and `net/http` to OpenCode's
internal packages. Its pubsub and logging packages compile after patch 0007.
The next full-build blocker was the missing time.RFC1123Z layout used by cast;
original libgo time formatting and parsing have now been imported, compile
as a 386 object, and pass target execution. Afero now compiles after adding
upstream file APIs. The next failure was a locally defined type used as a
generic argument in locafero.Finder.Find. The specialization stage now lifts
ordinary function-local definitions into canonical compiler declarations,
retaining lexical aliases and the original checked type information. The
follow-up build exposed a nil signature in generic named type conversions;
both conversion calls and indexed function calls now retain the appropriate
inference metadata. Unchanged locafero compiles as a native 386 object in
`.build-cache/opencode/locafero-conversion-build.log`. A new full build is
checking the remaining dependency graph. The following failures were the
missing SDK IPMask.String and IPNet.String methods used by pflag. The portable
IP implementation now uses the complete unchanged libgo net/ip.go, with its
original parsing helpers, and pflag compiles. The next missing API was
reflect.Value.Comparable used by mapstructure; its unchanged Go 1.23 method
has been imported. Viper then required os.ExpandEnv and filepath.EvalSymlinks;
the corresponding unchanged libgo algorithms are imported, Viper compiles, and
the native Viper test passes. The unchanged OpenCode config package next
required os.UserHomeDir; Go 1.23 UserHomeDir, UserCacheDir and UserConfigDir
are now imported and pass the target fixture. The config package compiles;
the next full-build failure is Wazero's missing portable syscall errno values.
These now follow KolibriOS newlib, with explicit FD error translation and
unchanged Go Errno methods. The next missing constants, SHUT_RD/SHUT_WR/SHUT_RDWR,
are imported without inventing a native socket shutdown binding. Portable
os error APIs now use complete unchanged Go 1.23 sources. Wazero next required
Rmdir, Unlink, Rename, Stat_t, SetNonblock, SameFile and TCPConn.SyscallConn.
Native syscall 70 adapters implement file removal and rename with newlib errno
translation; the expanded filesystem fixture passes in KolibriOS
(`.build-cache/opencode/syscall-filesystem-native-kernel.log`). Stat_t is an
unchanged upstream portable declaration, not fabricated native metadata.
SetNonblock returns ENOTSUP because syscall 77 cannot change blocking flags.
SameFile compares absolute paths captured at Stat time, as allowed by the
portable Go API, but cannot distinguish replacements or track renames.
RawConn adapts the upstream API to native syscall 75 socket handles, protects
callbacks from concurrent Close, and polls with the connection deadlines. The
native raw socket fixture passes actual send/receive, polling, deadlines and
Close interruption (`.build-cache/opencode/rawconn-native-kernel.log`). The
upstream libgo KeepAlive function preserves compiler liveness. The SDK's
existing collector now also supports the finalizer queue described below.
A full build is checking subsequent packages. Native
symbolic links remain unsupported by the OS backend. Local types in
generic function bodies still need per-instantiation identities and are not
covered by this fix.
Wazero's unchanged sysfs and interpreter packages now compile. Its SSA package
exposed a further lowering error: private fields whose types are instantiated
generic named types bypassed the ownership rewrite. Field ownership is now
restored before that type-specific early return. The cross-package regression
passes ordinary and optimized host execution and rejects caller access to
private fields (`.build-cache/private-nested-after.log`). The unchanged SSA
package compiles as a native 386 object in
`.build-cache/opencode/github.com-tetratelabs-wazero-internal-engine-wazevo-ssa-fixed.o`.
This is not yet runtime validation of Wazero or SQLite: the prepared
`apps/tests/opencode/wazero-smoke` fixture still needs to build and execute.
The HTTP/TLS regression after the filesystem and RawConn changes passes in
KolibriOS (`.build-cache/opencode/rawconn-http-complete-native-kernel.log`).
The compiler stage now
retains existing GCC `__asm__` symbol annotations across generic lowering,
without changing native SDK source or symbols. A host regression verifies real
C calls through a generic function, ordinary and optimized builds, and a 386
object: `tooling/gccgo/test-nativeasm.py`.
An actual process stream backend and terminal support
still need work. Upstream `exec.Cmd.Output`/`CombinedOutput` are imported, while
the existing process backend explicitly rejects unsupported stream redirection.

The initial missing-package audit included `cmp`, `maps`, `slices`, `iter`, `log/slog`,
`math/rand/v2`, `runtime/debug`, `os/signal`, `flag`, `encoding/gob`,
`text/template`, `html/template`, `mime/multipart`, `net/textproto`,
`net/netip`, `net/rpc`, `net/http/httptrace`, `net/http/httputil`,
`net/http/httptest`, and `testing`.

The SDK already has HTTPS, TLS, X.509 chain/hostname verification, and PEM
root loading through `SSL_CERT_FILE` or explicit `RootCAs`, with a demo in
`apps/examples/https`. This support is reused by the upstream HTTP transport.
The buffered HTTP bootstrap has been replaced: a target test now reads an SSE
event incrementally from `Response.Body` before the peer completes the response.
Native listener and HTTPS shutdown tests require the kernel socket/TCP corrections documented in
[OPENCODE_STDLIB.md](OPENCODE_STDLIB.md). TLS certificate support is not missing.
The native crypto/rand reader now uses verified upstream OpenSSL CPU random
routines, with RDSEED/RDRAND support checked before execution. The clock-seeded
fallback is removed. Unsupported CPUs return an error; CPU random instructions
and functioning hardware/firmware are required until an OS entropy service
is available. Native byte/key/signature and unsupported-CPU checks pass as
documented in OPENCODE_STDLIB.md; earlier TLS tests alone did not prove this.

The initial qualification list included Bubble Tea terminal behavior,
filesystem notifications, process streams for shell/LSP/MCP tools, provider
HTTP API compatibility, and SQLite/Wazero. Their native evidence is recorded
above and in these bring-up notes; compilation alone never established them.
`//go:embed` now receives
compiler configuration derived from selected original sources. Go's own pattern
parsing and traversal algorithms resolve the original resource locations,
including quoted paths, directory patterns, hidden-file filtering and `all:`.
Host execution with generics passes for embed.FS, string and binary resources;
invalid patterns, symbolic links and module boundaries are rejected. Original
sources and resources remain unchanged. Native target execution passes in
`.build-cache/opencode/embed-native-kernel.log`, including embed.FS directory
traversal and exact text contents together with generics and ticker checks.

The compressed smoke artifact is
`apps/tests/opencode/smoke/opencode-smoke.kex`. It tests the unchanged upstream LSP
protocol's plain/escaped file URI handling and Position JSON round trip, plus
the three new compiler features against the KolibriOS runtime. QEMU screenshots
are stored in `.build-cache/opencode/smoke-12s.png` and neighboring captures.
The existing `apps/examples/url` also builds with the upstream URL replacement.

On 2026-10-05 a full package rebuild followed by QEMU execution reached
`OPENCODE_SMOKE_PASS` with integer/string constants, runtime integer values,
floating NaN and signed-zero checks. The automatic debug-port marker check
passed; `.build-cache/opencode/minmax-final-kernel.log` and the matching
12/24/36-second screenshots record the run. Rebuild dependent packages when
reusing an older smoke cache after SDK runtime/reflect changes; `make -B`
avoids mixing old and current package layouts.

On 2026-10-05 target checks reached `OPENCODE_SMOKE_PASS` for upstream context
cancellation causes, `WithoutCancel`, `AfterFunc`, and runtime `Goexit` running a
defer with nil recovery. The record is
`.build-cache/opencode/context-goexit-kernel.log`. The later unsafe/slog target
test reached `OPENCODE_SMOKE_PASS` in
`.build-cache/opencode/reflect-conversion-kernel.log`: actual string backing
storage, upstream slog JSON output and complete JSON object decoding passed.
The SDK's dummy `reflect.Value.Convert` was replaced with upstream conversions.
The unsafe host
fixture passes unoptimized, optimized and upstream Go execution, including runtime
error checks, nil pointers, integer overflow and single evaluation.

Make now tracks the actual frontend, generic helper and driver sources as Go
object prerequisites. Compiler changes rebuild both SDK and third-party package
exports, avoiding stale compiler bridges during incremental builds.

The expanded target check also reached `OPENCODE_SMOKE_PASS` in
`.build-cache/opencode/generic-identity-mapclone-kernel.log`. It verifies generic
assignment/assertions, private methods and package variable identity, original
reflection names, named JSON keys, Unicode/numeric conversions, independent
slice-to-array copies, and upstream map cloning including nil maps and NaN keys.

Dependency-specific KolibriOS files are registered in the shared SDK manifest
`third_party/kolibrios-compat/ADAPTERS.json`. Each file has one maintained origin
under `third_party/`. `tooling/stage-kolibrios-compat.py` applies them
alongside original vendored files for OpenCode or other applications. It checks
module versions and fingerprints and refuses to replace unrelated or modified
files. The native shell is independently reusable at `apps/tools/gosh/`.
`termenv` reuses its original portable defaults;
`go-isatty` recognizes only the SDK's active standard console descriptors.
Terminal raw input, resize handling and ANSI behavior still require target work.

The libgo Timer/Ticker port passed target execution in
`.build-cache/opencode/ticker-native-kernel.log`; see the stdlib document for
its scheduler adapter and remaining timer-runtime semantics. The compiler's
synthetic import positions now preserve declaration pragmas such as `go:embed`.
Host regression coverage is reproducible with
`python3 tooling/gccgo/test-embed.py --compiler tooling/gccgo/gccgo-kolibri`.

The later target run in
`.build-cache/opencode/reflect-slicing-native-kernel.log` also validates real
reflection slicing and channel sizes. See the stdlib document for the full
cases and remaining runtime scope. The selected closure currently contains
719 packages and 3037 Go files; original OpenCode files still match all 140
upstream hashes.

The unchanged Go 1.24 strings iterators also pass direct callback execution in
`.build-cache/opencode/strings-iterator-native-kernel.log`, including early
termination, Unicode splitting and line endings. OpenCode's range over these
functions is handled by the direct string-splitting intrinsic described above;
general function iterators use the upstream-derived rewrite described above.
Generated compiler inputs now retain
attached compiler pragmas while omitting ordinary comments, so canonical import
aliases cannot move inline comments into a selector and introduce a semicolon.
Original source files remain unchanged.

Target generic embedding coverage passes in
`.build-cache/opencode/generic-embedding-native-kernel.log`, including public
pointer/value fields, private fields, promoted interface methods and reflection.
Upstream iter.Pull/Pull2 also pass in
`.build-cache/opencode/iter-pull-native-kernel.log`: lazy startup, ordered values,
exhaustion, early/repeated stop, stop before next, deferred cleanup, panic and
Goexit propagation, and a thread-locked caller. The coroutine user function and
deferred exit follow Go 1.23 runtime/coro.go; native transitions reuse the SDK's
parked goroutines, context switching and stack reclamation. Thread locking
retains the SDK's existing semantics; nested lock counts and cgo are not tested.
The expanded run in `.build-cache/opencode/iter-pull-nested-native-kernel.log`
also passes nested iterators and panic propagation during stop.

Original libgo time formatting and parsing pass in
`.build-cache/opencode/time-format-native-kernel.log`: RFC1123Z, RFC3339/Nano,
RFC822, ANSIC, custom layouts, day-of-year, AppendFormat, numeric zones and
ParseInLocation, fractional nanoseconds, invalid leap days and duration bounds.
UTC/fixed-zone adapters preserve the SDK Time layout; timezone databases and
DST remain separate platform work. The follow-up HTTPS check passes in
`.build-cache/opencode/time-format-http-native-kernel.log`, including certificate
chain verification, response streaming and TLS server shutdown.

The next full build passed `spf13/cast` and stopped at Afero's use of
`os.ModeTemporary`. All FileMode constants now reuse the unchanged definitions
from libgo's os/types.go; this adds portable metadata definitions, without
claiming that KolibriOS implements every represented filesystem feature.

The filesystem target suite passes in
`.build-cache/opencode/filesystem-dot-utf8-native-kernel.log`. Upstream WriteAt,
Truncate, ReadDir and RemoveAll use native SDK adapters. It checks positional
writes without moving the sequential cursor, append/closed/read-only errors,
shrinking and zero-filled expansion beyond 16 MiB, timestamps and DOS read-only
metadata, sorted Unicode names, shared directory cursors and recursive removal.
Directory enumeration now excludes dot entries using the upstream rule and
requests UTF-8 names according to sysfuncs.txt. Ownership and links report
ENOSYS because the native filesystem interface does not expose those operations.

Automatic dependency resolution now writes the selected import graph for Make.
Changed package exports rebuild their users transitively; unrelated packages
retain their objects. Previous-package exports still enforce build order.
`tooling/tests/test-resolve-packages.py` verifies a real gccgo import/export
rebuild, unchanged unrelated objects, a no-change rebuild and target filtering.
The resource-change/addition/removal embed regression also passes with this
Make behavior.

Unchanged Go 1.23 filesystem traversal also passes in
`.build-cache/opencode/filesystem-walk-verified-native-kernel.log`: Walk visits
the full tree, WalkDir skips a selected subtree, SkipAll stops traversal, and a
missing root reaches the callback. The test accounts for the native FAT backend
returning short DOS names in upper case; long Unicode names retain their text.

Local type arguments pass native host execution with and without optimization,
cross-package specialization and rejection of out-of-scope names. The new
`tooling/gccgo/test-local-types.py` checks shadowed definitions, aliases,
recursive pointers, constant array bounds, anonymous fields, maps, channels
and reflection. Cross-package, embed and native symbol annotation regressions
also pass after this change. The corresponding KolibriOS application reaches
`OPENCODE_LOCAL_TYPES_PASS` in
`.build-cache/opencode/local-types-native-kernel.log`; native metadata restores
the original names and package paths for both local definitions and their
generic specializations. Hoisting changes only generated compiler input.
The gc compiler adds a local declaration ordinal to a generic argument's
reflection display (for example `Box[main.item·1]`). The current gccgo metadata
uses `Box[main.item]`; distinct types still have distinct canonical identities.
Exact gc display suffixes remain a compatibility detail to address.
The extended local-type fixture reproduces the nil-signature crash before the
fix and passes afterward, including generic mapper conversions, method calls
and nested callbacks with local named arguments. Default and optimized native
host execution, cross-package generic identity and GCC symbol annotations pass.
The extended fixture also reaches `OPENCODE_LOCAL_TYPES_PASS` in KolibriOS;
the evidence is `.build-cache/opencode/local-types-conversion-native-kernel.log`.

The original locafero Finder, Afero OS backend and sourcegraph/conc iterator pass
target execution in `.build-cache/opencode/original-finder-errors-native-kernel.log`.
This checks two real configuration search paths, exact names and glob matches,
missing candidates, empty input, concurrent mapper results and joined callback
errors. The first run exposed separate os.ErrNotExist and fs.ErrNotExist values;
the SDK now uses the unchanged upstream portable error aliases, so errors.Is
recognizes missing native files correctly. The checker requires the
`OPENCODE_FINDER_PASS` marker.

The portable IP source replacement also restores IPv6 parsing and canonical
formatting, IP text marshaling, network formatting, CIDR parsing, membership
and address classification. It retains the existing KolibriOS transport
backend; parsing IPv6 does not add native IPv6 socket support. The expanded
finder application passes these algorithms on the target in
`.build-cache/opencode/original-finder-full-ip-native-kernel.log`.

`apps/tests/opencode/reflect-smoke` checks the unchanged mapstructure decoder and
Go 1.23 Value.Comparable implementation. Its first native execution exposed
an existing SDK Value.Elem bug: nonempty interface method tables were being
read as empty-interface type descriptors, and unexported-field restrictions
were lost. The libgo interface conversion and flag propagation fix passes the
expanded checks, including error interfaces with slice dynamic values,
nil interfaces, nested arrays and private fields. Original Go runs the same
reflection and decoder assertions successfully. Native evidence is
`.build-cache/opencode/comparable-interface-fixed-native-kernel.log`.

The original libgo environment expansion handles braced variables, special
shell variable names and malformed input without source substitutions in
Viper. The original filepath walk resolves actual directory components and
reports missing paths or a non-directory component through Lstat. The expanded
reflection fixture passes the original Viper JSON decoder, environment
overrides and native file loading together with those portable algorithms in
`.build-cache/opencode/original-viper-path-env-native-kernel.log`.
The same assertions pass using the original Go compiler on the host.
The user-directory functions retain the upstream environment-variable behavior;
KolibriOS launch configuration still needs to supply HOME for a usable home
directory, rather than inventing a per-user account directory in the SDK.
Native evidence for defaults, environment overrides and missing-variable errors
is `.build-cache/opencode/original-user-directories-native-kernel.log`.

Typed error handling now uses the complete unchanged Go 1.23 errors package
and libgo internal/reflectlite. The old SDK only assigned error/interface{}
targets or delegated to package-specific As hooks, which failed a normal typed
provider-error extraction. That failure is reproduced before the change and
passes afterward on the host and KolibriOS. Native evidence is
`.build-cache/opencode/original-errors-reflectlite-native-kernel.log`.
The accompanying upstream unsafe pointer descriptors and native FD errno
translation pass the extended target checks in
`.build-cache/opencode/unsafe-descriptors-errno-native-kernel.log`.
Filesystem and HTTP/TLS tests pass after those changes in
`.build-cache/opencode/upstream-errors-filesystem-native-kernel.log` and
`.build-cache/opencode/upstream-errors-http-native-kernel.log`.
The complete portable OS error API also passes native aliases, errno
classification and historical helper behavior in
`.build-cache/opencode/original-os-error-api-native-kernel.log`.
The native filesystem regression still passes with the original PathError alias
in `.build-cache/opencode/original-os-error-filesystem-native-kernel.log`.

The IP and interface changes also pass the native HTTP/TLS regression again
in `.build-cache/opencode/full-ip-http-native-kernel.log`, including verified
HTTPS certificate chains and streaming response bodies.

The native runtime now records GCC's compiler-emitted type lists using the
upstream argument ABI and implements `reflect.lookupType`. Name normalization
follows libgo's runtime type-string rules, and pointer/composite descriptor
identity checks pass in
`.build-cache/opencode/typelists-composite-native-kernel.log`.

`runtime.SetFinalizer` and `runtime.GC` now use the SDK's existing collector.
Registration follows libgo's pointer/function validation; queued objects and
closures remain reachable until a separate goroutine executes the callbacks.
Dependency ordering, cancellation, pointer/empty-interface/nonempty-interface
arguments, ignored return values and resurrection pass in KolibriOS in
`.build-cache/opencode/finalizer-native-kernel.log`.
The callback ABI currently requires the reflect FFI backend to be linked, as
it is in OpenCode; finalizers run asynchronously on ordinary scheduler switches,
including time.Sleep, without an explicit Gosched call. Captured closure data,
KeepAlive and two OS threads pass in
`.build-cache/opencode/finalizer-threads-native-kernel.log`.
This does not validate weak-reference interning or complete the CLI port.

KolibriOS no longer receives the Unix build tag. File selection also follows
Go's rule that `_unix.go` is not an implicit OS suffix: its explicit build
constraint controls selection. This selects SQLite's upstream portable memory
and filesystem backends and removes its invalid x/sys/unix dependency.
The selector checks pass in `tooling/test-go-file-filter.py`, and selected-file
logic is now a compiler cache dependency. Wazero executes a WebAssembly function
and a reflected host callback in KolibriOS in
`.build-cache/opencode/wazero-nonunix-native-kernel.log`.

The original libgo PointerTo, SliceOf, ChanOf, MapOf and ArrayOf constructors
replace the incomplete SDK descriptors. Native maps and equality callbacks now
receive the gccgo function closure environment, preserving captured type data.
Comparable composite hashes follow libgo's semantic field traversal. The
upstream GC program decoder handles large and nested dynamic arrays. These
checks pass in `.build-cache/opencode/composite-gc-program-native-kernel.log`.
Typed failed assertions now use libgo's TypeAssertionError and panic machinery,
including nil values and missing interface methods, and can be recovered.
Their error messages pass in
`.build-cache/opencode/composite-interface-assertion-native-kernel.log`.
