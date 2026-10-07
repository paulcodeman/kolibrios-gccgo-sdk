package main

import (
	"kos"
	"unsafe"
)

func main() {
	console, ok := kos.OpenConsole("Native UTF-8 and VT checks")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	table := console.ExportTable()
	mode := table.Lookup("con_set_output_mode")
	cursor := table.Lookup("con_get_cursor_pos")
	if !mode.Valid() || !cursor.Valid() {
		panic("terminal exports")
	}
	kos.CallStdcall1Raw(uint32(mode), 1)
	check := func(wantX, wantY uint32) {
		var x, y uint32
		kos.CallStdcall2VoidRaw(uint32(cursor), uint32(uintptr(unsafe.Pointer(&x))), uint32(uintptr(unsafe.Pointer(&y))))
		if x != wantX || y != wantY {
			panic("UTF-8 cell width or VT cursor movement")
		}
	}
	console.WriteString("┌─┐Привет")
	check(9, 0)
	console.WriteString("\xd0")
	check(9, 0)
	console.WriteString("\x96")
	check(10, 0)
	console.WriteString("\xe0\x80\x80\xe2A\xed\xa0\x80")
	check(14, 0)
	console.WriteString("\x1b[2;5H")
	check(4, 1)
	console.WriteString("\x1b[2D")
	check(2, 1)
	console.WriteString("\x1b[12G")
	check(11, 1)
	// An unknown OSC must return without corrupting the caller's stack.
	console.WriteString("\x1b]999;ignored\a")
	check(11, 1)
	console.WriteString("\x1b[2J\x1b[H╭──────────────────────╮\n│ Привет, KolibriOS!    │\n╰──────────────────────╯\n")
	console.WriteString("Native UTF-8 and VT PASS\n")
	kos.DebugString("OPENCODE_CONSOLE_UNICODE_PASS")
}
