package kos

type ShutdownMode uint32

const (
	ShutdownPowerOff      ShutdownMode = 2
	ShutdownReboot        ShutdownMode = 3
	ShutdownRestartKernel ShutdownMode = 4
)

func KernelVersion() KernelVersionInfo {
	var buffer [16]byte

	GetKernelVersion(&buffer[0])
	return KernelVersionInfo{
		Major:    buffer[0],
		Minor:    buffer[1],
		Patch:    buffer[2],
		Build:    buffer[3],
		DebugTag: buffer[4],
		ABIMinor: buffer[5],
		ABIMajor: littleEndianUint16(buffer[:], 6),
		CommitID: littleEndianUint32(buffer[:], 8),
	}
}

func FreeRAMKB() uint32 {
	return GetFreeRAM()
}

func TotalRAMKB() uint32 {
	return GetTotalRAM()
}

func Shutdown(mode ShutdownMode) bool {
	return SystemShutdown(uint32(mode)) == 0
}

func FocusWindowSlot(slot int) {
	FocusWindowBySlot(slot)
}

func ActiveWindowSlot() int {
	return GetActiveWindowSlotRaw()
}

func PowerOff() bool {
	return Shutdown(ShutdownPowerOff)
}

func Reboot() bool {
	return Shutdown(ShutdownReboot)
}

func RestartKernel() bool {
	return Shutdown(ShutdownRestartKernel)
}
