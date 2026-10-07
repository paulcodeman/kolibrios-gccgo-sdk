; Native C ABI for the upstream libvterm backend.
; Heap and scheduler register contracts: sysfuncs.txt, functions 68/12,13,1.
public sdk_console_alloc
public sdk_console_free
public sdk_console_yield
public sdk_console_debug_byte
public sdk_console_terminate
extrn sdk_vterm_start
extrn sdk_vterm_write
extrn sdk_vterm_cursor
extrn sdk_vterm_cell
extrn sdk_vterm_render
extrn sdk_vterm_output_pending
extrn sdk_vterm_read_response
extrn sdk_vterm_layout
extrn sdk_vterm_set_font
extrn sdk_vterm_glyph_width
extrn sdk_vterm_encode_cp866
extrn sdk_vterm_mouse_active
extrn sdk_vterm_mouse
extrn sdk_console_abort

sdk_console_alloc:
        push ebx
        mov ecx, [esp+8]
        mov eax, 68
        mov ebx, 12
        int 0x40
        pop ebx
        test eax, eax
        jnz .allocated
        call sdk_console_abort
  .allocated:
        ret
sdk_console_free:
        push ebx
        mov ecx, [esp+8]
        mov eax, 68
        mov ebx, 13
        int 0x40
        pop ebx
        ret
sdk_console_yield:
        push ebx
        mov eax, 68
        mov ebx, 1
        int 0x40
        pop ebx
        ret

; sysfuncs.txt: debug-board byte output (63/1), terminate thread (-1).
sdk_console_debug_byte:
        push ebx
        mov ecx, [esp+8]
        mov eax, 63
        mov ebx, 1
        int 0x40
        pop ebx
        ret
sdk_console_terminate:
        or eax, -1
        int 0x40

con_enable_vterm:
        push [con.wnd_width]
        push [con.wnd_height]
        call sdk_vterm_start
        add esp, 8
        test eax, eax
        jz .done
        mov [con.sdk_utf8_mode], 2
        call con.sdk_vterm_layout
        call con.update_screen
        mov eax, 1
  .done:
        ret

; int __stdcall con_get_cell(int x, int y, uint32_t out[4]);
con_get_cell:
        push dword [esp+12]
        push dword [esp+12]
        push dword [esp+12]
        call sdk_vterm_cell
        add esp, 12
        ret 12

; Borrow immutable font data for the lifetime of this console process.
con_set_unicode_font:
        push dword [esp+8]
        push dword [esp+8]
        call sdk_vterm_set_font
        add esp, 8
        ret 8

con_unicode_glyph_width:
        push dword [esp+4]
        call sdk_vterm_glyph_width
        add esp, 4
        ret 4

con.sdk_vterm_write:
        pushad
        mov ecx, ebx
        cmp ecx, -1
        jne .length
        xor ecx, ecx
  .scan:
        cmp byte [esi+ecx], 0
        je .length
        inc ecx
        jmp .scan
  .length:
        push ecx
        push esi
        call sdk_vterm_write
        add esp, 8
        push con.cur_y
        push con.cur_x
        call sdk_vterm_cursor
        add esp, 8
        call con.sdk_vterm_layout
        popad
        jmp con.update_screen

con.sdk_vterm_render:
        pushad
        push [con.wnd_ypos]
        push con.colors
        push font_height
        push font_width
        push font
        push [con.wnd_height]
        push [con.wnd_width]
        push [con.image]
        call sdk_vterm_render
        add esp, 32
        popad
        ret

con.sdk_vterm_layout:
        push con.cur_y
        push con.wnd_ypos
        push con.scr_height
        call sdk_vterm_layout
        add esp, 12
        ret

con.sdk_vterm_cursor:
        push dword [esp+8]
        push dword [esp+8]
        call sdk_vterm_cursor
        add esp, 8
        ret 8

con.sdk_vterm_response:
        push dword [esp+8]
        push dword [esp+8]
        call sdk_vterm_read_response
        add esp, 8
        ret 8

; Called only by the original console's native window/event thread.
; Documented register contracts: sysfuncs.txt 37/1,2,7 and 66/3.
; The upstream kernel returns 37/1 coordinates relative to the client area,
; matching con.draw_image's origin (kernel/trunk/gui/mouse.inc .mswin).
con.sdk_vterm_mouse:
        pushad
        mov eax, 66
        mov ebx, 3
        int 0x40
        push font_height
        push font_width
        push eax
        mov eax, 37
        mov ebx, 7
        int 0x40
        push eax
        mov eax, 37
        mov ebx, 2
        int 0x40
        push eax
        mov eax, 37
        mov ebx, 1
        int 0x40
        movsx ecx, ax
        sar eax, 16
        ; Correct the high-word borrow described by sysfuncs.txt for y<0.
        test ecx, ecx
        jns .position
        inc eax
  .position:
        push ecx
        push eax
        call sdk_vterm_mouse
        add esp, 28
        popad
        jmp con.msg_loop

szcon_enable_vterm db 'con_enable_vterm',0
szcon_set_unicode_font db 'con_set_unicode_font',0
szcon_unicode_glyph_width db 'con_unicode_glyph_width',0
szcon_get_cell db 'con_get_cell',0
