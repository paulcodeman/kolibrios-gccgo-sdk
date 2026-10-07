# Upstream standard packages for OpenCode on KolibriOS

All 20 previously missing import paths now have upstream sources in `stdlib/`.
This is not a claim of complete API support: all 20 roots have compiled to 386
package objects with the experimental generic compiler stage. Runtime tracing
remains explicitly unavailable. Compilation specializes only
used generic bodies, so this is not validation of every generic API.

## Source fidelity

Old standard packages are copied from GCC 13.3.0 libgo. New packages (`cmp`,
`maps`, `slices`, `iter`, `log/slog`, `math/rand/v2`) use Go 1.23.0 sources, retaining
their real generic APIs rather than replacing them with restricted substitutes.
The selected dependency closure adds seven more upstream package roots.

`stdlib/OPENCODE_UPSTREAM.json` records versions and hashes of production Go
files. 179 files are byte-identical to upstream; six production files have
declared adaptations. Another 181 support source fingerprints are verified.

The complete unchanged Go 1.23 log/log.go now supplies the original
log/internal.DefaultOutput initialization hook required by slog. It replaces
the old custom logger and extracted termination methods. Native checks in
log-bridge-native-kernel.log cover logging before main, forward and reverse
log/slog output, caller formatting and synchronized concurrent records.
Full CLI startup also exposed the old 16,384-FDE scan limit. The native unwind
backend now adapts libgcc's indexed find_fde_tail search to the single flat
image; it keeps the bounded fallback for other header encodings.
The native goroutine stack size now follows libgo's non-split-stack StackMin:
2 MiB on i386. This fixes the original CLI's regexp initialization stack
fault; regexp-stack-native-kernel.log validates a 700-level nested regexp.
Original CLI version/help and fixed-provider pipeline checks now pass, and
original-cli-real-qwen-no-ping-native completes a real model request/response.
The unchanged original TUI now renders its colored dialog through native
libvterm and passes keyboard selection and confirmed exit. Unicode cells,
ANSI256 colors and terminal query responses also pass native checks; remaining
font glyphs, broader interactive usage and external-tool process support are
unfinished. Successful linking is not complete API compatibility.

Environment APIs now use complete unchanged libgo os/env.go and testlog/log.go,
with env_unix.go's original synchronized table behind a native loader backend.
Literal KEY=VALUE lines in `<executable path>.env` supply the initial environment
before package initialization. Native tests verify provider settings before
main, empty and duplicate entries, literal Unicode values, upstream expansion,
snapshot isolation and two-thread mutation. Evidence is
`.build-cache/opencode/environment-native-kernel.log`.

Go 1.23 function ranges use an upstream-derived compiler callback/state rewrite
and original runtime panic values. Native checks cover control flow, nested
valued returns and yield misuse in
`.build-cache/opencode/rangefunc-mixed-native-kernel.log`. General range-body
defer and labeled transfers still need enclosing-frame support and are rejected
explicitly; the string sequence slice intrinsic retains those operations.

`cli-link-native-kernel.log` verifies the libgo ifaceE2E2 assertion result,
128-bit map-key hashing, and Go 1.23 ChaCha8 golden output and saved-state
restoration. The native block entrypoint calls unchanged chacha8_generic.go;
math/rand/v2 uses the original bootstrap random algorithms adapted to a
mutex-protected native stream. Its time-derived seed is not a cryptographic
entropy source. Inactive runtime/trace task, region and log annotations link
and execute with tracing disabled. runtime/debug.SetTraceback explicitly
rejects the unsupported fatal all-goroutine capture instead of accepting it
silently.

