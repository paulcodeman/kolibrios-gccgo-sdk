// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package kos

// Low-level bindings are kept exported to preserve the original assembly ABI.
// sysfuncs.txt is the source of truth for the function numbers and contracts.

// Function 5 - delay.
func Sleep(uint32)

// Function 3 - get system time.
func GetTime() uint32

// Function 29 - get system date.
func GetDate() uint32

// Function 26, subfunction 9 - get uptime counter in 1/100 second.
func GetTimeCounter() uint32

// Function 26, subfunction 10 - get high precision uptime counter in nanoseconds.
func GetTimeCounterPro() uint64

// Function 10 - wait for event.
func Event() int

// Function 2 - get the code of the pressed key.
func GetKey() int

// Function 66, subfunction 3 - get status of control keys.
func GetControlKeysRaw() uint32

// Function 21, subfunction 2 - set one of the keyboard layout tables.
func SetKeyboardLayoutRaw(which int, table *byte) int

// Function 21, subfunction 2 - set the global keyboard layout language id.
func SetKeyboardLanguageRaw(language int) int

// Function 21, subfunction 5 - set the global system language id.
func SetSystemLanguageRaw(language int) int

// Function 26, subfunction 2 - get one of the keyboard layout tables.
func GetKeyboardLayoutRaw(which int, buffer *byte) int

// Function 26, subfunction 2 - get the global keyboard layout language id.
func GetKeyboardLanguageRaw() int

// Function 26, subfunction 5 - get the global system language id.
func GetSystemLanguageRaw() int

// Function 11 - check for event, no wait.
func CheckEvent() int

// Function 9 - information on execution thread.
func GetThreadInfo(buffer *byte, slot int) int

// Function 51, subfunction 1 - create thread.
func CreateThreadRaw(entry uint32, stack uint32) int

// Function 51, subfunction 2 - get current thread slot number.
func GetCurrentThreadSlotRaw() int

// Thread entrypoint helper for CreateThreadRaw.
func ThreadEntryAddrRaw() uint32

// Function 23 - wait for event with timeout.
func WaitEventTimeout(uint32) int

// Function 40 - set the mask for expected events.
func SetEventMask(uint32) uint32

// Function 46 - reserve/free a group of I/O ports.
func SetPortsRaw(mode int, start uint32, end uint32) int

// Function 60, subfunction 1 - register the IPC receive area.
func SetIPCArea(buffer *byte, size uint32) uint32

// Function 60, subfunction 2 - send an IPC message to a PID/TID.
func SendIPCMessage(pid uint32, data *byte, size uint32) uint32

// Function 18, subfunction 3 - make active the window of the given thread slot.
func FocusWindowBySlot(int)

// Function 18, subfunction 7 - get the slot number of the active window.
func GetActiveWindowSlotRaw() int

// Function 18, subfunction 19, subsubfunction 4 - set mouse pointer position.
func SetMousePointerPositionRaw(uint32)

// Function 18, subfunction 19, subsubfunction 5 - simulate mouse buttons state.
func SimulateMouseButtonsRaw(uint32)

// Function 18, subfunction 25, subsubfunction 2 - set window position relative to other windows.
func SetWindowLayerBehaviourRaw(pid int, position int) int

// Function 48, subfunction 4 - get skinned-window header height.
func GetSkinHeight() int

// Function 48, subfunction 7 - get skin margins for header text layout.
func GetSkinMarginsRaw(vertical *uint32) uint32

// Function 48, subfunction 9 - get font smoothing setting.
func GetFontSmoothingRaw() uint32

// Function 48, subfunction 10 - set font smoothing setting.
func SetFontSmoothingRaw(mode uint8)

// Function 48, subfunction 8 - set the current skin using the default encoding path contract.
func SetSkin(path string) uint32

// Function 48, subfunction 13 - set the current skin using an explicit path encoding.
func SetSkinWithEncoding(encoding StringEncoding, path string) uint32

// Function 48, subfunction 5 - get packed screen working-area coordinates.
func GetScreenWorkingArea(bottom *uint32) uint32

// Function 18, subfunction 13 - get kernel version metadata.
func GetKernelVersion(buffer *byte)

// Function 18, subfunction 9 - system shutdown with a mode parameter.
func SystemShutdown(uint32) uint32

// Function 18, subfunction 16 - get size of free RAM in kilobytes.
func GetFreeRAM() uint32

// Function 18, subfunction 17 - get total RAM in kilobytes.
func GetTotalRAM() uint32

// Function 68, subfunction 11 - initialize the process heap.
func InitHeapRaw() uint32

// Function 68, subfunction 12 - allocate a heap block.
func HeapAllocRaw(size uint32) uint32

