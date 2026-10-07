package kos

import "unsafe"

// sysfuncs.txt: socketpair 75/10 returns descriptors in EAX and EBX.
// These wrappers use the existing SyscallRaw assembly register bridge.
func CreateLocalSocketPair() (first, second uint32, errorCode uint32) {
	registers := SyscallRegs{EAX: 75, EBX: 10}
	SyscallRaw(&registers)
	if int32(registers.EAX) == -1 {
		return 0, 0, registers.EBX
	}
	return registers.EAX, registers.EBX, 0
}

const LocalSocketDontWait uint32 = 0x40 // MSG_DONTWAIT, upstream network/stack.inc.
const LocalSocketWouldBlock uint32 = 6  // EWOULDBLOCK, upstream network/stack.inc.

// sysfuncs.txt 75/6,7: ECX descriptor, EDX buffer, ESI length, EDI flags;
// EAX byte count (-1 on error), EBX native socket error on failure.
func localSocketIO(operation, descriptor uint32, buffer []byte, flags uint32) (int, uint32) {
	var empty byte
	address := uint32(uintptr(unsafe.Pointer(&empty)))
	if len(buffer) > 0 {
		address = uint32(uintptr(unsafe.Pointer(&buffer[0])))
	}
	registers := SyscallRegs{EAX: 75, EBX: operation, ECX: descriptor, EDX: address, ESI: uint32(len(buffer)), EDI: flags}
	SyscallRaw(&registers)
	if int32(registers.EAX) == -1 {
		return -1, registers.EBX
	}
	return int(registers.EAX), 0
}

func ReadLocalSocket(descriptor uint32, buffer []byte, flags uint32) (int, uint32) {
	return localSocketIO(7, descriptor, buffer, flags)
}
func WriteLocalSocket(descriptor uint32, buffer []byte, flags uint32) (int, uint32) {
	return localSocketIO(6, descriptor, buffer, flags)
}

// sysfuncs.txt: close 75/1 takes ECX descriptor, returns EAX=-1 on error.
func CloseLocalSocket(descriptor uint32) uint32 {
	registers := SyscallRegs{EAX: 75, EBX: 1, ECX: descriptor}
	SyscallRaw(&registers)
	if int32(registers.EAX) == -1 {
		return registers.EBX
	}
	return 0
}

// SDK child startup encodes socket descriptors separately from kernel
// syscall-77 file handles; they are different descriptor namespaces.
func LocalSocketStartupArgument(descriptor uint32) string {
	const digits = "0123456789abcdef"
	data := []byte("@L00000000")
	for i := 0; i < 8; i++ {
		data[9-i] = digits[descriptor&15]
		descriptor >>= 4
	}
	return string(data)
}

func LocalSocketStartupDescriptor(value string) (uint32, bool) {
	if len(value) != 10 || value[:2] != "@L" {
		return 0, false
	}
	var descriptor uint32
	for i := 2; i < 10; i++ {
		var digit byte
		if value[i] >= '0' && value[i] <= '9' {
			digit = value[i] - '0'
		} else if value[i] >= 'a' && value[i] <= 'f' {
			digit = value[i] - 'a' + 10
		} else {
			return 0, false
		}
		descriptor = descriptor<<4 | uint32(digit)
	}
	return descriptor, true
}
