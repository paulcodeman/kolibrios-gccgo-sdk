SECTION .text

extern runtime_prepare_window_title
extern runtime_prepare_window_title_with_prefix
extern go_0kos.ThreadBootstrap

global go_0kos.Sleep
global go_0kos.GetKey
global go_0kos.GetControlKeysRaw
global go_0kos.SetKeyboardLayoutRaw
global go_0kos.SetKeyboardLanguageRaw
global go_0kos.SetSystemLanguageRaw
global go_0kos.GetKeyboardLayoutRaw
global go_0kos.GetKeyboardLanguageRaw
global go_0kos.GetSystemLanguageRaw
global go_0kos.Event
global go_0kos.CheckEvent
global go_0kos.GetThreadInfo
global go_0kos.CreateThreadRaw
global go_0kos.GetCurrentThreadSlotRaw
global go_0kos.ThreadEntryAddrRaw
global go_0kos.threadEntry
global go_0kos.SyscallRaw
global go_0kos.GetButtonID
global go_0kos.CreateButton
global go_0kos.ExitRaw
global go_0kos.Redraw
global go_0kos.Window
global go_0kos.WriteText
global go_0kos.WriteTextEx
global go_0kos.GetTime
global go_0kos.GetDate
global go_0kos.GetTimeCounter
global go_0kos.GetTimeCounterPro
global go_0kos.GetScreenSize
global go_0kos.GetScreenWorkingArea
global go_0kos.GetSkinHeight
global go_0kos.GetSkinMarginsRaw
global go_0kos.GetFontSmoothingRaw
global go_0kos.SetFontSmoothingRaw
global go_0kos.WaitEventTimeout
global go_0kos.SetEventMask
global go_0kos.SetPortsRaw
global go_0kos.SetIPCArea
global go_0kos.SendIPCMessage
global go_0kos.FocusWindowBySlot
global go_0kos.GetActiveWindowSlotRaw
global go_0kos.SetMousePointerPositionRaw
global go_0kos.SimulateMouseButtonsRaw
global go_0kos.GetKernelVersion
global go_0kos.SystemShutdown
global go_0kos.GetFreeRAM
global go_0kos.GetTotalRAM
global go_0kos.InitHeapRaw
global go_0kos.HeapAllocRaw
global go_0kos.HeapFreeRaw
global go_0kos.HeapReallocRaw
global runtime_kos_heap_init_raw
global runtime_kos_heap_alloc_raw
global runtime_kos_heap_free_raw
global runtime_kos_heap_realloc_raw
global go_0kos.LoadDLLWithEncoding
global go_0kos.LoadDLL
global runtime_kos_load_dll_cstring_raw
global go_0kos.GetCurrentFolderRaw
global go_0kos.SetCaption
global go_0kos.SetCaptionWithPrefix
global go_0kos.SendMessage
global go_0kos.FileSystem
global go_0kos.FileSystemEncoded
global go_0kos.PosixReadRaw
global go_0kos.PosixWriteRaw
global go_0kos.PosixPipe2Raw
global go_0kos.GetMouseScreenPosition
global go_0kos.GetMouseWindowPosition
global go_0kos.GetMouseButtonState
global go_0kos.GetMouseButtonEventState
global go_0kos.LoadCursorRaw
global go_0kos.SetCursorRaw
global go_0kos.DeleteCursorRaw
global go_0kos.GetMouseScrollData
global go_0kos.GetPixelColorFromScreenRaw
global go_0kos.LoadCursorWithEncoding
global go_0kos.SetWindowLayerBehaviourRaw
global go_0kos.ClipboardSlotCountRaw
global go_0kos.ClipboardSlotDataRaw
global go_0kos.ClipboardWriteRaw
global go_0kos.ClipboardDeleteLastRaw
global go_0kos.ClipboardUnlockBufferRaw
global go_0kos.DrawLine
global go_0kos.DrawBar
global go_0kos.PutPaletteImage
global go_0kos.SetSkin
global go_0kos.SetSkinWithEncoding
global go_0kos.DebugOutHex
global go_0kos.DebugOutChar
global go_0kos.DebugOutStr
global go_0kos.DebugReadRaw
global go_0kos.DebugSetMessageAreaRaw
global go_0kos.DebugGetRegistersRaw
global go_0kos.DebugSuspendRaw
global go_0kos.DebugResumeRaw
global go_0kos.DebugReadMemoryRaw
global go_0kos.PortWriteByteRaw