// Function 68, subfunction 13 - free a heap block.
func HeapFreeRaw(ptr uint32) uint32

// Function 68, subfunction 20 - reallocate a heap block.
func HeapReallocRaw(size uint32, ptr uint32) uint32

// Function 68, subfunction 18 - load DLL with explicit path encoding.
func LoadDLLWithEncoding(encoding StringEncoding, path string) uint32

// Function 68, subfunction 19 - load DLL using the legacy/default path contract.
func LoadDLL(path string) uint32

// Runtime helper - load DLL using a zero-terminated path and the legacy/default path contract.
func LoadDLLFromCStringRaw(path *byte) uint32 __asm__("runtime_kos_load_dll_cstring_raw")

// Runtime helper - resolve a function pointer from a DLL export table.
func LookupDLLExportRaw(table uint32, name *byte) uint32 __asm__("runtime_kos_lookup_dll_export")

// Runtime helper - invoke a stdcall function pointer with 0 arguments.
func CallStdcall0Raw(proc uint32) uint32 __asm__("runtime_kos_call_stdcall0")

// Runtime helper - invoke a stdcall function pointer with 1 argument.
func CallStdcall1Raw(proc uint32, arg0 uint32) uint32 __asm__("runtime_kos_call_stdcall1")

// Runtime helper - invoke a stdcall function pointer with 2 arguments.
func CallStdcall2Raw(proc uint32, arg0 uint32, arg1 uint32) uint32 __asm__("runtime_kos_call_stdcall2")

// Runtime helper - invoke a stdcall function pointer with 3 arguments.
func CallStdcall3Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32) uint32 __asm__("runtime_kos_call_stdcall3")

// Runtime helper - invoke a stdcall function pointer with 4 arguments.
func CallStdcall4Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32) uint32 __asm__("runtime_kos_call_stdcall4")

// Runtime helper - invoke a stdcall function pointer with 5 arguments.
func CallStdcall5Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) uint32 __asm__("runtime_kos_call_stdcall5")

// Runtime helper - invoke a stdcall function pointer with 6 arguments.
func CallStdcall6Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32, arg5 uint32) uint32 __asm__("runtime_kos_call_stdcall6")

// Runtime helper - invoke a stdcall function pointer with 7 arguments.
func CallStdcall7Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32, arg5 uint32, arg6 uint32) uint32 __asm__("runtime_kos_call_stdcall7")

// Runtime helper - invoke a cdecl function pointer with 2 arguments.
func CallCdecl2Raw(proc uint32, arg0 uint32, arg1 uint32) uint32 __asm__("runtime_kos_call_cdecl2")

// Runtime helper - invoke a cdecl function pointer with 5 arguments.
func CallCdecl5Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) uint32 __asm__("runtime_kos_call_cdecl5")

// Runtime helper - invoke a stdcall function pointer with 1 argument and no return value.
func CallStdcall1VoidRaw(proc uint32, arg0 uint32) __asm__("runtime_kos_call_stdcall1_void")

// Runtime helper - invoke a stdcall function pointer with 2 arguments and no return value.
func CallStdcall2VoidRaw(proc uint32, arg0 uint32, arg1 uint32) __asm__("runtime_kos_call_stdcall2_void")

// Runtime helper - invoke a stdcall function pointer with 5 arguments and no return value.
func CallStdcall5VoidRaw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) __asm__("runtime_kos_call_stdcall5_void")

// Runtime helper - check whether a shared active console bridge is registered.
func ConsoleBridgeReadyRaw() uint32 __asm__("runtime_console_bridge_ready")

// Runtime helper - register active console write/exit procedures in shared runtime state.
func ConsoleBridgeSetRaw(table uint32, writeProc uint32, exitProc uint32, getsProc uint32) __asm__("runtime_console_bridge_set")

// Runtime helper - clear the shared active console bridge if the table matches.
func ConsoleBridgeClearRaw(table uint32) __asm__("runtime_console_bridge_clear")

// Runtime helper - write directly through the shared active console bridge.
func ConsoleBridgeWriteRaw(data uint32, size uint32) uint32 __asm__("runtime_console_bridge_write")

// Runtime helper - read one line through the shared active console bridge.
func ConsoleBridgeReadLineRaw(data uint32, size uint32) uint32 __asm__("runtime_console_bridge_read_line")

// Runtime helper - close the shared active console bridge and clear it.
func ConsoleBridgeCloseRaw(closeWindow uint32) __asm__("runtime_console_bridge_close")

// Runtime helper - call a DLL lib_init entry with runtime alloc/free/realloc/load callbacks
// using the KolibriOS dll.inc register contract (eax/ebx/ecx/edx).
func InitDLLLibraryRaw(proc uint32) uint32 __asm__("runtime_kos_init_dll_library")

