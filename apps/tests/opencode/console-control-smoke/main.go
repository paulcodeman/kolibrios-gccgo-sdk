package main

import (
	"kos"
	"time"
	"unsafe"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode native control events")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	table := console.ExportTable()
	mode := table.Lookup("con_set_input_mode")
	policy := table.Lookup("con_set_signal_mode")
	take := table.Lookup("con_take_interrupts")
	input := table.Lookup("con_get_input")
	if !mode.Valid() || !policy.Valid() || !take.Valid() || !input.Valid() {
		panic("native control-event exports")
	}
	if kos.CallStdcall1Raw(uint32(policy), 1) != 0 {
		panic("initial native control policy")
	}
	deadline := time.Now().Add(15 * time.Second)
	for kos.CallStdcall0Raw(uint32(take)) == 0 {
		if time.Now().After(deadline) {
			panic("canonical Ctrl-C was not captured")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if console.KeyHit() {
		panic("captured Ctrl-C leaked into input")
	}
	console.WriteString("PASS canonical Ctrl-C event\n")
	if kos.CallStdcall1Raw(uint32(mode), 1) != 0 {
		panic("initial native input mode")
	}
	for !console.KeyHit() {
		if time.Now().After(deadline) {
			panic("raw Ctrl-C input was lost")
		}
		time.Sleep(5 * time.Millisecond)
	}
	var buffer [32]byte
	n := kos.CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&buffer[0]))), uint32(len(buffer)))
	if n != 1 || buffer[0] != 3 || kos.CallStdcall0Raw(uint32(take)) != 0 {
		panic("raw Ctrl-C became a signal")
	}
	console.WriteString("PASS raw Ctrl-C byte\n")
	kos.CallStdcall1Raw(uint32(mode), 0)
	kos.CallStdcall1Raw(uint32(policy), 2)
	time.Sleep(4 * time.Second)
	if console.KeyHit() || kos.CallStdcall0Raw(uint32(take)) != 0 {
		panic("ignored Ctrl-C was queued")
	}
	kos.CallStdcall1Raw(uint32(policy), 0)
	console.WriteString("OpenCode native control events PASS\n")
	kos.DebugString("OPENCODE_CONSOLE_CONTROL_PASS")
}