go_0kos.Sleep:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 5
    mov ebx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.Event:
    mov eax, 10
    int 0x40
    ret

go_0kos.GetKey:
    mov eax, 2
    int 0x40
    ret

go_0kos.GetControlKeysRaw:
    push ebx
    mov eax, 66
    mov ebx, 3
    int 0x40
    pop ebx
    ret

go_0kos.SetKeyboardLayoutRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 21
    mov ebx, 2
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetKeyboardLanguageRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 21
    mov ebx, 2
    mov ecx, 9
    mov edx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetSystemLanguageRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 21
    mov ebx, 5
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetKeyboardLayoutRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 26
    mov ebx, 2
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetKeyboardLanguageRaw:
    push ebx
    mov eax, 26
    mov ebx, 2
    mov ecx, 9
    int 0x40
    pop ebx
    ret

go_0kos.GetSystemLanguageRaw:
    push ebx
    mov eax, 26
    mov ebx, 5
    int 0x40
    pop ebx
    ret

go_0kos.CheckEvent:
    mov eax, 11
    int 0x40
    ret

go_0kos.GetThreadInfo:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 9
    mov ebx, [ebp+8]
    mov ecx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.CreateThreadRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 51
    mov ebx, 1
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetCurrentThreadSlotRaw:
    push ebx
    mov eax, 51
    mov ebx, 2
    int 0x40
    pop ebx
    ret

go_0kos.ThreadEntryAddrRaw:
    mov eax, go_0kos.threadEntry
    ret

go_0kos.threadEntry:
    mov eax, [esp]
    and esp, 0xFFFFFFF0
    sub esp, 8
    push eax
    call go_0kos.ThreadBootstrap
    add esp, 12
    call go_0kos.ExitRaw
    ret

go_0kos.SyscallRaw:
    push ebp
    push ebx
    push esi
    push edi
    mov eax, [esp+20]
    mov ebx, [eax+4]
    mov ecx, [eax+8]
    mov edx, [eax+12]
    mov esi, [eax+16]
    mov edi, [eax+20]
    mov ebp, [eax+24]
    mov eax, [eax+0]
    int 0x40
    push edi
    mov edi, [esp+24]
    mov [edi+0], eax
    mov [edi+4], ebx
    mov [edi+8], ecx
    mov [edi+12], edx
    mov [edi+16], esi
    mov eax, [esp]
    mov [edi+20], eax
    mov [edi+24], ebp
    pop eax
    pop edi
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.GetButtonID:
    mov eax, 17
    int 0x40
    cmp eax, 1
    je .no_button
    shr eax, 8
    ret
.no_button:
    xor eax, eax
    dec eax
    ret

go_0kos.ExitRaw:
    mov eax, -1
    int 0x40
    ret

go_0kos.Redraw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 12
    mov ebx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.Window:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    push edi
    push dword [ebp+28]
    push dword [ebp+24]
    call runtime_prepare_window_title
    add esp, 8
    mov edi, eax
    mov ebx, [ebp+8]
    shl ebx, 16
    or ebx, [ebp+16]
    mov ecx, [ebp+12]
    shl ecx, 16
    or ecx, [ebp+20]
    mov edx, 0x13
    shl edx, 24
    or edx, 0xFFFFFF
    mov esi, 0x808899FF
    xor eax, eax
    int 0x40
    pop edi
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.WriteText:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 4
    mov ebx, [ebp+8]
    shl ebx, 16
    mov bx, [ebp+12]
    mov ecx, [ebp+16]
    and ecx, 0x00FFFFFF
    or ecx, 0x30000000
    mov edx, [ebp+20]
    mov esi, [ebp+24]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.WriteTextEx:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    push edi
    mov eax, 4
    mov ebx, [ebp+8]
    shl ebx, 16
    mov bx, [ebp+12]
    mov ecx, [ebp+16]
    mov edx, [ebp+20]
    mov esi, [ebp+24]
    mov edi, [ebp+28]
    int 0x40
    pop edi
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.DrawLine:
    push ebp
    mov ebp, esp
    push ebx
    mov ebx, [ebp+8]
    shl ebx, 16
    mov bx, [ebp+16]
    mov ecx, [ebp+12]
    shl ecx, 16
    mov cx, [ebp+20]
    mov edx, [ebp+24]
    mov eax, 38
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DrawBar:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 13
    mov ebx, [ebp+8]
    shl ebx, 16
    mov bx, [ebp+16]
    mov ecx, [ebp+12]
    shl ecx, 16
    mov cx, [ebp+20]
    mov edx, [ebp+24]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.PutPaletteImage:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    push edi
    mov eax, 65
    mov ebx, [ebp+8]
    mov ecx, [ebp+12]
    shl ecx, 16
    mov cx, [ebp+16]
    mov edx, [ebp+20]
    shl edx, 16
    mov dx, [ebp+24]
    mov esi, [ebp+28]
    mov edi, [ebp+32]
    mov ebp, [ebp+36]
    int 0x40
    pop edi
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.GetTime:
    mov eax, 3
    int 0x40
    ret