The crypto/rand native reader now starts from Go 1.23's WASI reader boundary
and calls unchanged generated OpenSSL 3.0.15 RDSEED/RDRAND routines. GCC's
CPUID helpers check CPU support before executing either instruction. RDSEED
is preferred; RDRAND supplies any remaining bytes when available. The former
time/PID SplitMix64 fallback is removed. A CPU without both sources returns
an explicit error rather than generating predictable cryptographic data.
`secure-random-cpu-max-native` passes non-word-aligned byte reads and actual
Ed25519 key/signature generation and verification; `secure-random-unavailable-native`
passes the unsupported-CPU error path without modifying the requested buffer.
These checks validate integration and error handling, not statistical entropy
quality or FIPS certification. CPU random instructions and working hardware/
firmware are currently required; a general OS entropy service remains absent.
The runner defaults to QEMU `--cpu max` and accepts `--cpu qemu32` to exercise
the unsupported path. math/rand/v2's time-derived seed is a separate,
noncryptographic implementation.

The unchanged libgo `net/interface.go` and `net/mac.go` now supply interface
lookups, address enumeration and MAC parsing. The KolibriOS backend enumerates
all 16 native slots, including holes, and reads device names, Ethernet MACs,
IPv4 addresses and masks through syscalls 74/76. MTU is zero because the public
ABI has no query; multicast membership enumeration returns ENOTSUP. The native
fixture verifies real loopback `127.0.0.1/8`, lookups and MAC parsing in
`.build-cache/opencode/interface-native-kernel.log`.

SQLite also passes ordinary file opening with real exclusive locks, WAL
readers during a writer transaction, SQLITE_BUSY for a conflicting writer,
rollback and persistence after reopening in
`.build-cache/opencode/sqlite-dotlock-native-kernel.log`. Its original dot-lock
VFS and WAL index remain unchanged. A KolibriOS-only package selection manifest
selects that existing backend, and the small native dotlk adapter uses atomic
named-memory creation instead of unsupported O_EXCL files. This requires the
SDK-built kernel's SHM_CREATE error-return correction; the upstream kernel
otherwise overwrites E_ACCESS with the existing area's size. Cross-process
ownership also passes: another process receives SQLITE_BUSY while the owner
holds the private WAL index, then reads the committed rows after release.
The child exits with an open transaction and database; kernel cleanup releases
the locks, committed rows survive, the abandoned update is absent, and the
parent writes again. Evidence is
`.build-cache/opencode/sqlite-process-main-exit-native-kernel.log`.
The fixture also exposed and verifies a runtime fix: returning from main exits
the process immediately, as libgo runtime.main does, instead of draining parked
background goroutines and reporting a false deadlock.
The production adaptations are:

- `internal/intern/intern.go`: add `!kolibrios` selection. Its KolibriOS backend
  uses upstream's strong-reference interning algorithm because the bootstrap
  runtime's weak-reference interning has not yet been validated. Interned values
  remain retained until process exit.
- `os/signal/signal.go`: move the external `signalWaitUntilIdle` declaration into
  a file for upstream runtimes. The KolibriOS backend waits on a native console
  delivery barrier while Stop removes a subscriber.
- `net/http/request.go` and `transport.go`: retain SDK progress callback fields.
- `net/http/roundtrip.go`: attach progress callbacks through upstream HTTP traces.

Original tests and platform variants are retained; testdata directories were
not imported. The SDK does not yet run these upstream test suites. No OpenCode
source files are changed.

BSD licenses and copyright notices are included. Additional upstream extracts
and architecture files are listed under `support_sources` in the manifest.

```sh
python3 tooling/verify-stdlib-upstream.py
```

## Compilation status