// Runtime helper - return the address of a no-op callback suitable for dialog redraw hooks.
func DialogNoopProcRaw() uint32 __asm__("runtime_kos_dialog_noop_addr")

// Runtime helper - copy a zero-terminated C string into a Go string.
func CStringToStringRaw(ptr uint32) string __asm__("runtime_cstring_to_gostring")

// Runtime helper - return the current process loader parameters buffer.
func LoaderParametersRaw() uint32 __asm__("runtime_kolibri_loader_parameters_raw")

// Runtime helper - return the current process loader path buffer.
func LoaderPathRaw() uint32 __asm__("runtime_kolibri_loader_path_raw")

// Runtime helper - copy bytes from an arbitrary memory address into a Go slice.
func CopyBytesRaw(ptr uint32, size uint32) []byte __asm__("runtime_copy_bytes")

// Runtime helper - read one dword from an arbitrary memory address plus offset.
func ReadUint32Raw(base uint32, offset uint32) uint32 __asm__("runtime_read_u32")

// Runtime helper - report whether the bootstrap runtime includes a collector.
func BootstrapRuntimeHasGCRaw() uint32 __asm__("runtime_bootstrap_has_gc")

// Runtime helper - trigger a collection when allocation pressure has crossed the GC threshold.
func PollRuntimeGCRaw() __asm__("runtime_gc_poll")
// Runtime helper - allow a thread to participate in stop-the-world.
func PollRuntimeWorldStopRaw() __asm__("runtime_kolibri_poll_world_stop")
// Runtime helper - get current runtime M count.
func GetRuntimeMCountRaw() uint32 __asm__("runtime_kolibri_get_m_count")

// Runtime helper - start a goroutine on a new runtime-managed OS thread.
func StartRuntimeThreadRaw(record uint32, stackSize uint32) uint32 __asm__("runtime_kolibri_start_locked")

// Runtime helper - set the number of runtime threads (M) to use.
func SetRuntimeThreadsRaw(count uint32) uint32 __asm__("runtime_kolibri_set_threads")

// Runtime helper - get the configured runtime thread (M) count.
func GetRuntimeThreadsRaw() uint32 __asm__("runtime_kolibri_get_threads")

// Runtime helper - heap/GC counters for diagnostics.
func HeapAllocCountRaw() uint32 __asm__("runtime_kos_heap_alloc_count_read")
func HeapAllocBytesRaw() uint32 __asm__("runtime_kos_heap_alloc_bytes_read")
func HeapFreeCountRaw() uint32 __asm__("runtime_kos_heap_free_count_read")
func HeapReallocCountRaw() uint32 __asm__("runtime_kos_heap_realloc_count_read")
func HeapReallocBytesRaw() uint32 __asm__("runtime_kos_heap_realloc_bytes_read")
func GCAllocCountRaw() uint32 __asm__("runtime_gc_alloc_count_read")
func GCAllocBytesRaw() uint32 __asm__("runtime_gc_alloc_bytes_read")
func GCLiveBytesRaw() uint32 __asm__("runtime_gc_live_bytes_read")
func GCThresholdRaw() uint32 __asm__("runtime_gc_threshold_read")
func GCCollectionCountRaw() uint32 __asm__("runtime_gc_collection_count_read")
func GCPollRetryRaw() uint32 __asm__("runtime_gc_poll_retry_read")

// Function 70 - file system interface with long names support.
func FileSystem(request *FileSystemRequest, secondary *uint32) int

// Function 80 - file system interface with parameter of encoding.
func FileSystemEncoded(request *EncodedFileSystemRequest, secondary *uint32) int

// Function 54, subfunction 0 - get clipboard slot count.
func ClipboardSlotCountRaw() int

// Function 54, subfunction 1 - get pointer to clipboard slot data.
func ClipboardSlotDataRaw(slot int) uint32

// Function 54, subfunction 2 - write data to the clipboard.
func ClipboardWriteRaw(size uint32, data *byte) int

// Function 54, subfunction 3 - delete last clipboard slot.
func ClipboardDeleteLastRaw() int

// Function 54, subfunction 4 - unlock clipboard buffer.
func ClipboardUnlockBufferRaw() int

// Function 77, subfunction 10 - read from a file handle.
// The current kernel contract documents pipe descriptors on this path.
func PosixReadRaw(fd uint32, buffer *byte, size uint32) int

// Function 77, subfunction 11 - write to a file handle.
// The current kernel contract documents pipe descriptors on this path.
func PosixWriteRaw(fd uint32, buffer *byte, size uint32) int

