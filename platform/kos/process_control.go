//go:build kolibrios && gccgo

package kos

import "runtime"

// The private SDK control area does not change the native process/syscall ABI.
// Its unique name comes from StartProcess's atomically created storage folder.
func ProcessControlName(statusPath string) string {
	end := len(statusPath)
	for end > 0 && statusPath[end-1] != '/' { end-- }
	if end == 0 { return "" }
	start := end-1
	for start > 0 && statusPath[start-1] != '/' { start-- }
	name := statusPath[start:end-1]
	if len(name) == 0 || len(name) > 30 { return "" }
	return name
}

func installProcessThreadControl(address uintptr) __asm__("runtime_kolibri_process_thread_control")

func initializeChildProcessControl(startup *ProcessStartup) {
	name := ProcessControlName(startup.StatusPath)
	if name == "" { return }
	// A named-area opening belongs to its native thread, which must remain
	// alive while the process's runtime workers publish their identifiers.
	runtime.LockOSThread()
	address, size := OpenNamedMemory(name, 0, SharedMemoryOpen | SharedMemoryWrite)
	if address == 0 || size < 4096 {
		runtime.UnlockOSThread()
		return // Older SDK parents do not create the optional area.
	}
	installProcessThreadControl(address)
}