go_0kos.GetDate:
    mov eax, 29
    int 0x40
    ret

go_0kos.GetTimeCounter:
    push ebx
    mov eax, 26
    mov ebx, 9
    int 0x40
    pop ebx
    ret

go_0kos.GetTimeCounterPro:
    push ebx
    mov eax, 26
    mov ebx, 10
    int 0x40
    pop ebx
    ret

go_0kos.GetScreenSize:
    mov eax, 14
    int 0x40
    ret

go_0kos.GetScreenWorkingArea:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 48
    mov ebx, 5
    int 0x40
    mov edx, [ebp+8]
    test edx, edx
    jz .done_screen_working_area
    mov [edx], ebx
.done_screen_working_area:
    pop ebx
    pop ebp
    ret

go_0kos.GetSkinHeight:
    push ebx
    mov eax, 48
    mov ebx, 4
    int 0x40
    pop ebx
    ret

go_0kos.GetSkinMarginsRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 48
    mov ebx, 7
    int 0x40
    mov edx, [ebp+8]
    test edx, edx
    jz .done_skin_margins
    mov [edx], ebx
.done_skin_margins:
    pop ebx
    pop ebp
    ret

go_0kos.GetFontSmoothingRaw:
    push ebx
    mov eax, 48
    mov ebx, 9
    int 0x40
    pop ebx
    ret

go_0kos.SetFontSmoothingRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 48
    mov ebx, 10
    mov ecx, [ebp+8]
    and ecx, 0xFF
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetSkin:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+12]
    push dword [ebp+8]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov eax, 48
    mov ebx, 8
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetSkinWithEncoding:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+16]
    push dword [ebp+12]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov edx, [ebp+8]
    mov eax, 48
    mov ebx, 13
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.WaitEventTimeout:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 23
    mov ebx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetEventMask:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 40
    mov ebx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetPortsRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 46
    mov ebx, [ebp+8]
    mov ecx, [ebp+12]
    mov edx, [ebp+16]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetIPCArea:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 60
    mov ebx, 1
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SendIPCMessage:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 60
    mov ebx, 2
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    mov esi, [ebp+16]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.FocusWindowBySlot:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 18
    mov ebx, 3
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetActiveWindowSlotRaw:
    push ebx
    mov eax, 18
    mov ebx, 7
    int 0x40
    pop ebx
    ret

go_0kos.SetMousePointerPositionRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 18
    mov ebx, 19
    mov ecx, 4
    mov edx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SimulateMouseButtonsRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 18
    mov ebx, 19
    mov ecx, 5
    mov edx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetWindowLayerBehaviourRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 18
    mov ebx, 25
    mov ecx, 2
    mov edx, [ebp+8]
    mov esi, [ebp+12]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.GetKernelVersion:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 18
    mov ebx, 13
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SystemShutdown:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 18
    mov ebx, 9
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetFreeRAM:
    push ebx
    mov eax, 18
    mov ebx, 16
    int 0x40
    pop ebx
    ret

go_0kos.GetTotalRAM:
    push ebx
    mov eax, 18
    mov ebx, 17
    int 0x40
    pop ebx
    ret

