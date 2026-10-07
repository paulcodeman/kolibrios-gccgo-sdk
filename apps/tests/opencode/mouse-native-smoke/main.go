package main

import (
	"bufio"
	"fmt"
	"kos"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func main() {
	console, ok := kos.OpenConsole("Original libvterm native mouse input")
	if !ok {
		panic("console")
	}
	defer console.Exit(false)
	entry := console.ExportTable().Lookup("con_enable_vterm")
	if !entry.Valid() || kos.CallStdcall0Raw(uint32(entry)) != 1 {
		panic("vterm")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	fmt.Fprint(os.Stdout, "\x1b[?1002h\x1b[?1006h")
	kos.DebugString("OPENCODE_MOUSE_NATIVE_READY\n")
	reader := bufio.NewReader(os.Stdin)
	pressed, released, wheel := false, false, false
	for !pressed || !released || !wheel {
		data := ""
		for {
			value, err := reader.ReadByte()
			if err != nil {
				panic(err)
			}
			data += string([]byte{value})
			if value == 'M' || value == 'm' {
				break
			}
			if len(data) > 80 {
				panic("unbounded mouse sequence")
			}
		}
		if !strings.HasPrefix(data, "\x1b[<") {
			panic("original SGR mouse protocol")
		}
		fields := strings.Split(data[3:len(data)-1], ";")
		if len(fields) != 3 {
			panic("mouse fields")
		}
		code, err := strconv.Atoi(fields[0])
		if err != nil {
			panic(err)
		}
		x, _ := strconv.Atoi(fields[1])
		y, _ := strconv.Atoi(fields[2])
		if x < 1 || x > 80 || y < 1 || y > 25 {
			panic("native cell coordinates")
		}
		switch code {
		case 0:
			if data[len(data)-1] == 'M' {
				pressed = true
			} else {
				released = true
			}
		case 64, 65:
			wheel = true
		}
		kos.DebugString(fmt.Sprintf("OPENCODE_MOUSE_REPORT %d %d %d %c\n", code, x, y, data[len(data)-1]))
	}
	fmt.Fprint(os.Stdout, "\x1b[?1002l\x1b[?1006l\nNative press, release and wheel through unchanged libvterm PASS\n")
	kos.DebugString("OPENCODE_MOUSE_NATIVE_PASS\n")
}