| Packages | Current validation |
| --- | --- |
| `encoding/gob`, `flag`, `mime/multipart`, `net/netip`, `net/textproto`, `text/template` | 386 objects; target behavior tested by `stdlib-smoke` |
| `net/http/httptrace` | 386 object; context propagation and composed transport hooks included in target checks |
| `runtime/debug` | 386 object; build-info parsing and current-stack collection included in target test; GC controls and heap dumps need runtime exports |
| `os/signal` | Native Ctrl-C Notify/NotifyContext, two subscribers, Ignore/Reset, raw input and Stop tested; resize polling implemented but unverified; general POSIX signals unavailable |
| `html/template` | 386 object; original HTML autoescaping tested on the target |
| `cmp`, `iter`, `maps`, `slices`, `log/slog`, `math/rand/v2` | 386 package objects using generic specialization; upstream slog JSON, map cloning, generic identity/embedding, iter.Pull/Pull2, function ranges and random backend pass selected native checks |
| `net/http/httptest`, `net/http/httputil`, `net/rpc` | 386 objects against upstream HTTP and real reflective calls; selected target checks below |
| `testing` | 386 package object; disabled trace annotations link, but runtime tracing and fatal all-goroutine SetTraceback requests fail explicitly; upstream test-suite execution remains unverified |

Compile each root independently; failures do not hide later packages:

```sh
python3 tooling/stdlib-check.py
python3 tooling/stdlib-check.py flag mime/multipart text/template
```

The default compiler is the locally patched GCC Go frontend from
[OPENCODE_PORT.md](OPENCODE_PORT.md). Override it with `--compiler /path/to/gccgo`.
Logs and per-package results are in `.build-cache/opencode-stdlib/`. Source and
import/native-source/header fingerprints invalidate changed packages and their dependent objects
before the runner uses make's `FAST_PKG` mode. Package compilation does not
prove all runtime entrypoints are available at link time.

## Compatibility work in the existing SDK

- The complete libgo `net/ip.go` and its parsing helpers restore portable IP
  and network APIs required by pflag, including IPMask.String and IPNet.String.
  The existing native socket backend remains IPv4-only. The original
  internal/bytealg.Equal implementation supplies byte comparisons.
- Upstream portable os error aliases share fs error identities. Original
  locafero searches now recognize missing candidates with errors.Is; target
  execution also validates Afero file access and the original concurrent mapper.
- Upstream `io.SectionReader` and `io.NopCloser` implementations. The SDK's
  `io.CopyN` now uses the original libgo implementation: a short source returns
  `io.EOF`, as required by multipart parsing, instead of `io.ErrUnexpectedEOF`.
- Upstream `filepath.Match`/`Glob`, directory-name enumeration, and `Lstat` using
  KolibriOS's metadata operation. Syscall 70/5 has no separate link-following API.
- Upstream `os.CreateTemp` and `MkdirTemp` with native runtime randomness, plus
  original libgo `MkdirAll` including create-race handling. The original kernel
  cannot create exclusively: 70/2 rewrites existing files and 70/9 succeeds for
  existing directories. The isolated SDK kernel's `--exclusive-create` option
  adds atomic FAT operations 14/15 under the partition mutex, preserving upstream
  operations 11/12 for symbolic links and 13 for volume information. Existing entries
  return status 17 without modification. Older kernels and other filesystems
  return `os.ErrExclusiveCreateUnsupported` without touching the path. See
  [KERNEL_FILESYSTEM_EXTENSIONS.md](KERNEL_FILESYSTEM_EXTENSIONS.md).
- Upstream log panic/fatal functions, `net.IPAddr`, `runtime.Error`/`MemStats`
  declarations, and complete upstream 386 architecture constants. Adding the
  `ReadMemStats` now snapshots actual collector counters with the world stopped;
  heap allocation, live objects, thresholds and GC counts are available. Span,
  stack, pause and profile accounting remain zero and are not complete libgo
  memory statistics. Tracing and enabling block/mutex sampling explicitly fail.
- Upstream reflective channel receive, field traversal, method metadata,
  assignability, `Call`/`CallSlice`, bound method conversion and `MakeFunc`,
  adapted to the SDK gccgo layouts and runtime ABI. Calls and callbacks use
  original libffi 3.4.2 x86 Go closures. Interface conversion unwraps dynamic
  values and copies addressable values; nonempty assignment uses runtime itabs.
  General `Convert` and dynamically constructed types still need work.
  libgo's separate MakeFunc recovery bookkeeping is absent; callback
  panic/recovery behavior remains unverified.