runtime_kos_heap_init_raw:
go_0kos.InitHeapRaw:
    push ebx
    mov eax, 68
    mov ebx, 11
    int 0x40
    pop ebx
    ret

runtime_kos_heap_alloc_raw:
go_0kos.HeapAllocRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 68
    mov ebx, 12
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

runtime_kos_heap_free_raw:
go_0kos.HeapFreeRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 68
    mov ebx, 13
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

runtime_kos_heap_realloc_raw:
go_0kos.HeapReallocRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 68
    mov ebx, 20
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.LoadDLLWithEncoding:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+16]
    push dword [ebp+12]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov edx, [ebp+8]
    mov eax, 68
    mov ebx, 18
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.LoadDLL:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+12]
    push dword [ebp+8]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov eax, 68
    mov ebx, 19
    int 0x40
    pop ebx
    pop ebp
    ret

runtime_kos_load_dll_cstring_raw:
    push ebp
    mov ebp, esp
    push ebx
    mov ecx, [ebp+8]
    mov eax, 68
    mov ebx, 19
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetCurrentFolderRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 30
    mov ebx, 5
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    mov esi, [ebp+16]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.SetCaption:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+12]
    push dword [ebp+8]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov eax, 71
    mov ebx, 2
    xor edx, edx
    mov dl, 3
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetCaptionWithPrefix:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+16]
    push dword [ebp+12]
    push dword [ebp+8]
    call runtime_prepare_window_title_with_prefix
    add esp, 12
    mov ecx, eax
    mov eax, 71
    mov ebx, 1
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SendMessage:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 72
    mov ebx, 1
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.FileSystem:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 70
    mov ebx, [ebp+8]
    int 0x40
    mov edx, [ebp+12]
    test edx, edx
    jz .done_file_system
    mov [edx], ebx
.done_file_system:
    pop ebx
    pop ebp
    ret

go_0kos.FileSystemEncoded:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 80
    mov ebx, [ebp+8]
    int 0x40
    mov edx, [ebp+12]
    test edx, edx
    jz .done_file_system_encoded
    mov [edx], ebx
.done_file_system_encoded:
    pop ebx
    pop ebp
    ret

go_0kos.ClipboardSlotCountRaw:
    push ebx
    mov eax, 54
    mov ebx, 0
    int 0x40
    pop ebx
    ret

go_0kos.ClipboardSlotDataRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 54
    mov ebx, 1
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.ClipboardWriteRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 54
    mov ebx, 2
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.ClipboardDeleteLastRaw:
    push ebx
    mov eax, 54
    mov ebx, 3
    int 0x40
    pop ebx
    ret

go_0kos.ClipboardUnlockBufferRaw:
    push ebx
    mov eax, 54
    mov ebx, 4
    int 0x40
    pop ebx
    ret

go_0kos.PosixReadRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 77
    mov ebx, 10
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    mov esi, [ebp+16]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.PosixWriteRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 77
    mov ebx, 11
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    mov esi, [ebp+16]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.PosixPipe2Raw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 77
    mov ebx, 13
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetMouseScreenPosition:
    push ebx
    mov eax, 37
    xor ebx, ebx
    int 0x40
    pop ebx
    ret

go_0kos.GetMouseWindowPosition:
    push ebx
    mov eax, 37
    mov ebx, 1
    int 0x40
    pop ebx
    ret

go_0kos.GetMouseButtonState:
    push ebx
    mov eax, 37
    mov ebx, 2
    int 0x40
    pop ebx
    ret

go_0kos.GetMouseButtonEventState:
    push ebx
    mov eax, 37
    mov ebx, 3
    int 0x40
    pop ebx
    ret

go_0kos.LoadCursorRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 37
    mov ebx, 4
    mov ecx, [ebp+8]
    mov edx, [ebp+12]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.SetCursorRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 37
    mov ebx, 5
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DeleteCursorRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 37
    mov ebx, 6
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.GetMouseScrollData:
    push ebx
    mov eax, 37
    mov ebx, 7
    int 0x40
    pop ebx
    ret

go_0kos.GetPixelColorFromScreenRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 35
    mov ebx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.LoadCursorWithEncoding:
    push ebp
    mov ebp, esp
    push ebx
    push dword [ebp+16]
    push dword [ebp+12]
    call runtime_prepare_window_title
    add esp, 8
    mov ecx, eax
    mov edx, [ebp+8]
    mov eax, 37
    mov ebx, 8
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DebugOutHex:
    mov eax, [esp+4]
    mov edx, 8
