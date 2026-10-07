"""Isolated libvterm backend for the upstream KolibriOS console COFF DLL."""
from pathlib import Path
import shutil
import struct
import subprocess
import hashlib
import json


def convert_assembly_object(source, target):
    subprocess.run(['objcopy', '-O', 'elf32-i386', str(source), str(target)], check=True)
    data = bytearray(target.read_bytes())
    headers = struct.unpack_from('<I', data, 32)[0]
    stride, count = struct.unpack_from('<HH', data, 46)
    for i in range(count):
        header = headers + i*stride
        kind = struct.unpack_from('<I', data, header+4)[0]
        if kind != 9:  # SHT_REL
            continue
        offset, length = struct.unpack_from('<II', data, header+16)
        target_index = struct.unpack_from('<I', data, header+28)[0]
        target_header = headers + target_index*stride
        target_raw = struct.unpack_from('<I', data, target_header+16)[0]
        for entry in range(offset, offset+length, 8):
            address, info = struct.unpack_from('<II', data, entry)
            # BFD translates the numbers, but retains COFF's stored addend.
            if info&255 not in (1,2):
                raise RuntimeError('unsupported translated assembly relocation: %d' % (info&255))
            if info&255 == 2:
                value = struct.unpack_from('<I', data, target_raw+address)[0]
                struct.pack_into('<I', data, target_raw+address, (value-4)&0xffffffff)
    target.write_bytes(data)


def adapt_source(data, library):
    replacements = {
        b'con_get_flags:\n': b"include 'vterm.inc'\n\ncon_get_flags:\n",
        b'        dd      szcon_get_flags,        con_get_flags\n':
            b'        dd      szcon_get_flags,        con_get_flags\n'
            b'        dd      szcon_enable_vterm,     con_enable_vterm\n'
            b'        dd      szcon_set_unicode_font, con_set_unicode_font\n'
            b'        dd      szcon_unicode_glyph_width, con_unicode_glyph_width\n'
            b'        dd      szcon_get_cell,         con_get_cell\n',
        b'con.write:\n': b'''con.write:
        cmp [con.sdk_utf8_mode], 2
        je con.sdk_vterm_write
''',
        b'con.data2image:\n': b'''con.data2image:
        cmp [con.sdk_utf8_mode], 2
        je con.sdk_vterm_render
''',
        b'con_get_cursor_pos:\n': b'''con_get_cursor_pos:
        cmp [con.sdk_utf8_mode], 2
        je con.sdk_vterm_cursor
''',
        b'con_kbhit:\n': b'''con_kbhit:
        cmp [con.sdk_utf8_mode], 2
        jne .legacy
        call sdk_vterm_output_pending
        test eax, eax
        jz .legacy
        mov eax, 1
        ret
  .legacy:
''',
        b'con.mouse:\n': b'''con.mouse:
        cmp [con.sdk_utf8_mode], 2
        jne .sdk_legacy_mouse
        call sdk_vterm_mouse_active
        test eax, eax
        jnz con.sdk_vterm_mouse
  .sdk_legacy_mouse:
''',
        b'con_get_input:\n': b'''con_get_input:
        cmp [con.sdk_utf8_mode], 2
        jne .legacy
        call sdk_vterm_output_pending
        test eax, eax
        jnz con.sdk_vterm_response
  .legacy:
''',
        b'.skip_vscroll:\n        ret\n': b'''.skip_vscroll:
; The native window retains its initial scrollbar width when libvterm
; switches to an alternate screen without history. Paint the unused strip
; rather than leaving old pixels or the desktop visible (sysfuncs.txt 13).
        cmp [con.sdk_utf8_mode], 2
        jne .sdk_scrollbar_done
        mov eax, [con.scr_height]
        cmp eax, [con.wnd_height]
        ja .sdk_scrollbar_done
        pushad
        mov eax, 9
        mov ebx, process_info_buffer
        mov ecx, -1
        int 0x40
        mov eax, [process_info_buffer.client_box.width]
        movzx ebx, [con.data_width]
        sub eax, ebx
        jle .sdk_scrollbar_restore
        cmp eax, con.vscroll_width
        jbe .sdk_scrollbar_width
        mov eax, con.vscroll_width
  .sdk_scrollbar_width:
        movzx ebx, [con.data_width]
        shl ebx, 16
        or ebx, eax
        movzx ecx, [con.data_height]
        cmp ecx, [process_info_buffer.client_box.height]
        jbe .sdk_scrollbar_height
        mov ecx, [process_info_buffer.client_box.height]
  .sdk_scrollbar_height:
        test ecx, ecx
        jz .sdk_scrollbar_restore
        mov edx, [con.colors]
        mov eax, 13
        int 0x40
  .sdk_scrollbar_restore:
        popad
  .sdk_scrollbar_done:
        ret
''',
        b'\nscan_has_ascii:\n': b'\n        dd 240 dup (0)\n\nscan_has_ascii:\n',
        b'; regular ASCII\n        mov     byte[ebx], al\n        mov     eax, 1\n':
            b'''; regular ASCII / native CP866 keyboard
; Match upstream libvterm's BACKSPACE == ASCII DEL in terminal mode.
; Check the physical scan code so Ctrl-H remains ASCII BS.
        cmp [con.sdk_utf8_mode], 2
        jne .sdk_backspace_done
        cmp ah, 0x0e
        jne .sdk_backspace_done
        cmp al, 8
        jne .sdk_backspace_done
        mov al, 127
  .sdk_backspace_done:
        cmp [con.sdk_utf8_mode], 0
        je .sdk_legacy_byte
        movzx eax, al
        push ebx
        push eax
        call sdk_vterm_encode_cp866
        add esp, 8
        jmp .got_input
  .sdk_legacy_byte:
        mov byte[ebx], al
        mov eax, 1
''',
    }
    for old, new in replacements.items():
        if data.count(old) != 1:
            raise RuntimeError('unexpected console vterm source: ' + repr(old))
        data = data.replace(old, new)
    shutil.copyfile(Path(__file__).parent / 'console/vterm.inc', library / 'vterm.inc')
    return data


