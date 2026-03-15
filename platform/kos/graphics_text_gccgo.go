//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kos

import (
	"unicode/utf8"
	"unsafe"
)

func writeTextWithLength(x int, y int, flagsColor uint32, text string, buffer *byte) {
	if text == "" {
		return
	}
	textPtr := stringDataPointer(text)
	if textPtr == 0 {
		return
	}
	length := textLengthForFlags(text, flagsColor)
	regs := SyscallRegs{
		EAX: 4,
		EBX: uint32(uint16(x))<<16 | uint32(uint16(y)),
		ECX: flagsColor,
		EDX: textPtr,
		ESI: uint32(length),
	}
	if buffer != nil {
		regs.EDI = pointerValue(buffer)
	}
	SyscallRaw(&regs)
}

func textLengthForFlags(text string, flagsColor uint32) int {
	flags := byte(flagsColor >> 24)
	if flags&textFlagUTF8 == textFlagUTF8 {
		return utf8.RuneCountInString(text)
	}
	return len(text)
}

type stringHeader struct {
	Data uintptr
	Len  int
}

func stringDataPointer(value string) uint32 {
	if value == "" {
		return 0
	}
	header := (*stringHeader)(unsafe.Pointer(&value))
	return uint32(header.Data)
}