- `runtime.ifaceefaceeq`, following libgo's interface equality algorithm and
  the SDK's existing type/equality helpers, including nil-interface semantics.
- Map insertion accepts named and unnamed map descriptors with identical key
  and element types, following libgo's use of the underlying layout. gccgo
  allocates named maps such as `MIMEHeader` but uses the underlying unnamed
  descriptor for some compiler helpers; descriptor pointer identity was an
  incorrect restriction in the SDK runtime.
- Current-stack PC collection through libgcc's DWARF unwinder. `debug.Stack`
  reports real program counters. Link-time symbol/DWARF tables and upstream
  frame traversal supply `runtime.Caller`, `CallersFrames` and `FuncForPC`.
  All-goroutine traces are unavailable: `runtime.Stack(buf, true)` fails explicitly. `ReadBuildInfo`
  returns `(nil, false)` because the SDK does not embed cmd/go module metadata.
- `os.Signal` source identifiers and a KolibriOS console event backend. Native
  Ctrl-C dispatch uses upstream handlers and Stop/NotifyContext algorithms;
  raw mode preserves Ctrl-C bytes. SIGWINCH polls actual console dimensions.
  General POSIX process signals and default Ctrl-C process termination are
  unavailable; native window close retains upstream immediate termination.

Syscall numbers and register layouts were not changed. Filesystem and network
constraints were checked against `sysfuncs.txt`.

## HTTP, TCP and scheduler compatibility

The HTTP bootstrap has been replaced with original libgo client, server and
transport sources, including HTTP/2 (HTTP/2 behavior remains untested).
`Response.Body` now streams: a target test reads an SSE event before the peer
releases the remainder. Existing TLS/X.509 verification and CA loading are reused.

Listeners use nonblocking accepts. TCP read/write retries check updated
deadlines. Port-zero listeners bind a free candidate through the kernel so
`Addr` reports the actual port without a getsockname ABI. Connect remains
blocking: pending dial timeouts/cancellation are incomplete. IPv6 transport,
half-close and configurable keepalive remain unavailable.

CPU name/features use upstream x86 CPUID. Time rounding uses upstream algorithms
adapted to the SDK fields. Bootstrap `sync` waiters now yield to Go's cooperative
scheduler, so goroutines releasing locks or completing WaitGroups can run.
Yielding only the OS task starved those goroutines and hung server shutdown.
`time.Sleep` now uses the runtime's goroutine parking/wakeup protocol and
monotonic deadlines, following libgo's sleep semantics. The bootstrap scheduler
scans a deadline list rather than libgo's per-P timer heaps. Sleeping in a timer
therefore leaves other goroutines runnable; independent timers have a target
regression check. Canceling a timer does not immediately remove its sleeping
goroutine, which remains retained until its original deadline.