def link_backend(console, output, temporary):
    root = Path(__file__).resolve().parent.parent
    for directory in ('libvterm','kolibri-printf','kolibri-newlib-string'):
        origin=root/'third_party'/directory
        manifest=json.loads((origin/'UPSTREAM.json').read_text())
        for name, expected in manifest['files'].items():
            if hashlib.sha256((origin/name).read_bytes()).hexdigest()!=expected:
                raise RuntimeError('upstream console source changed: '+directory+'/'+name)
    upstream = root / 'third_party/libvterm/libvterm-0.3.3'
    support = root / 'tooling/console'
    formatter = root / 'third_party/kolibri-printf'
    strings = root / 'third_party/kolibri-newlib-string'
    flags = ['-m32', '-Os', '-std=c99', '-ffunction-sections', '-fdata-sections',
             '-U_FORTIFY_SOURCE', '-D_FORTIFY_SOURCE=0',
             '-fno-pic', '-fno-pie', '-fno-builtin', '-fno-stack-protector',
             '-fno-unwind-tables', '-fno-asynchronous-unwind-tables',
             '-fno-tree-loop-distribute-patterns', '-mpreferred-stack-boundary=2',
             '-mincoming-stack-boundary=2', '-DKOLIBRIOS',
             '-Dmalloc=sdk_console_alloc', '-Dfree=sdk_console_free',
             '-Dfprintf=sdk_console_fprintf',
             '-Dexit=sdk_console_exit', '-Dabort=sdk_console_abort',
             '-include', str(support/'vterm-native-stdio.h'),
             '-I'+str(upstream / 'include'), '-I'+str(upstream / 'src'), '-I'+str(support)]
    objects = []
    sources = list(sorted((upstream / 'src').glob('*.c'))) + [support / 'vterm_backend.c']
    sources += [support / 'vterm_diagnostics.c']
    sources += list(sorted(formatter.glob('*.c'))) + list(sorted(strings.glob('*.c')))
    for i, source in enumerate(sources):
        obj = temporary / ('vterm-%02d.o' % i)
        extra = []
        if source.parent == formatter:
            extra = ['-DPRINTF_DISABLE_SUPPORT_FLOAT', '-DPRINTF_DISABLE_SUPPORT_EXPONENTIAL',
                     '-DPRINTF_DISABLE_SUPPORT_LONG_LONG']
            if source.name != 'format_print.c':
                extra += ['-include', str(formatter/'format_print.h')]
        elif source.parent == strings:
            extra = ['-DPREFER_SIZE_OVER_SPEED', '-I'+str(support/'newlib-compat'),
                     '-include', str(support/'newlib-compat/_ansi.h')]
        elif source.name == 'vterm_diagnostics.c':
            extra = ['-I'+str(formatter)]
        subprocess.run(['gcc', *flags, *extra, '-c', str(source), '-o', str(obj)], check=True)
        objects.append(str(obj))
    assembly = temporary/'console.elf.o'
    convert_assembly_object(console, assembly)
    combined = temporary / 'console-combined.o'
    subprocess.run(['ld', '-r', '-m', 'elf_i386', '--gc-sections', '-u', 'EXPORTS',
                    str(assembly), *objects, '-o', str(combined)], check=True)
    unresolved = subprocess.check_output(['nm', '-u', str(combined)], text=True).strip()
    if unresolved:
        raise RuntimeError('unresolved native console symbols:\n' + unresolved)
    # The native loader walks plain symbol records without skipping COFF file
    # auxiliaries. Encode only supported records and translate ELF addends;
    # cross-format objcopy also alters local-symbol addends in this direction.
    from elf_to_kolibri_coff import convert
    convert(combined, output)
    shutil.copyfile(combined, output.with_suffix('.elf.o'))
