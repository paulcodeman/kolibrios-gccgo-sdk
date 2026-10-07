; SPDX-License-Identifier: GPL-2.0
; UTF-8 stream adapter for the upstream KolibriOS console CP866 font.
; Decoding starts from kernel/trunk/unicode.inc. The Unicode mapping comes
; from upstream Go x/text; its copyright and license accompany the table.
; This preserves the existing one-byte console ABI. Glyphs unavailable in
; its bitmap font use '?' (rounded box corners use the corresponding corner).

; uint32_t __stdcall con_set_output_mode(uint32_t utf8);
; 0 retains the original CP866 behavior, 1 decodes external string writes.
con_set_output_mode:
        mov     eax, [esp+4]
        and     eax, 1
        xchg    eax, [con.sdk_utf8_mode]
        mov     dword [con.sdk_utf8_used], 0
        ret     4

; AL is one byte from con.write; EDI is its live destination cell pointer.
; Native keyboard echo continues to use the original CP866 write_char path.
con.sdk_write_utf8:
        cmp     [con.sdk_utf8_mode], 0
        je      con.write_char_ex
        push    ebx ecx edx esi
        movzx   edx, al
        mov     ecx, [con.sdk_utf8_used]
        test    ecx, ecx
        jnz     .continuation
  .first:
        cmp     dl, 0x80
        jb      .emit
        cmp     dl, 0xc2
        jb      .replacement
        mov     ebx, 2
        cmp     dl, 0xe0
        jb      .start
        inc     ebx
        cmp     dl, 0xf0
        jb      .start
        inc     ebx
        cmp     dl, 0xf5
        jae     .replacement
  .start:
        mov     [con.sdk_utf8_bytes], dl
        mov     [con.sdk_utf8_need], ebx
        mov     [con.sdk_utf8_used], 1
        jmp     .pending
  .continuation:
        cmp     dl, 0x80
        jb      .broken
        cmp     dl, 0xc0
        jae     .broken
        mov     [con.sdk_utf8_bytes+ecx], dl
        inc     ecx
        mov     [con.sdk_utf8_used], ecx
        cmp     ecx, [con.sdk_utf8_need]
        jb      .pending
        mov     esi, con.sdk_utf8_bytes
        call    con.sdk_utf8_decode
        mov     [con.sdk_utf8_used], 0
        ; Reject overlong encodings, surrogate values and out-of-range values.
        cmp     [con.sdk_utf8_need], 2
        jne     .three
        cmp     eax, 0x80
        jb      .replacement
        jmp     .map
  .three:
        cmp     [con.sdk_utf8_need], 3
        jne     .four
        cmp     eax, 0x800
        jb      .replacement
        cmp     eax, 0xd800
        jb      .map
        cmp     eax, 0xe000
        jb      .replacement
        jmp     .map
  .four:
        cmp     eax, 0x10000
        jb      .replacement
        cmp     eax, 0x10ffff
        ja      .replacement
        ; The existing font has no supplementary-plane glyphs.
        jmp     .replacement
  .map:
        ; Lip Gloss's rounded corners have the same single-cell geometry.
        cmp     eax, 0x256d
        jne     @f
        mov     eax, 0x250c
       @@:
        cmp     eax, 0x256e
        jne     @f
        mov     eax, 0x2510
       @@:
        cmp     eax, 0x256f
        jne     @f
        mov     eax, 0x2518
       @@:
        cmp     eax, 0x2570
        jne     @f
        mov     eax, 0x2514
       @@:
        push    edi
        mov     edi, con.sdk_cp866_unicode
        mov     ecx, 128
        repne   scasw
        pop     edi
        jne     .replacement
        mov     eax, 255
        sub     eax, ecx
        jmp     .emit
  .broken:
        mov     [con.sdk_utf8_used], 0
        push    edx
        mov     al, '?'
        call    con.write_char_ex
        pop     edx
        mov     eax, edx
        jmp     .first
  .replacement:
        mov     al, '?'
  .emit:
        pop     esi edx ecx ebx
        jmp     con.write_char_ex
  .pending:
        pop     esi edx ecx ebx
        ret

include 'utf8-decode.inc'
include 'cp866-unicode.inc'

con.sdk_utf8_mode dd 0
con.sdk_utf8_used dd 0
con.sdk_utf8_need dd 0
con.sdk_utf8_bytes rb 4
szcon_set_output_mode db 'con_set_output_mode',0