// Function 77, subfunction 13 - create a pipe and return two file handles.
func PosixPipe2Raw(pipefd *uint32, flags uint32) int

// Function 30, subfunction 5 - get current folder with explicit encoding.
func GetCurrentFolderRaw(buffer *byte, size uint32, encoding StringEncoding) int

// Function 17 - get the identifier of the pressed button.
func GetButtonID() int

// Function 8 - define/delete button.
func CreateButton(x int, y int, width int, height int, id int, color uint32)

// Function -1 - terminate thread/process.
func ExitRaw()
func RuntimeExitProcessRaw() __asm__("runtime_kolibri_exit_process")
func RuntimeExitThreadRaw() __asm__("runtime_kolibri_exit_thread")

// Function 12 - begin/end window redraw.
func Redraw(mode int)

// Function 0 - define and draw the window.
func Window(x int, y int, width int, height int, title string)

// Function 71, subfunction 2 - set window caption with explicit encoding.
func SetCaption(title string)

// Function 71, subfunction 1 - set window caption using an inline encoding prefix.
func SetCaptionWithPrefix(encoding StringEncoding, title string)

// Function 72, subfunction 1 - send a key or button event to the active window.
func SendMessage(event int, param uint32) int

// Function 37, subfunction 0 - get screen coordinates of the mouse.
func GetMouseScreenPosition() uint32

// Function 37, subfunction 1 - get mouse coordinates relative to the window.
func GetMouseWindowPosition() uint32

// Function 37, subfunction 2 - get states of the mouse buttons.
func GetMouseButtonState() uint32

// Function 37, subfunction 3 - get states and events of the mouse buttons.
func GetMouseButtonEventState() uint32

// Function 37, subfunction 4 - load a cursor from memory/file descriptor arguments.
func LoadCursorRaw(data uint32, descriptor uint32) uint32

// Function 37, subfunction 5 - set the current thread window cursor.
func SetCursorRaw(handle uint32) uint32

// Function 37, subfunction 6 - delete a cursor previously loaded by the thread.
func DeleteCursorRaw(handle uint32)

// Function 37, subfunction 7 - get scroll data.
func GetMouseScrollData() uint32

// Function 35 - read the color of a pixel on the screen.
// offset = y*screen_width + x
func GetPixelColorFromScreenRaw(offset int) uint32

// Function 37, subfunction 8 - load a cursor from a path with explicit string encoding.
func LoadCursorWithEncoding(encoding StringEncoding, path string) uint32

// Function 4 - draw text string.
func WriteText(x int, y int, color uint32, text string)

// Function 4 - draw text string with explicit flags/color and optional buffer.
// flagsColor combines ABFFCSSS in the high byte with 0x00RRGGBB in the low 24 bits.
// When C=1, buffer points to the user buffer header (width, height, pixels).
func WriteTextEx(x int, y int, flagsColor uint32, text string, buffer *byte)

// Function 38 - draw line.
func DrawLine(x1 int, y1 int, x2 int, y2 int, color uint32)

// Function 13 - draw rectangle.
func DrawBar(x int, y int, width int, height int, color uint32)

// Function 65 - draw image with palette in the window.
func PutPaletteImage(buffer *byte, width int, height int, x int, y int, bpp int, palette *uint32, rowOffset int)

// Function 14 - get screen size.
func GetScreenSize() uint32

// Function 63 - work with the debug board, write byte helper.
func DebugOutHex(uint32)

// Function 63 - work with the debug board, write byte helper.
func DebugOutChar(byte)

// Function 63 - work with the debug board, write string helper.
func DebugOutStr(string)

// Function 63 - work with the debug board, read byte helper.
// Low byte contains the byte value, bit 8 is set when a byte was read.
func DebugReadRaw() uint32

// Function 69, subfunction 0 - define the debugger message area.
func DebugSetMessageAreaRaw(*byte)

// Function 69, subfunction 1 - read registers of the suspended debugged thread.
func DebugGetRegistersRaw(thread uint32, buffer *byte)

// Function 69, subfunction 4 - suspend the debugged thread.
func DebugSuspendRaw(thread uint32)

// Function 69, subfunction 5 - resume the suspended debugged thread.
func DebugResumeRaw(thread uint32)

// Function 69, subfunction 6 - read memory from the debugged process.
func DebugReadMemoryRaw(thread uint32, remoteAddress uint32, buffer *byte, size uint32) int32

// Direct OUT instruction helper for previously reserved ports.
func PortWriteByteRaw(port uint32, value byte)

func Pointer2byteSlice(ptr uint32) *[]byte __asm__("__unsafe_get_addr")