Native accepts also exposed a kernel error in `network/socket.inc`:
`socket_accept` wrote error returns before restoring two saved registers.
This corrupted the return frame and appeared to accept invalid connections.
`tooling/build-kolibri-socket-kernel.py` corrects the three error paths in an
isolated copy of `kernel/trunk`; source checkout and syscall ABI are unchanged.
Two TCP bugs also appeared in certificate-verified HTTPS shutdown. The window
update calculation clamped after subtracting the advertised window, making an
unchanged 32 KiB window appear to grow by 32767 bytes. The builder clamps first,
following the [BSD TCP window algorithm](https://github.com/freebsd/freebsd-src/blob/main/sys/netinet/tcp_output.c).
The receive fast path also rejected in-order data in FIN_WAIT_1/FIN_WAIT_2,
routing it into the unimplemented reassembly path. It now accepts peer data
until the peer's FIN, including outstanding TLS close-notify records after a
local close. Rejecting these records caused endless loopback ACK traffic and
starved application tasks. The target network tests require all three fixes.
General TCP out-of-order reassembly remains unimplemented in this kernel;
these fixes do not establish production network reliability.

The same isolated kernel builder now fixes syscall 68/22's exclusive
named-memory creation error path. It returned an existing area's size in EDX
after setting E_ACCESS; branching to the error return preserves the documented
code 10. The original kernel checkout is unchanged. Native tests verify
exclusive creation, shared reads/writes from another OS thread and a separate
process, last-close removal and automatic last-mapping cleanup when the child
exits (`.build-cache/opencode/shared-memory-process-native-kernel.log`).

SQLite's unchanged driver, SQLite WASM module and Wazero interpreter have passed
in-memory SQL, rollback, UTF-8 text/BLOB persistence, GC, and file reopen in
KolibriOS (`.build-cache/opencode/sqlite-parking-native-kernel.log`). That run
explicitly used `nolock=1`; WAL and normal-path locking validation is separate.
The native compatibility adapter uses exclusive named memory to supply the
upstream dot-lock backend's process-ownership contract. Package-local
`kolibrios.sources.json` files select the original dot-lock VFS and its shared
WAL index, replacing only incompatible lock-file operations. Original vendor
files remain intact and other targets retain their normal source selection.

The runtime also supplies libgo's `printsp`, `printnl` and `printbool` semantics
through the existing debug writer, so compiler-generated `print`/`println`
calls no longer fail to link on spaces, newlines or boolean arguments.

```sh
python3 tooling/build-kolibri-socket-kernel.py --source ../kernel/trunk \
  --output .build-cache/opencode/kernel-network.mnt --fasm /path/to/fasm
/path/to/kerpack .build-cache/opencode/kernel-network.mnt
```

On Windows a native FASM `.exe` also works; run the Linux kernel packer in WSL.

## Target execution test

```sh
python3 tooling/stdlib-check.py --smoke
```

This links and compresses
`apps/tests/opencode/stdlib-smoke/opencode-stdlib-smoke.kex`. Run it on KolibriOS; a
successful run ends with `OPENCODE_STDLIB_PASS`. It checks:

- flag parsing and invalid values;
- gob struct/slice/map round trips;
- simple template iteration and reflected channel receives;
- netip prefixes, mapped IPv4 and comparable interned IPv6 zones;
- MIME header parsing and HTTP trace context propagation;
- multipart fields/files in memory and bounded section reading;
- short and exact `io.CopyN` transfers;
- historical rejection of unsupported exclusive creation, multipart disk spill
  and signal registration (those assertions predate the native filesystem and
  console event backends; use their separate current target checks);
- module-info parsing, absent embedded module info and real unwound stack PCs;
- mixed empty/non-empty interface equality and nil/type mismatch cases.
- shared entries, slice growth and deletion across named map conversions.

This test does not validate GC controls or the full OpenCode CLI. The separate
`apps/tests/opencode/http-smoke` test exercises reflective calls/callbacks and method
values, HTML escaping, original RPC over `net.Pipe`, CPU/source metadata, time
rounding, HTTP test helpers, incremental bodies, deadlines, actual loopback TCP,
composed HTTP trace hooks, and certificate-verified HTTPS/server shutdown.

```sh
make -C apps/tests/opencode/http-smoke GO="$PWD/tooling/gccgo/gccgo-kolibri" FAST_PKG=1 KPACK=1
python3 tooling/run-opencode-smoke.py --image /path/to/kolibri.img \
  --binary apps/tests/opencode/http-smoke/opencode-http-smoke.kex \
  --kernel .build-cache/opencode/kernel-network.mnt --prefix http-smoke
```

The runner uses temporary floppy/FAT disk images and captures screenshots.
Capture completion alone is not a pass: inspect `OPENCODE_HTTP_PASS` or
`OPENCODE_HTTP_FAIL`. These focused tests do not replace upstream test suites.
For an automated marker check, build the kernel with `--debug-output` and pass
`--expect-marker OPENCODE_HTTP_PASS` to the runner. This sends debug messages
to QEMU port 0xe9 without enabling verbose network logging; a missing marker
makes the runner fail.

On 2026-10-05 the final test executable ran in KolibriOS under QEMU with 128 MiB
RAM and reached `OPENCODE_STDLIB_PASS`. Screenshots are retained at
`.build-cache/opencode/stdlib-final-12s.png`, `stdlib-final-24s.png` and
`stdlib-final-36s.png`. QEMU used a temporary copy of the boot image; the original
image was not modified. These execution checks cover the cases listed above,
not the complete APIs or upstream test suites.

The final HTTP executable also reached `OPENCODE_HTTP_PASS` with the corrected
kernel and without temporary transport diagnostics or verbose kernel network
logging. `.build-cache/opencode/http-final-network-12s.png` records this run.
An independent run with debug-port output passed the automatic marker check;
its log is `.build-cache/opencode/http-final-verified-kernel.log`. Both runs
verified the TLS certificate chain, response body and server shutdown.

The SDK now uses the byte-identical Go 1.23 `context` implementation. Its
cancellation cause propagation, `WithoutCancel`, and `AfterFunc` passed in QEMU.
`runtime.Goexit` adapts upstream libgo defer/panic bookkeeping to the KolibriOS
scheduler; target tests confirm that defer runs and recover returns nil. Upstream
`atomic.Int32`, `atomic.Pointer[T]`, duration unit methods and unsafe string
runtime checks have source records in the manifest. Atomic primitives now use
libgo's sequentially consistent GCC builtins, with original `atomic.Value` and
Go 1.23 typed numeric values and And/Or methods. A native two-thread test verifies
40,000 increments for both Int32 and Int64, Value/Pointer operations, typed
alignment in embedded fields and slices, and recovery from an unaligned 64-bit
operation. Its log is `.build-cache/opencode/atomic-heap-alignment-native-kernel.log`.
The Value initialization pinning adapters rely on the cooperative scheduler;
this does not establish support for asynchronous Go preemption. Other `sync`
primitives now use nine unchanged Go 1.23 files for Mutex/RWMutex, Once and its
generic helpers, WaitGroup, Cond, Map and runtime declarations. Native semaphore
queues park goroutines, recheck under a lock to prevent missed releases, honor
FIFO/LIFO waiting and starvation handoff, and wake condition tickets through the
scheduler. The target two-thread test checks contended locks, read/write
exclusion, condition broadcast, concurrent Map/CAS/Clear and repeated OnceFunc
panics (`.build-cache/opencode/sync-scheduler-native-kernel.log`). Pool remains
the earlier SDK backend and still needs its own upstream compatibility work.

`math/rand` now declares its original package name `rand`, allowing ordinary
upstream imports and restoring byte-identical HTTP server source. `os.File.Fd`
returns actual SDK console/pipe descriptors, and an invalid descriptor for path
files, which KolibriOS does not open through OS handles. General descriptor file
I/O compatibility and the process stream backend still require implementation.

Upstream `exec.Cmd.Output` and `CombinedOutput` wrappers are now present; their
process stream backend remains unsupported and is rejected explicitly.
Upstream-derived reflection conversions and the native map-clone table backend
pass the expanded smoke test, including named JSON keys, copied array values,
nil maps and multiple non-reflexive NaN keys.

Original libgo `Ticker`/`Tick` and the public `Timer`/`AfterFunc` algorithms now
run on a shared KolibriOS scheduler adapter. The target record
`.build-cache/opencode/ticker-native-kernel.log` reaches `OPENCODE_SMOKE_PASS`:
periodic ticks, slow-reader dropping, Stop without closing the channel, restart
through Reset and non-positive periods all pass, alongside context callbacks
and Goexit. The adapter uses the existing cooperative runtime sleep primitive
and cancellation generations. It retains libgo's buffered timer channels;
Go 1.23 synchronous timer-channel hooks and GC recovery are not implemented.
Canceled sleeping workers return when their scheduled deadline is reached.

`syscall.Signal` and its identifiers/methods now reuse Go's Windows source
compatibility definitions. `os.Interrupt` and `os.Kill` refer to these same
values. This does not add POSIX signal delivery to KolibriOS.

Original libgo reflection slicing and floating overflow checks now replace
dummy methods. Array/slice/string slicing, shared backing storage, three-index
capacity, private-field restrictions, invalid indexes/kinds and channel sizes
pass in `.build-cache/opencode/reflect-slicing-native-kernel.log`. Native channel
size entrypoints follow libgo's qcount/dataqsiz accessors.

`strings/iter.go` is byte-identical to Go 1.24.0. It provides SplitSeq,
SplitAfterSeq, Lines, FieldsSeq and FieldsFuncSeq required by newer source.
Direct iterator calls pass on KolibriOS in
`.build-cache/opencode/strings-iterator-native-kernel.log`: SplitSeq stops when
the callback returns false, splitting an empty separator preserves Unicode
characters, and Lines retains newlines and the final unterminated line.
These tests invoke callbacks directly; range over functions remains a compiler
requirement.

Original iter.Pull/Pull2 now pass in
`.build-cache/opencode/iter-pull-native-kernel.log`, including lazy startup,
values/pairs, exhaustion, early/repeated stop, stop before next, deferred cleanup,
panic/Goexit propagation and a thread-locked caller. Go 1.23 runtime/coro.go's
user-function/deferred-exit flow uses the native SDK parked goroutine exchange
and ordinary stack reclamation. No iter source changes were required. Thread
locking retains the SDK's current behavior; nested lock counts and cgo remain
outside this verification.
The expanded target run `.build-cache/opencode/iter-pull-nested-native-kernel.log`
also passes nested pulls and panic propagation during stop.

The original libgo format.go layout scanner, formatting, parsing and duration
parser now compile on 386, replacing the SDK's limited layout switches.
Date/clock/zone and monotonic field access use SDK adapters; UTC and fixed
locations remain supported, while timezone databases and DST are not added.
DateTime, DateOnly and TimeOnly reuse Go 1.23 constants. Target checks pass in
`.build-cache/opencode/time-format-native-kernel.log`: RFC1123Z, RFC3339/Nano,
RFC822, ANSIC, custom layouts and day-of-year, AppendFormat, ParseInLocation,
numeric zone offsets, fractional nanoseconds, leap-day rejection and duration
negative/positive overflow bounds. This does not establish complete time.Time
compatibility: the SDK representation and remaining calendar/zone APIs still
need separate coverage.
The HTTP target suite also passes after the time changes in
`.build-cache/opencode/time-format-http-native-kernel.log`, including TLS
certificate chain verification and server shutdown.

`os/filemode.go` extracts all original libgo FileMode constants, reusing the
existing io/fs definitions. These portable metadata bits do not establish
native support for symlinks, pipes, devices or Unix permission enforcement.

Upstream `io.WriterAt`, `os.File.WriteAt`, `Truncate`, `ReadDir` and `RemoveAll`
now use the existing native filesystem operations. The target record
`.build-cache/opencode/filesystem-dot-utf8-native-kernel.log` passes positional
write/cursor checks, shrink/zero-filled expansion including the 16 MiB boundary,
append/closed/read-only-handle errors, Chtimes and DOS read-only metadata,
sorted Unicode names, mixed directory cursors, EOF and recursive deletion.
The existing enumeration backend now filters `.`/`..` before counting public
entries, matching libgo, and requests UTF-8 names using the documented ABI.
Expansion uses steps at most 16 MiB so syscall 70/4 guarantees zero filling.
Chmod maps the owner write bit to the DOS read-only attribute, rather than Unix
permissions; timestamps retain FAT precision and its supported year range.
Ownership and links return ENOSYS. This does not establish full os.File or
process-stream compatibility.

Go 1.23 io/fs/walk.go is unchanged, adding SkipAll and current traversal
semantics. The Walk/WalkDir region from path/filepath/path.go is also unchanged.
Target checks pass in `.build-cache/opencode/filesystem-walk-verified-native-kernel.log`
for complete Walk traversal, SkipDir subtrees, SkipAll early termination and
missing-root callbacks, alongside the preceding file and directory checks.
The native FAT backend returns short DOS names in upper case; Unicode long
filenames retain their original text.

The complete unchanged libgo net/ip.go and its portable parsing helpers pass
native IPv4/IPv6 text, CIDR, network membership and classification checks in
`.build-cache/opencode/original-finder-full-ip-native-kernel.log`. This does
not add IPv6 sockets to the KolibriOS transport backend.

Go 1.23 reflect.Value.Comparable is imported with its method unchanged.
The SDK Value.Elem now uses libgo's nonempty-interface conversion and carries
the original unexported-field flags. Native tests include dynamic error
interfaces, slices, arrays, structs and the unchanged mapstructure decoder in
`.build-cache/opencode/comparable-interface-fixed-native-kernel.log`.

The complete Go 1.23 errors package and GCC 13.3.0 internal/reflectlite replace
the bootstrap's limited typed-error handling. All seven production files are
byte-identical to upstream. Native tests pass for ordinary typed provider
errors, nonempty interface assignment, joined error trees, uncomparable error
values, custom As hooks, invalid targets and sorting pointer-containing structs
in `.build-cache/opencode/original-errors-reflectlite-native-kernel.log`.

The libgo go-unsafe-pointer.c descriptors are ported in
`platform/abi/runtime_unsafe_pointer.h`, using the existing native type and
equality layouts. Both unsafe.Pointer and *unsafe.Pointer now have the original
kind, hashes, reflection strings, equality descriptor and GC bitmaps. The
extended test passes reflection/equality checks and actual native descriptor
error calls in `.build-cache/opencode/unsafe-descriptors-errno-native-kernel.log`.

Portable syscall errno values follow KolibriOS newlib. Errno.Is, Temporary and
Timeout use unchanged Go 1.23 methods. The native FD ABI returns -11 for EINVAL,
as documented in sysfuncs.txt; Read, Write and Pipe2 translate that to the Go
compatibility EINVAL (22), distinct from EAGAIN (11). Other documented FD errors
retain their numeric values. Raw bindings, socket error numbers and syscall 70
status codes are unchanged. The native test verifies invalid pipe flags and
invalid file descriptors through real kernel calls.

The native filesystem and HTTP/TLS regressions pass after the errors and errno
changes in `.build-cache/opencode/upstream-errors-filesystem-native-kernel.log`
and `.build-cache/opencode/upstream-errors-http-native-kernel.log`.

Complete Go 1.23 os/error.go and error_errno.go now replace the separate SDK
portable error fragments. PathError is the original fs.PathError alias, and
IsNotExist/IsExist/IsPermission/IsTimeout use the original historical unwrapping
rules and syscall.Errno classification. The extracted upstream poll.ErrNoDeadline
declaration supplies the shared sentinel; this does not implement file deadlines.
The target fixture passes aliases and classifications in
`.build-cache/opencode/original-os-error-api-native-kernel.log`; native
filesystem regressions pass in
`.build-cache/opencode/original-os-error-filesystem-native-kernel.log`.
The SHUT_RD/SHUT_WR/SHUT_RDWR compatibility constants
are also imported; they do not add a native socket half-close operation.
