// +build kolibrios,!gccgo

//go:build kolibrios && !gccgo
// +build kolibrios,!gccgo

package kos

import "unsafe"

var emptyCStringByte byte

func Sleep(uint32)
func GetTime() uint32
func GetDate() uint32
func GetTimeCounter() uint32
func GetTimeCounterPro() uint64 { return 0 }
func Event() int
func GetKey() int                                      { return 0 }
func GetControlKeysRaw() uint32                        { return 0 }
func SetKeyboardLayoutRaw(which int, table *byte) int  { return 0 }
func SetKeyboardLanguageRaw(language int) int          { return 0 }
func SetSystemLanguageRaw(language int) int            { return 0 }
func GetKeyboardLayoutRaw(which int, buffer *byte) int { return 0 }
func GetKeyboardLanguageRaw() int                      { return 0 }
func GetSystemLanguageRaw() int                        { return 0 }
func CheckEvent() int
func GetThreadInfo(buffer *byte, slot int) int       { return 0 }
func CreateThreadRaw(entry uint32, stack uint32) int { return 0 }
func GetCurrentThreadSlotRaw() int                   { return 0 }
func ThreadEntryAddrRaw() uint32                     { return 0 }
func WaitEventTimeout(uint32) int
func SetEventMask(uint32) uint32
func SetPortsRaw(mode int, start uint32, end uint32) int
func SetIPCArea(buffer *byte, size uint32) uint32                     { return 0 }
func SendIPCMessage(pid uint32, data *byte, size uint32) uint32       { return 0 }
func FocusWindowBySlot(int)                                           {}
func GetActiveWindowSlotRaw() int                                     { return 0 }
func SetMousePointerPositionRaw(uint32)                               {}
func SimulateMouseButtonsRaw(uint32)                                  {}
func SetWindowLayerBehaviourRaw(pid int, position int) int            { return 0 }
func GetSkinHeight() int                                              { return 0 }
func GetSkinMarginsRaw(vertical *uint32) uint32                       { return 0 }
func GetFontSmoothingRaw() uint32                                     { return 0 }
func SetFontSmoothingRaw(mode uint8)                                  {}
func SetSkin(path string) uint32                                      { return 0 }
func SetSkinWithEncoding(encoding StringEncoding, path string) uint32 { return 0 }
func GetScreenWorkingArea(bottom *uint32) uint32                      { return 0 }
func GetKernelVersion(buffer *byte)                                   {}
func SystemShutdown(uint32) uint32
func GetFreeRAM() uint32                                                        { return 0 }
func GetTotalRAM() uint32                                                       { return 0 }
func InitHeapRaw() uint32                                                       { return 0 }
func HeapAllocRaw(size uint32) uint32                                           { return 0 }
func HeapFreeRaw(ptr uint32) uint32                                             { return 0 }
func HeapReallocRaw(size uint32, ptr uint32) uint32                             { return 0 }
func LoadDLLWithEncoding(encoding StringEncoding, path string) uint32           { return 0 }
func LoadDLL(path string) uint32                                                { return 0 }
func LoadDLLFromCStringRaw(path *byte) uint32                                   { return 0 }
func LookupDLLExportRaw(table uint32, name *byte) uint32                        { return 0 }
func CallStdcall0Raw(proc uint32) uint32                                        { return 0 }
func CallStdcall1Raw(proc uint32, arg0 uint32) uint32                           { return 0 }
func CallStdcall2Raw(proc uint32, arg0 uint32, arg1 uint32) uint32              { return 0 }
func CallStdcall3Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32) uint32 { return 0 }
func CallStdcall4Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32) uint32 {
	return 0
}
func CallStdcall5Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) uint32 {
	return 0
}
func CallStdcall6Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32, arg5 uint32) uint32 {
	return 0
}
func CallStdcall7Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32, arg5 uint32, arg6 uint32) uint32 {
	return 0
}
func CallCdecl2Raw(proc uint32, arg0 uint32, arg1 uint32) uint32 { return 0 }
func CallCdecl5Raw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) uint32 {
	return 0
}
func CallStdcall1VoidRaw(proc uint32, arg0 uint32)              {}
func CallStdcall2VoidRaw(proc uint32, arg0 uint32, arg1 uint32) {}
func CallStdcall5VoidRaw(proc uint32, arg0 uint32, arg1 uint32, arg2 uint32, arg3 uint32, arg4 uint32) {
}
func ConsoleBridgeReadyRaw() uint32                                                        { return 0 }
func ConsoleBridgeSetRaw(table uint32, writeProc uint32, exitProc uint32, getsProc uint32) {}
func ConsoleBridgeClearRaw(table uint32)                                                   {}
func ConsoleBridgeWriteRaw(data uint32, size uint32) uint32                                { return 0 }
func ConsoleBridgeReadLineRaw(data uint32, size uint32) uint32                             { return 0 }
func ConsoleBridgeCloseRaw(closeWindow uint32)                                             {}
func InitDLLLibraryRaw(proc uint32) uint32                                                 { return 0 }
func DialogNoopProcRaw() uint32                                                            { return 0 }
func CStringToStringRaw(ptr uint32) string                                                 { return "" }
func LoaderParametersRaw() uint32                                                          { return 0 }
func LoaderPathRaw() uint32                                                                { return 0 }
func CopyBytesRaw(ptr uint32, size uint32) []byte                                          { return nil }
func ReadUint32Raw(base uint32, offset uint32) uint32                                      { return 0 }
func BootstrapRuntimeHasGCRaw() uint32                                                     { return 0 }
func PollRuntimeGCRaw()                                                                    {}
func HeapAllocCountRaw() uint32                                                            { return 0 }
func HeapAllocBytesRaw() uint32                                                            { return 0 }
func HeapFreeCountRaw() uint32                                                             { return 0 }
func HeapReallocCountRaw() uint32                                                          { return 0 }
func HeapReallocBytesRaw() uint32                                                          { return 0 }
func GCAllocCountRaw() uint32                                                              { return 0 }
func GCAllocBytesRaw() uint32                                                              { return 0 }
func GCLiveBytesRaw() uint32                                                               { return 0 }
func GCThresholdRaw() uint32                                                               { return 0 }
func GCCollectionCountRaw() uint32                                                         { return 0 }
func GCPollRetryRaw() uint32                                                               { return 0 }
func FileSystem(request *FileSystemRequest, secondary *uint32) int                         { return 0 }
func FileSystemEncoded(request *EncodedFileSystemRequest, secondary *uint32) int
func ClipboardSlotCountRaw() int                             { return 0 }
func ClipboardSlotDataRaw(slot int) uint32                   { return 0 }
func ClipboardWriteRaw(size uint32, data *byte) int          { return 0 }
func ClipboardDeleteLastRaw() int                            { return 0 }
func ClipboardUnlockBufferRaw() int                          { return 0 }
func PosixReadRaw(fd uint32, buffer *byte, size uint32) int  { return 0 }
func PosixWriteRaw(fd uint32, buffer *byte, size uint32) int { return 0 }
func PosixPipe2Raw(pipefd *uint32, flags uint32) int         { return 0 }
func GetCurrentFolderRaw(buffer *byte, size uint32, encoding StringEncoding) int
func GetButtonID() int
func CreateButton(x int, y int, width int, height int, id int, color uint32)
func ExitRaw()
func RuntimeExitProcessRaw()
func Redraw(mode int)
func windowRaw(x int, y int, width int, height int, title *byte)