.next_hex_digit:
    rol eax, 4
    movzx ecx, al
    and cl, 0x0F
    mov cl, [__hexdigits + ecx]
    pushad
    mov eax, 63
    mov ebx, 1
    int 0x40
    popad
    dec edx
    jnz .next_hex_digit
    ret

go_0kos.DebugOutChar:
    mov al, [esp+4]
    pushf
    pushad
    mov cl, al
    mov eax, 63
    mov ebx, 1
    int 0x40
    popad
    popf
    ret

go_0kos.DebugOutStr:
    push ebx
    push esi
    mov edx, [esp+12]
    mov esi, [esp+16]
    mov eax, 63
    mov ebx, 1
.next_char:
    test esi, esi
    jz .done
    mov cl, [edx]
    int 0x40
    inc edx
    dec esi
    jmp .next_char
.done:
    pop esi
    pop ebx
    ret

go_0kos.DebugReadRaw:
    push ebx
    mov eax, 63
    mov ebx, 2
    int 0x40
    shl ebx, 8
    and eax, 0xFF
    or eax, ebx
    pop ebx
    ret

go_0kos.DebugSetMessageAreaRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 69
    mov ebx, 0
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DebugGetRegistersRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 69
    mov ebx, 1
    mov ecx, [ebp+8]
    mov edx, 40
    mov esi, [ebp+12]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.DebugSuspendRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 69
    mov ebx, 4
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DebugResumeRaw:
    push ebp
    mov ebp, esp
    push ebx
    mov eax, 69
    mov ebx, 5
    mov ecx, [ebp+8]
    int 0x40
    pop ebx
    pop ebp
    ret

go_0kos.DebugReadMemoryRaw:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    push edi
    mov eax, 69
    mov ebx, 6
    mov ecx, [ebp+8]
    mov edx, [ebp+20]
    mov esi, [ebp+12]
    mov edi, [ebp+16]
    int 0x40
    pop edi
    pop esi
    pop ebx
    pop ebp
    ret

go_0kos.PortWriteByteRaw:
    mov edx, [esp+4]
    mov eax, [esp+8]
    out dx, al
    ret

go_0kos.CreateButton:
    push ebp
    mov ebp, esp
    push ebx
    push esi
    mov eax, 8
    mov ebx, [ebp+8]
    shl ebx, 16
    mov bx, [ebp+16]
    mov ecx, [ebp+12]
    shl ecx, 16
    mov cx, [ebp+20]
    mov edx, [ebp+24]
    mov esi, [ebp+28]
    int 0x40
    pop esi
    pop ebx
    pop ebp
    ret

global malloc
global free
global realloc

malloc:
    push ebx
    call __ensure_heap_initialized
    test eax, eax
    jz .malloc_failed
    mov eax, 68
    mov ebx, 12
    mov ecx, [esp+8]
    int 0x40
    pop ebx
    ret
.malloc_failed:
    xor eax, eax
    pop ebx
    ret

free:
    push ebx
    call __ensure_heap_initialized
    test eax, eax
    jz .free_failed
    mov eax, 68
    mov ebx, 13
    mov ecx, [esp+8]
    int 0x40
    pop ebx
    ret
.free_failed:
    xor eax, eax
    pop ebx
    ret

realloc:
    push ebx
    call __ensure_heap_initialized
    test eax, eax
    jz .realloc_failed
    mov eax, 68
    mov ebx, 20
    mov edx, [esp+8]
    mov ecx, [esp+12]
    int 0x40
    pop ebx
    ret
.realloc_failed:
    xor eax, eax
    pop ebx
    ret

__ensure_heap_initialized:
    cmp dword [__heap_initialized], 0
    jne .ready
    mov eax, 68
    mov ebx, 11
    int 0x40
    test eax, eax
    jz .failed
    mov dword [__heap_initialized], 1
.ready:
    mov eax, 1
    ret
.failed:
    xor eax, eax
    ret

SECTION .data

__hexdigits:
    db '0123456789ABCDEF'

SECTION .bss

__heap_initialized:
    resd 1
