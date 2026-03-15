// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package kos

// SyscallRegs describes the register frame for int 0x40 syscalls.
// Fill in the inputs, call SyscallRaw, then read outputs from the same struct.
type SyscallRegs struct {
	EAX uint32
	EBX uint32
	ECX uint32
	EDX uint32
	ESI uint32
	EDI uint32
	EBP uint32
}

// SyscallRaw executes int 0x40 with the provided register frame.
func SyscallRaw(regs *SyscallRegs)
