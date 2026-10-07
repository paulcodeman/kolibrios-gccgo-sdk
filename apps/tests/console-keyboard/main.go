package main

import (
	"fmt"
	"kos"
	"os"
	"unsafe"
)

func main() {
	console, ok := kos.OpenConsole("Console keyboard regression")
	if !ok {
		panic("console unavailable")
	}
	defer console.Exit(false)
	table := console.ExportTable()
	report := ""
	check := func(name string, actual, expected string) {
		report += fmt.Sprintf("%s: got %x expected %x\n", name, actual, expected)
		_ = os.WriteFile("/hd0/1/KEYBOARD.txt", []byte(report), 0600)
		if actual != expected {
			panic(name)
		}
	}
	mode := table.Lookup("con_set_output_mode")
	kos.CallStdcall1Raw(uint32(mode), 0)
	check("legacy con_getch2 Backspace", string([]byte{byte(console.Getch2())}), "\x08")
	input := table.Lookup("con_get_input")
	read := func() string {
		var buffer [32]byte
		n := kos.CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&buffer[0]))), uint32(len(buffer)))
		return string(buffer[:n])
	}
	check("legacy con_get_input Backspace", read(), "\x08")
	if kos.CallStdcall0Raw(uint32(table.Lookup("con_enable_vterm"))) == 0 {
		panic("vterm unavailable")
	}
	check("terminal Backspace", read(), "\x7f")
	check("terminal Ctrl-H", read(), "\x08")
	check("terminal Delete", read(), "\x1b[3~")
	check("terminal Shift-Backspace", read(), "\x7f")
	check("terminal Enter", read(), "\x0d")
	// Legacy line editing must still use ASCII BS, including after vterm.
	kos.CallStdcall1Raw(uint32(mode), 0)
	buffer := make([]byte, 32)
	n, err := console.ReadLine(buffer)
	if err != nil {
		panic(err)
	}
	check("legacy con_gets edit abc Backspace Enter", string(buffer[:n]), "ab\n")
	report += "PASS: terminal editing and legacy console input\n"
	_ = os.WriteFile("/hd0/1/KEYBOARD.txt", []byte(report), 0600)
	console.WriteString(report)
}
