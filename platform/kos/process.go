// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package kos

const (
	systemFunctionNumber              = 18
	systemTerminateByIdentifier       = 18
	systemGetThreadSlotByIdentifier   = 21
)

// ThreadSlotByIdentifier returns the thread slot for the provided PID/TID.
// The kernel returns 0 for an invalid identifier.
func ThreadSlotByIdentifier(id int) int {
	regs := SyscallRegs{
		EAX: systemFunctionNumber,
		EBX: systemGetThreadSlotByIdentifier,
		ECX: uint32(id),
	}
	SyscallRaw(&regs)
	return int(regs.EAX)
}

// TerminateByIdentifier requests termination of the provided PID/TID.
// It reports false when the kernel rejects the request.
func TerminateByIdentifier(id int) bool {
	regs := SyscallRegs{
		EAX: systemFunctionNumber,
		EBX: systemTerminateByIdentifier,
		ECX: uint32(id),
	}
	SyscallRaw(&regs)
	return int32(regs.EAX) == 0
}
