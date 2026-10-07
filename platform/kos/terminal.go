package kos

import (
	"sync"
	"unsafe"
)

var ErrConsoleClosed = &consoleError{text: "console window closed"}
var ErrConsoleReadCanceled = &consoleError{text: "console read canceled"}

var consoleInputState struct {
	sync.Mutex
	raw     bool
	pending []byte
}

func ConsoleInputRaw() bool {
	consoleInputState.Lock()
	defer consoleInputState.Unlock()
	return consoleInputState.raw
}

func SetConsoleInputRaw(raw bool) (bool, error) {
	console, ok := ActiveConsole()
	if !ok || !console.keyHitProc.Valid() || !console.table.Lookup("con_get_input").Valid() {
		return false, &consoleError{text: "native raw console input unavailable"}
	}
	consoleInputState.Lock()
	defer consoleInputState.Unlock()
	old := consoleInputState.raw
	// New native libraries need the mode too, so canonical Ctrl-C capture
	// cannot consume bytes intended for the original raw terminal parser.
	if proc := console.table.Lookup("con_set_input_mode"); proc.Valid() {
		var mode uint32
		if raw {
			mode = 1
		}
		CallStdcall1Raw(uint32(proc), mode)
	}
	consoleInputState.raw = raw
	return old, nil
}

func ActiveConsoleSize() (columns, rows int, err error) {
	console, ok := ActiveConsole()
	if !ok {
		return 0, 0, &consoleError{text: "active console unavailable"}
	}
	proc := console.table.Lookup("con_get_size")
	if !proc.Valid() {
		return 0, 0, &consoleError{text: "native terminal-size query unavailable"}
	}
	var width, height uint32
	CallStdcall2VoidRaw(uint32(proc), uint32(uintptr(unsafe.Pointer(&width))), uint32(uintptr(unsafe.Pointer(&height))))
	if width == 0 || height == 0 || width > 0x7fffffff || height > 0x7fffffff {
		return 0, 0, &consoleError{text: "invalid native console dimensions"}
	}
	return int(width), int(height), nil
}

// TryReadActiveConsoleInput consumes currently available native terminal bytes.
// Zero with no error means no input is available. Small reads retain the rest
// of the native escape sequence for the next call.
func TryReadActiveConsoleInput(buffer []byte, cancel <-chan struct{}) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	consoleInputState.Lock()
	defer consoleInputState.Unlock()
	select {
	case <-cancel:
		return 0, ErrConsoleReadCanceled
	default:
	}
	console, ok := ActiveConsole()
	if !ok {
		return 0, ErrConsoleClosed
	}
	if len(consoleInputState.pending) == 0 {
		if !console.KeyHit() {
			return 0, nil
		}
		flags := console.table.Lookup("con_get_flags")
		if flags.Valid() && CallStdcall0Raw(uint32(flags))&0x200 != 0 {
			return 0, ErrConsoleClosed
		}
		input := console.table.Lookup("con_get_input")
		if !input.Valid() {
			return 0, &consoleError{text: "native raw console input unavailable"}
		}
		var temporary [256]byte
		n := CallStdcall2Raw(uint32(input), uint32(uintptr(unsafe.Pointer(&temporary[0]))), uint32(len(temporary)))
		if n > uint32(len(temporary)) {
			return 0, &consoleError{text: "invalid native input length"}
		}
		consoleInputState.pending = append(consoleInputState.pending, temporary[:n]...)
	}
	n := copy(buffer, consoleInputState.pending)
	consoleInputState.pending = consoleInputState.pending[n:]
	return n, nil
}
