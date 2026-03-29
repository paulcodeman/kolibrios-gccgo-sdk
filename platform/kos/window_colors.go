package kos

import "unsafe"

type WindowColors struct {
	Frame          Color
	Grab           Color
	WorkDark       Color
	WorkLight      Color
	GrabText       Color
	Work           Color
	WorkButton     Color
	WorkButtonText Color
	WorkText       Color
	Graph          Color
}

func StandardWindowColors() WindowColors {
	var table [10]uint32
	regs := SyscallRegs{
		EAX: 48,
		EBX: 3,
		ECX: pointerValue((*byte)(unsafe.Pointer(&table[0]))),
		EDX: uint32(len(table) * 4),
	}
	SyscallRaw(&regs)
	return WindowColors{
		Frame:          Color(table[0]),
		Grab:           Color(table[1]),
		WorkDark:       Color(table[2]),
		WorkLight:      Color(table[3]),
		GrabText:       Color(table[4]),
		Work:           Color(table[5]),
		WorkButton:     Color(table[6]),
		WorkButtonText: Color(table[7]),
		WorkText:       Color(table[8]),
		Graph:          Color(table[9]),
	}
}
