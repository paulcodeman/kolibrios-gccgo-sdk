package main

import (
	"kos"
	"os"
	"strconv"
	"time"
	"unsafe"
)

func main() {
	fail := func(message string) { kos.DebugString("OPENCODE_VTERM_FAIL: " + message + "\n"); panic(message) }
	console, ok := kos.OpenConsole("Upstream libvterm native checks")
	if !ok {
		fail("console initialization")
	}
	defer console.Exit(false)
	table := console.ExportTable()
	enable := table.Lookup("con_enable_vterm")
	cell := table.Lookup("con_get_cell")
	cursor := table.Lookup("con_get_cursor_pos")
	if !enable.Valid() || !cell.Valid() || !cursor.Valid() {
		fail("vterm exports")
	}
	if kos.CallStdcall0Raw(uint32(enable)) != 1 {
		fail("vterm initialization")
	}
	kos.DebugString("OPENCODE_VTERM_INITIALIZED\n")
	var fontData []byte
	if path := os.Getenv("KOLIBRI_TEST_UNICODE_FONT"); path != "" {
		var err error
		fontData, err = os.ReadFile(path)
		if err != nil { fail("read Unicode font") }
		setFont := table.Lookup("con_set_unicode_font")
		glyphWidth := table.Lookup("con_unicode_glyph_width")
		if !setFont.Valid() || !glyphWidth.Valid() { fail("Unicode font exports") }
		if kos.CallStdcall2Raw(uint32(setFont), uint32(uintptr(unsafe.Pointer(&fontData[0]))), uint32(len(fontData))) != 1 { fail("Unicode font initialization") }
		for _, code := range []uint32{0x1f600, 0x4e2d} {
			if kos.CallStdcall1Raw(uint32(glyphWidth), code) != 16 { fail("wide Unicode font glyph") }
		}
		if kos.CallStdcall1Raw(uint32(glyphWidth), 0x256d) != 8 { fail("Unicode rounded border glyph") }
	}
	checkCursor := func(wantX, wantY uint32) {
		var x, y uint32
		kos.CallStdcall2VoidRaw(uint32(cursor), uint32(uintptr(unsafe.Pointer(&x))), uint32(uintptr(unsafe.Pointer(&y))))
		if x != wantX || y != wantY {
			panic("vterm cursor geometry")
		}
	}
	readCell := func(x, y uint32) [4]uint32 {
		var result [4]uint32
		if kos.CallStdcall3Raw(uint32(cell), x, y, uint32(uintptr(unsafe.Pointer(&result)))) != 1 {
			panic("vterm cell access")
		}
		return result
	}
	console.WriteString("\xd0")
	checkCursor(0, 0)
	console.WriteString("\x96")
	checkCursor(1, 0)
	if readCell(0, 0)[0] != 0x416 {
		panic("original Unicode codepoint")
	}
	console.WriteString("\x1b[38;5;196m\x1b[48;5;22mX")
	colors := readCell(1, 0)
	if colors[1] != 0xff0000 || colors[2] != 0x005f00 {
		panic("ANSI256 color")
	}
	console.WriteString("\x1b[2;6H\x1b[2D")
	checkCursor(3, 1)
	console.WriteString("\x1b[?1049hALT\x1b[?1049l")
	checkCursor(3, 1)
	if readCell(0, 0)[0] != 0x416 {
		panic("alternate screen restore")
	}
	console.WriteString("😀")
	checkCursor(5, 1)
	if readCell(3, 1)[3] != 2 {
		panic("wide Unicode cell")
	}
	console.WriteString("\x1b[6n")
	input := table.Lookup("con_get_input")
	if !input.Valid() || !console.KeyHit() {
		panic("terminal cursor response readiness")
	}
	var response [32]byte
	n := kos.CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&response))), 32)
	if n != 6 || string(response[:n]) != "\x1b[2;6R" {
		panic("terminal cursor response")
	}
	console.WriteString("\x1b[0m\x1b[2J\x1b[H╭──────────────────────╮\r\n│ Привет, KolibriOS!    │\r\n╰──────────────────────╯\r\n")
	for i := uint32(0); i < 256; i++ {
		console.WriteString("\x1b[48;5;")
		console.WriteString(strconv.Itoa(int(i)))
		console.WriteString("m ")
		if i%32 == 31 {
			console.WriteString("\x1b[0m\r\n")
		}
	}
	console.WriteString("\x1b[0mNative upstream libvterm PASS\r\n")
	if len(fontData) != 0 {
		console.WriteString("Unchanged Unifont: 😀 中文 日本語 한국어 e\u0301\r\n")
		kos.DebugString("OPENCODE_CONSOLE_UNIFONT_PASS\n")
	}
	if os.Getenv("KOLIBRI_TEST_UNICODE_INPUT") == "1" {
		// sysfuncs.txt: 26/2 copies the normal 128-byte keyboard layout;
		// 21/2 installs it. Restore it after the isolated QEMU input check.
		var layout [128]byte
		regs := kos.SyscallRegs{EAX: 26, EBX: 2, ECX: 1, EDX: uint32(uintptr(unsafe.Pointer(&layout[0])))}
		kos.SyscallRaw(&regs)
		original := layout
		layout[0x1e], layout[0x30] = 0xa0, 0x81 // native CP866 а, Б
		regs = kos.SyscallRegs{EAX: 21, EBX: 2, ECX: 1, EDX: uint32(uintptr(unsafe.Pointer(&layout[0])))}
		kos.SyscallRaw(&regs)
		kos.DebugString("OPENCODE_UNICODE_INPUT_READY\n")
		var received []byte
		for len(received) < 4 {
			if console.KeyHit() {
				var chunk [32]byte
				n := kos.CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&chunk))), 32)
				received = append(received, chunk[:n]...)
			} else { time.Sleep(10*time.Millisecond) }
		}
		regs = kos.SyscallRegs{EAX: 21, EBX: 2, ECX: 1, EDX: uint32(uintptr(unsafe.Pointer(&original[0])))}
		kos.SyscallRaw(&regs)
		if string(received) != "аБ" { fail("original keyboard UTF-8 input") }
		console.WriteString("Native UTF-8 keyboard: " + string(received) + "\r\n")
		kos.DebugString("OPENCODE_CONSOLE_UNICODE_INPUT_PASS\n")
	}
	kos.DebugString("OPENCODE_CONSOLE_VTERM_PASS\n")
}
