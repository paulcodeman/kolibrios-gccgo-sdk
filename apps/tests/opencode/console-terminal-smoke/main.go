package main

import (
	"kos"
	"time"
	"unsafe"
)

func main() {
	console, ok := kos.LoadConsole()
	if !ok || !console.Init(83, 27, 83, 300, "OpenCode native terminal check") {
		panic("native console initialization")
	}
	defer console.Exit(false)
	size := console.ExportTable().Lookup("con_get_size")
	input := console.ExportTable().Lookup("con_get_input")
	if !size.Valid() || !input.Valid() {
		panic("native terminal exports")
	}
	var columns, rows uint32
	kos.CallStdcall2VoidRaw(uint32(size), uint32(uintptr(unsafe.Pointer(&columns))), uint32(uintptr(unsafe.Pointer(&rows))))
	if columns != 83 || rows != 27 {
		panic("actual terminal dimensions")
	}
	console.WriteString("Waiting for raw ASCII, arrow, Enter and Ctrl-C...\n")
	var received []byte
	deadline := time.Now().Add(20 * time.Second)
	for len(received) < 6 && time.Now().Before(deadline) {
		if !console.KeyHit() {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		var buffer [32]byte
		n := kos.CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&buffer[0]))), uint32(len(buffer)))
		if n > uint32(len(buffer)) {
			panic("native input length")
		}
		received = append(received, buffer[:n]...)
	}
	if string(received) != "a\x1b[A\r\x03" {
		kos.DebugString("Raw terminal bytes: ")
		for _, b := range received {
			const hex = "0123456789abcdef"
			kos.DebugString(string([]byte{hex[b>>4], hex[b&15], ' '}))
		}
		panic("native terminal input")
	}
	console.WriteString("OpenCode native terminal PASS\n")
	kos.DebugString("OPENCODE_CONSOLE_TERMINAL_PASS")
}