func Window(x int, y int, width int, height int, title string) {
	ptr, _ := stringAddress(title)
	if ptr == nil {
		ptr = &emptyCStringByte
	}
	windowRaw(x, y, width, height, ptr)
	freeCString(ptr)
}

func SetCaption(title string)                                            {}
func SetCaptionWithPrefix(encoding StringEncoding, title string)         {}
func SendMessage(event int, param uint32) int                            { return 0 }
func GetMouseScreenPosition() uint32                                     { return 0 }
func GetMouseWindowPosition() uint32                                     { return 0 }
func GetMouseButtonState() uint32                                        { return 0 }
func GetMouseButtonEventState() uint32                                   { return 0 }
func LoadCursorRaw(data uint32, descriptor uint32) uint32                { return 0 }
func SetCursorRaw(handle uint32) uint32                                  { return 0 }
func DeleteCursorRaw(handle uint32)                                      {}
func GetMouseScrollData() uint32                                         { return 0 }
func GetPixelColorFromScreenRaw(offset int) uint32                       { return 0 }
func LoadCursorWithEncoding(encoding StringEncoding, path string) uint32 { return 0 }
func writeTextRaw(x int, y int, color uint32, text *byte, textLen int)

func WriteText(x int, y int, color uint32, text string) {
	if text == "" {
		return
	}
	writeTextRaw(x, y, color, unsafe.StringData(text), len(text))
}

func WriteTextEx(x int, y int, flagsColor uint32, text string, buffer *byte) {
	if text == "" {
		return
	}
	writeTextRaw(x, y, flagsColor, unsafe.StringData(text), len(text))
}

func DrawLine(x1 int, y1 int, x2 int, y2 int, color uint32)
func DrawBar(x int, y int, width int, height int, color uint32)
func PutPaletteImage(buffer *byte, width int, height int, x int, y int, bpp int, palette *uint32, rowOffset int) {
}
func GetScreenSize() uint32                                  { return 0 }
func DebugOutHex(uint32)                                     {}
func DebugOutChar(byte)                                      {}
func DebugOutStr(string)                                     {}
func DebugReadRaw() uint32                                   { return 0 }
func DebugSetMessageAreaRaw(*byte)                           {}
func DebugGetRegistersRaw(uint32, *byte)                     {}
func DebugSuspendRaw(uint32)                                 {}
func DebugResumeRaw(uint32)                                  {}
func DebugReadMemoryRaw(uint32, uint32, *byte, uint32) int32 { return 0 }
func PortWriteByteRaw(port uint32, value byte)               {}

func allocCString(value string) *byte {
	if value == "" {
		return nil
	}
	data := append([]byte(value), 0)
	return &data[0]
}

func freeCString(ptr *byte) {}

func pointerValue(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func stringAddress(value string) (ptr *byte, addr uint32) {
	ptr = allocCString(value)
	if ptr == nil {
		return nil, 0
	}
	return ptr, pointerValue(ptr)
}

func byteSliceAddress(data []byte) uint32 {
	if len(data) == 0 {
		return 0
	}
	return pointerValue(&data[0])
}

func Pointer2byteSlice(ptr uint32) *[]byte {
	return (*[]byte)(unsafe.Pointer(uintptr(ptr)))
}
