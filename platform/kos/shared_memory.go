//go:build kolibrios && gccgo

package kos

const (
	SharedMemoryOpen       uint32 = 0x00
	SharedMemoryOpenAlways uint32 = 0x04
	SharedMemoryCreate     uint32 = 0x08
	SharedMemoryRead       uint32 = 0x00
	SharedMemoryWrite      uint32 = 0x01
)

func sharedMemoryNameValid(name string) bool {
	if len(name) == 0 || len(name) > 30 {
		return false
	}
	for i := 0; i < len(name); i++ {
		if name[i] == 0 {
			return false
		}
	}
	return true
}

// OpenNamedMemory calls syscall 68/22 as documented in sysfuncs.txt.
// On success address is nonzero: result is zero for a newly created area,
// or its byte size for an existing area. On failure result is an OS error.
// Mappings belong to the OS thread that opened them. Keep that thread locked
// for the mapping lifetime and call CloseNamedMemory on the same thread.
func OpenNamedMemory(name string, size, flags uint32) (address uintptr, result uint32) {
	mode := flags &^ SharedMemoryWrite
	if !sharedMemoryNameValid(name) || (mode != SharedMemoryOpen && mode != SharedMemoryOpenAlways && mode != SharedMemoryCreate) || (mode != SharedMemoryOpen && size == 0) {
		return 0, 33 // E_PARAM
	}
	ptr, addr := stringAddress(name)
	if ptr == nil {
		return 0, 30
	} // E_NOMEM
	regs := SyscallRegs{EAX: 68, EBX: 22, ECX: addr, EDX: size, ESI: flags}
	SyscallRaw(&regs)
	freeCString(ptr)
	return uintptr(regs.EAX), regs.EDX
}

// CloseNamedMemory calls syscall 68/23. The kernel supplies no result code.
// The area disappears when the last opening OS thread closes it or exits.
func CloseNamedMemory(name string) {
	if !sharedMemoryNameValid(name) {
		return
	}
	ptr, addr := stringAddress(name)
	if ptr == nil {
		return
	}
	regs := SyscallRegs{EAX: 68, EBX: 23, ECX: addr}
	SyscallRaw(&regs)
	freeCString(ptr)
}
