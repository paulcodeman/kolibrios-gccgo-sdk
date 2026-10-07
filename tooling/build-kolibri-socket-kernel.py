#!/usr/bin/env python3
"""Build an isolated KolibriOS kernel with socket, TCP and shared-memory ABI fixes."""
import argparse
import errno
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True, help='kernel/trunk directory')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--fasm', default='fasm', help='FASM executable')
    parser.add_argument('--network-debug', action='store_true')
    parser.add_argument('--local-streams', action='store_true',
                        help='notify AF_LOCAL peers on close so buffered data is followed by EOF')
    parser.add_argument('--fat-name-case', action='store_true',
                        help='preserve FAT NT lowercase flags on reads and existing LFN spelling on creates')
    parser.add_argument('--symbols', action='store_true',
                        help='write FASM symbolic information beside the kernel as .fas')
    parser.add_argument('--exclusive-create', action='store_true',
                        help='add documented SDK filesystem extensions for atomic FAT file/directory creation')
    parser.add_argument('--debug-output', action='store_true',
                        help='send debug messages to QEMU port 0xe9 without verbose network logging')
    args = parser.parse_args()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='kolibri-socket-kernel-') as directory:
        tree = Path(directory) / 'kernel'
        shutil.copytree(args.source, tree)
        socket = tree / 'network/socket.inc'
        text = socket.read_text()
        start = text.index('socket_accept:')
        end = text.index('socket_close:', start)
        accept = text[start:end]
        for label, code in [('wouldblock', 'EWOULDBLOCK'), ('invalid', 'EINVAL'), ('notsupp', 'EOPNOTSUPP')]:
            old = ('  .' + label + ':\n'
                   '        mov     dword[esp + SYSCALL_STACK.ebx], ' + code + '\n'
                   '        mov     dword[esp + SYSCALL_STACK.eax], -1\n'
                   '        pop     esi edx\n')
            new = ('  .' + label + ':\n'
                   '        pop     esi edx\n'
                   '        mov     dword[esp + SYSCALL_STACK.ebx], ' + code + '\n'
                   '        mov     dword[esp + SYSCALL_STACK.eax], -1\n')
            if accept.count(old) != 1:
                raise RuntimeError('unexpected socket_accept error path: ' + label)
            accept = accept.replace(old, new)
        socket.write_text(text[:start] + accept + text[end:])
        if args.local_streams:
            text = socket.read_text()
            old = '  .free:\n        call    socket_free\n  .end:\n'
            new = '''  .free:
        ; AF_LOCAL sockets are linked through device. Preserve buffered peer
        ; bytes, break the reciprocal link, and notify a waiting receiver.
        push eax
        cmp [eax + SOCKET.Domain], AF_LOCAL
        jne .local_close_done
        mov eax, [eax + SOCKET.device]
        test eax, eax
        jz .local_close_done
        call socket_check
        jz .local_close_done
        mov ecx, [esp]
        cmp [eax + SOCKET.device], ecx
        jne .local_close_done
        mov [eax + SOCKET.device], 0
        or [eax + SOCKET.state], SS_CANTRCVMORE or SS_CANTSENDMORE
        call socket_notify
  .local_close_done:
        pop eax
        call    socket_free
  .end:
'''
            if text.count(old) != 1:
                raise RuntimeError('unexpected local socket close path')
            text = text.replace(old,new)
            old = ('        mov     eax, [eax + SOCKET.device]\n'
                   '        call    socket_check\n'
                   '        jz      .invalid\n')
            new = ('        mov     eax, [eax + SOCKET.device]\n'
                   '        test    eax, eax\n'
                   '        jz      .invalid\n'
                   '        call    socket_check\n'
                   '        jz      .invalid\n')
            if text.count(old) != 1:
                raise RuntimeError('unexpected disconnected local send path')
            socket.write_text(text.replace(old,new))
        tcp_output = tree / 'network/tcp_output.inc'
        text = tcp_output.read_text()
        old = ('        pop     ecx\n'
               '        sub     ebx, [eax + TCP_SOCKET.RCV_ADV]\n'
               '        add     ebx, [eax + TCP_SOCKET.RCV_NXT]\n'
               '\n'
               '        cmp     ebx, ecx\n'
               '        jl      @f\n'
               '        mov     ebx, ecx\n'
               '       @@:\n')
        new = ('        pop     ecx\n'
               '\n'
               '        ; BSD: min(available, max window) - advertised window.\n'
               '        ; Clamp before subtracting, otherwise an unchanged\n'
               '        ; 32 KiB window appears to grow by 32767 bytes and\n'
               '        ; loopback peers endlessly answer ACKs with ACKs.\n'
               '        cmp     ebx, ecx\n'
               '        jl      @f\n'
               '        mov     ebx, ecx\n'
               '       @@:\n'
               '        sub     ebx, [eax + TCP_SOCKET.RCV_ADV]\n'
               '        add     ebx, [eax + TCP_SOCKET.RCV_NXT]\n')
        if text.count(old) == 1:
            tcp_output.write_text(text.replace(old, new))
        elif ('        cmp     ebx, ecx\n'
              '        jbe     @f\n'
              '        mov     ebx, ecx\n'
              '       @@:\n'
              '        sub     ebx, [eax + TCP_SOCKET.RCV_ADV]\n'
              '        add     ebx, [eax + TCP_SOCKET.RCV_NXT]\n') not in text:
            raise RuntimeError('unexpected TCP window update calculation')
        tcp_subr = tree / 'network/tcp_subr.inc'
        text = tcp_subr.read_text()
        old = ('        tcpt_rangeset [eax + TCP_SOCKET.timer_persist], ebx, TCP_time_pers_min, TCP_time_pers_max\n'
               '        or      [ebx + TCP_SOCKET.timer_flags], timer_flag_persist\n')
        new = ('        tcpt_rangeset [eax + TCP_SOCKET.timer_persist], ebx, TCP_time_pers_min, TCP_time_pers_max\n'
               '        ; EAX owns the socket; EBX holds the timeout value.\n'
               '        or      [eax + TCP_SOCKET.timer_flags], timer_flag_persist\n')
        if text.count(old) == 1:
            tcp_subr.write_text(text.replace(old, new))
        elif ('        tcpt_rangeset [eax + TCP_SOCKET.timer_persist], ebx, TCP_time_pers_min, TCP_time_pers_max\n'
              '        or      [eax + TCP_SOCKET.timer_flags], timer_flag_persist\n') not in text:
            raise RuntimeError('unexpected TCP persist timer socket pointer')
        tcp_input = tree / 'network/tcp_input.inc'
        text = tcp_input.read_text()
        old = ('        cmp     [ebx + TCP_SOCKET.t_state], TCPS_ESTABLISHED\n'
               '        jne     .out_of_order\n')
        new = ('        ; A local FIN closes only the sending direction.\n'
               '        ; Continue accepting in-order peer data until its FIN.\n'
               '        cmp     [ebx + TCP_SOCKET.t_state], TCPS_ESTABLISHED\n'
               '        je      @f\n'
               '        cmp     [ebx + TCP_SOCKET.t_state], TCPS_FIN_WAIT_1\n'
               '        je      @f\n'
               '        cmp     [ebx + TCP_SOCKET.t_state], TCPS_FIN_WAIT_2\n'
               '        jne     .out_of_order\n'
               '       @@:\n')
        if text.count(old) != 1 and text.count(new) != 1:
            raise RuntimeError('unexpected TCP in-order receive state check')
        tcp_input.write_text(text.replace(old, new))
        heap = tree / 'core/heap.inc'
        text = heap.read_text()
        old = ('        cmp     eax, SHM_CREATE\n'
               '        mov     edx, E_ACCESS\n'
               '        je      .exit\n')
        new = ('        cmp     eax, SHM_CREATE\n'
               '        mov     edx, E_ACCESS\n'
               '        je      .fail\n')
        if text.count(old) != 1 and text.count(new) != 1:
            raise RuntimeError('unexpected shared-memory exclusive-create error path')
        # .exit writes the existing area's size into EDX. .fail preserves the
        # E_ACCESS error required by syscall 68/22 when SHM_CREATE finds a name.
        heap.write_text(text.replace(old, new))
        if args.fat_name_case:
            fat = tree / 'fs/fat.inc'
            text = fat.read_text()
            start = text.index('fat_get_name:')
            end = text.index('fat_find_lfn:', start)
            names = text[start:end]
            old = '        pushd   ecx 8\n        pop     ecx\n'
            new = '        push    edx\n        mov     edx, esi\n' + old
            if names.count(old) != 1:
                raise RuntimeError('unexpected FAT short-name setup')
            names = names.replace(old, new)
            old = '        lodsb\n        call    ansi2uni_char\n        stosw\n        loop    @b\n'
            if names.count(old) != 2:
                raise RuntimeError('unexpected FAT short-name conversion loops')
            # FatFs get_fileinfo: DIR_NTres offset 12, NS_BODY=0x08,
            # NS_EXT=0x10; lower only ASCII uppercase characters.
            for flag in (8, 16):
                new = f'''        lodsb
        test    byte [edx+12], {flag}
        jz      .case_ready_{flag}
        cmp     al, 'A'
        jb      .case_ready_{flag}
        cmp     al, 'Z'
        ja      .case_ready_{flag}
        or      al, 20h
  .case_ready_{flag}:
        call    ansi2uni_char
        stosw
        loop    @b
'''
                names = names.replace(old, new, 1)
            old = '        pop     ecx edi\n        ret\n'
            if names.count(old) != 1:
                raise RuntimeError('unexpected FAT short-name return')
            names = names.replace(old, '        pop     ecx edx edi\n        ret\n')
            text = text[:start] + names + text[end:]
            # Reuse the original kernel LFN writer for names whose spelling
            # would otherwise be lost by its uppercase 8.3 generator.
            old = '''        movi    eax, 1     ; 1 entry
        jnz     .notilde
; we need ceil(strlen(esi)/13) additional entries'''
            new = '''        movi    eax, 1     ; 1 entry
        jz      .preserve_lfn_case
        push    eax esi
  .check_original_case:
        lodsb
        test    al, al
        jz      .original_uppercase
        cmp     al, 'a'
        jb      .check_original_case
        cmp     al, 'z'
        ja      .check_original_case
        pop     esi eax
        jmp     .preserve_lfn_case
  .original_uppercase:
        pop     esi eax
        jmp     .notilde
  .preserve_lfn_case:
; we need ceil(strlen(esi)/13) additional entries'''
            if text.count(old) != 1:
                raise RuntimeError('unexpected FAT LFN allocation decision')
            fat.write_text(text.replace(old, new))
        if args.exclusive_create:
            fat = tree / 'fs/fat.inc'
            text = fat.read_text()
            old = '        dd      fat_Rename\nfat_user_functions_end:'
            reserved = '        dd      0, 0, 0 ; upstream 11/12 symlinks, 13 volume info\n'
            if old not in text:
                old = ('        dd      fat_Rename\n'
                       '        dd      0, 0\n'
                       '        dd      fat_GetVolumeInfo\nfat_user_functions_end:')
                reserved = '        dd      0, 0\n        dd      fat_GetVolumeInfo\n'
            new = ('        dd      fat_Rename\n' + reserved +
                   '        dd      fat_CreateFile      ; 14: exclusive file\n'
                   '        dd      fat_CreateFolder    ; 15: exclusive directory\n'
                   'fat_user_functions_end:')
            if text.count(old) != 1:
                raise RuntimeError('unexpected FAT service table')
            text = text.replace(old, new)
            old = ('fat_CreateFolder:\n'
                   '        mov     [ebp+FAT.createOption], 0\n'
                   '        jmp     @f\n\n'
                   'fat_CreateFile:\n'
                   '        mov     [ebp+FAT.createOption], 1\n'
                   '@@:\n'
                   '        call    fat_lock\n')
            new = ('fat_CreateFolder:\n'
                   '        call    fat_lock\n'
                   '        mov     [ebp+FAT.createOption], 0\n'
                   '        jmp     @f\n\n'
                   'fat_CreateFile:\n'
                   '        call    fat_lock\n'
                   '        mov     [ebp+FAT.createOption], 1\n'
                   '@@:\n')
            if text.count(old) != 1:
                raise RuntimeError('unexpected FAT create locking path')
            # createOption belongs to the partition. Setting it before
            # acquiring the mutex lets a waiting creator alter the owner.
            text = text.replace(old, new)
            old = ('.common1:\n'
                   '        call    fat_find_lfn\n'
                   '        jc      .notfound\n'
                   '        test    byte [edi+11], 10h\n')
            new = ('.common1:\n'
                   '        call    fat_find_lfn\n'
                   '        jc      .notfound\n'
                   '        ; The original request EBX is saved below the\n'
                   '        ; nine directory iterator words and pushad.\n'
                   '        mov     eax, [esp+36+16]\n'
                   '        cmp     dword [eax], 14\n'
                   '        jb      .sdk_nonexclusive\n'
                   '        cmp     dword [eax], 15\n'
                   '        ja      .sdk_nonexclusive\n'
                   '        add     esp, 36\n'
                   '        movi    eax, 17 ; SDK ERROR_ALREADY_EXISTS\n'
                   '        jmp     .ret1\n'
                   '.sdk_nonexclusive:\n'
                   '        test    byte [edi+11], 10h\n')
            if text.count(old) != 1:
                raise RuntimeError('unexpected FAT existing-name path')
            fat.write_text(text.replace(old, new))
            filesystem = tree / 'fs/fs_lfn.inc'
            text = filesystem.read_text()
            old = ('.case2_3:\n'
                   '        cmp     dword [ebx], 3\n'
                   '        ja      .case5 ; if subfn > 3\n'
                   '        mov     ecx, dword [ebx + 12]\n')
            new = ('.case2_3:\n'
                   '        cmp     dword [ebx], 14 ; exclusive write\n'
                   '        je      .sdk_write_buffer\n'
                   '        cmp     dword [ebx], 3\n'
                   '        ja      .case5 ; if subfn > 3\n'
                   '.sdk_write_buffer:\n'
                   '        mov     ecx, dword [ebx + 12]\n')
            if text.count(old) != 1:
                raise RuntimeError('unexpected filesystem user-buffer validation')
            filesystem.write_text(text.replace(old, new))
        if args.network_debug:
            stack = tree / 'network/stack.inc'
            stack.write_text(stack.read_text().replace('DEBUG_NETWORK_VERBOSE   = 0', 'DEBUG_NETWORK_VERBOSE   = 1'))
        (tree / 'lang.inc').write_text('lang fix en_US\n')
        command = [args.fasm, '-m', '262144', 'kernel.asm', str(args.output.resolve())]
        if args.symbols:
            command += ['-s', str(args.output.resolve()) + '.fas']
        if args.network_debug or args.debug_output:
            command += ['-dpretest_build=1', '-ddebug_com_base=0xe9']
        try:
            subprocess.run(command, cwd=tree, check=True)
        except OSError as error:
            if error.errno != errno.ENOEXEC or not shutil.which('qemu-i386'):
                raise
            command[0] = shutil.which(args.fasm)
            subprocess.run(['qemu-i386'] + command, cwd=tree, check=True)
    print('Built kernel without changing the source checkout: ' + str(args.output))


if __name__ == '__main__':
    main()
