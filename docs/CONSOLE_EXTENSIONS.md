# Native console compatibility

`tooling/build-kolibri-console.py` builds a temporary copy of the upstream
KolibriOS `programs/develop/libraries/console_coff` library. Existing exports and
console behavior remain intact. The checkout is not modified.

The SDK adds `void __stdcall con_get_size(uint32_t *columns, uint32_t *rows)`.
It copies the actual `con.wnd_width` and `con.wnd_height` values to the supplied
non-null pointers. These are character dimensions of the visible console,
including changes maintained by the native library's resize implementation.
A null pointer skips that output. The caller supplies valid writable pointers;
the function consumes eight bytes of stack arguments and changes only caller
saved EAX and ECX. It performs no syscall or allocation.

The export allows terminal adapters to report real dimensions instead of
hardcoding the library defaults. Existing libraries without the export remain
usable for ordinary console output; size queries must report unavailable.

```sh
python3 tooling/build-kolibri-console.py \
  --source /path/to/kolibrios/programs/develop/libraries/console_coff \
  --output .build-cache/opencode/console.obj --fasm /path/to/fasm
```

The smoke runner accepts `--console-object` to replace the library only in its
temporary floppy image. The user's original image is not changed.

With `--signal-events`, the isolated build also exports these native control
functions. All use stdcall, return a uint32 in EAX, preserve callee-saved
registers, and start with zero state:

| Export | Meaning |
| --- | --- |
| `con_set_input_mode(uint32_t raw)` | Stores `raw & 1`, returns the previous mode, and removes four argument bytes. |
| `con_set_signal_mode(uint32_t policy)` | Stores policy 0 (ordinary input), 1 (capture), or 2 (ignore), and returns the previous policy. Values above 2 leave it unchanged. Removes four argument bytes. |
| `con_take_interrupts(void)` | Atomically returns the captured Ctrl-C count and resets it to zero. |

The original native keyboard handler captures ASCII 3 only in canonical mode
with policy 1. It increments the counter and omits that byte from the input
queue. Policy 2 drops canonical Ctrl-C without an event. Raw mode always
preserves Ctrl-C as an input byte, allowing the original CLI key parser to
handle it. Policy 0 retains upstream behavior. These exports add no kernel
syscalls and do not implement general POSIX process signals.

`console-control-native-kernel.log` verifies actual injected Ctrl-C capture,
raw input preservation, and ignored canonical input in KolibriOS. This native
library check alone does not verify `os/signal.Notify` or the full CLI.
