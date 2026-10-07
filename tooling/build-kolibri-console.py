#!/usr/bin/env python3
"""Build an isolated upstream CONSOLE.OBJ with an SDK terminal-size query."""

import argparse
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True, help='programs/develop/libraries/console_coff directory')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--fasm', default='fasm')
    parser.add_argument('--signal-events', action='store_true',
                        help='capture or ignore canonical Ctrl-C and preserve raw Ctrl-C input')
    parser.add_argument('--utf8-output', action='store_true',
                        help='add opt-in streamed UTF-8 output for the existing CP866 font')
    parser.add_argument('--vterm', action='store_true',
                        help='add upstream libvterm Unicode and ANSI256 backend (requires --utf8-output)')
    args = parser.parse_args()
    if args.vterm and not args.utf8_output:
        parser.error('--vterm requires --utf8-output')
    output = args.output.resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='kolibri-console-') as temporary:
        programs = Path(temporary)
        library = programs / 'develop/libraries/console_coff'
        shutil.copytree(args.source, library)
        shutil.copyfile(args.source.resolve().parents[2] / 'struct.inc', programs / 'struct.inc')
        source = library / 'console.asm'
        data = source.read_bytes()
        replacements = {
            b'con_get_flags:\n': b'''con_get_size:
        mov     ecx, [esp+4]
        test    ecx, ecx
        jz      .height
        mov     eax, [con.wnd_width]
        mov     [ecx], eax
  .height:
        mov     ecx, [esp+8]
        test    ecx, ecx
        jz      .done
        mov     eax, [con.wnd_height]
        mov     [ecx], eax
  .done:
        ret     8

con_get_flags:
''',
            b'        dd      szcon_get_flags,        con_get_flags\n':
                b'        dd      szcon_get_flags,        con_get_flags\n'
                b'        dd      szcon_get_size,         con_get_size\n',
            b"szcon_get_flags         db 'con_get_flags',0\n":
                b"szcon_get_flags         db 'con_get_flags',0\n"
                b"szcon_get_size          db 'con_get_size',0\n",
        }
        # Normalize line endings only in the isolated assembly copy.
        data = data.replace(b'\r\n', b'\n')
        for old, new in replacements.items():
            if data.count(old) != 1:
                raise RuntimeError('unexpected upstream console source: ' + old.decode('ascii').strip())
            data = data.replace(old, new)
        # Fix existing VT operations in the isolated upstream copy. Cursor-left
        # previously used the final command byte rather than the CSI argument.
        # An unrecognized OSC also returned while its pushad frame was live.
        replacements = {
            b'.cursor_left:\n        test    eax, eax\n':
                b'.cursor_left:\n        mov     eax, [con_esc_attrs]\n        test    eax, eax\n',
            b"        cmp     al, 'H'\n        je      .cursor_position\n":
                b"        cmp     al, 'G'\n        je      .column_position_abs\n"
                b"        cmp     al, 'H'\n        je      .cursor_position\n",
            b'.line_position_abs:\n': b'''.column_position_abs:
        mov     eax, [con_esc_attrs]
        dec     eax
        jns     @f
        xor     eax, eax
       @@:
        cmp     eax, [con.scr_width]
        jb      @f
        mov     eax, [con.scr_width]
        dec     eax
       @@:
        mov     [con.cur_x], eax
        jmp     con.get_data_ptr
.line_position_abs:
''',
            b"        cmp     [con_esc_attrs+0], 2            ; Set Window Title\n"
            b'        je      .set_title\n        ret\n':
                b"        cmp     [con_esc_attrs+0], 2            ; Set Window Title\n"
                b'        je      .set_title\n        popa\n        ret\n',
        }
        for old, new in replacements.items():
            if data.count(old) != 1:
                raise RuntimeError('unexpected upstream console VT source: ' + old.decode('ascii').strip())
            data = data.replace(old, new)
        if args.signal_events:
            replacements = {
                b'con_get_flags:\n': b'''con_set_input_mode:
        mov     eax, [esp+4]
        and     eax, 1
        xchg    eax, [con.sdk_input_mode]
        ret     4

con_set_signal_mode:
        mov     eax, [esp+4]
        cmp     eax, 2
        ja      .invalid
        xchg    eax, [con.sdk_signal_mode]
        ret     4
  .invalid:
        mov     eax, [con.sdk_signal_mode]
        ret     4

con_take_interrupts:
        xor     eax, eax
        xchg    eax, [con.sdk_interrupts]
        ret

con_get_flags:
''',
                b'        dd      szcon_get_size,         con_get_size\n':
                    b'        dd      szcon_get_size,         con_get_size\n'
                    b'        dd      szcon_set_input_mode,   con_set_input_mode\n'
                    b'        dd      szcon_set_signal_mode,  con_set_signal_mode\n'
                    b'        dd      szcon_take_interrupts,  con_take_interrupts\n',
                b"szcon_get_size          db 'con_get_size',0\n":
                    b"szcon_get_size          db 'con_get_size',0\n"
                    b"szcon_set_input_mode    db 'con_set_input_mode',0\n"
                    b"szcon_set_signal_mode   db 'con_set_signal_mode',0\n"
                    b"szcon_take_interrupts   db 'con_take_interrupts',0\n",
                b'con_flags       dd      0x07':
                    b'con.sdk_input_mode dd 0\ncon.sdk_signal_mode dd 0\ncon.sdk_interrupts dd 0\n\n'
                    b'con_flags       dd      0x07',
                b'; dx contains full keycode\n        cmp     [con.bGetchRequested], 0\n':
                    b'''; dx contains full keycode
        cmp     dl, 3
        jne     .sdk_normal_key
        cmp     [con.sdk_input_mode], 0
        jne     .sdk_normal_key
        cmp     [con.sdk_signal_mode], 1
        jne     .sdk_not_captured
        lock inc dword [con.sdk_interrupts]
        jmp     con.msg_loop
  .sdk_not_captured:
        cmp     [con.sdk_signal_mode], 2
        je      con.msg_loop
  .sdk_normal_key:
        cmp     [con.bGetchRequested], 0
''',
            }
            for old, new in replacements.items():
                if data.count(old) != 1:
                    raise RuntimeError('unexpected console control-event source: ' + old.decode('ascii').strip())
                data = data.replace(old, new)
        if args.utf8_output:
            replacements = {
                b'con_get_flags:\n': b"include 'utf8-output.inc'\n\ncon_get_flags:\n",
                b'        dd      szcon_get_flags,        con_get_flags\n':
                    b'        dd      szcon_get_flags,        con_get_flags\n'
                    b'        dd      szcon_set_output_mode,  con_set_output_mode\n',
                b'  @@:\n        call    con.write_char_ex\n  .next:\n':
                    b'  @@:\n        call    con.sdk_write_utf8\n  .next:\n',
            }
            for old, new in replacements.items():
                if data.count(old) != 1:
                    raise RuntimeError('unexpected console UTF-8 source: ' + old.decode('ascii').strip())
                data = data.replace(old, new)
            support = Path(__file__).resolve().parent / 'console'
            for name in ('utf8-output.inc', 'utf8-decode.inc', 'cp866-unicode.inc'):
                shutil.copyfile(support / name, library / name)
        if args.vterm:
            from build_console_vterm import adapt_source, link_backend
            data = adapt_source(data, library)
        source.write_bytes(data)
        assembled = programs / 'console-fasm.obj' if args.vterm else output
        subprocess.run([args.fasm, str(source), str(assembled)], cwd=library, check=True)
        if args.vterm:
            link_backend(assembled, output, programs)
    print('Built upstream console with native con_get_size:', output)


if __name__ == '__main__':
    main()
