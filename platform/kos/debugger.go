package kos

import "unsafe"

const DebugMessageAreaCapacity = 512

type DebugMessageArea struct {
	Size int32
	Used uint32
	Data [DebugMessageAreaCapacity]byte
}

type DebugRegisters struct {
	EIP    uint32
	EFLAGS uint32
	EAX    uint32
	ECX    uint32
	EDX    uint32
	EBX    uint32
	ESP    uint32
	EBP    uint32
	ESI    uint32
	EDI    uint32
}

func (area *DebugMessageArea) Reset() {
	if area == nil {
		return
	}
	area.Size = DebugMessageAreaCapacity
	area.Used = 0
	for index := range area.Data {
		area.Data[index] = 0
	}
}

func DebugSetMessageArea(area *DebugMessageArea) {
	if area == nil {
		return
	}
	DebugSetMessageAreaRaw((*byte)(unsafe.Pointer(area)))
}

func DebugGetRegisters(thread uint32) DebugRegisters {
	var regs DebugRegisters
	DebugGetRegistersRaw(thread, (*byte)(unsafe.Pointer(&regs)))
	return regs
}

func DebugSuspend(thread uint32) {
	DebugSuspendRaw(thread)
}

func DebugResume(thread uint32) {
	DebugResumeRaw(thread)
}

func DebugReadMemory(thread uint32, remoteAddress uint32, buffer []byte) int {
	if len(buffer) == 0 {
		return 0
	}
	return int(DebugReadMemoryRaw(thread, remoteAddress, &buffer[0], uint32(len(buffer))))
}
