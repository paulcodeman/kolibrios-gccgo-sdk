package kos

// SetConsoleSignalMode selects ordinary key queuing (0), canonical Ctrl-C
// capture (1), or canonical Ctrl-C suppression (2). Raw input always queues
// Ctrl-C as a byte. The native console validates the policy value.
func SetConsoleSignalMode(mode uint32) error {
	if mode > 2 {
		return &consoleError{text: "invalid native signal mode"}
	}
	console, ok := ActiveConsole()
	if !ok {
		return &consoleError{text: "active console unavailable"}
	}
	proc := console.table.Lookup("con_set_signal_mode")
	if !proc.Valid() {
		return &consoleError{text: "native console signal events unavailable"}
	}
	CallStdcall1Raw(uint32(proc), mode)
	return nil
}

func TakeConsoleInterrupts() uint32 {
	console, ok := ActiveConsole()
	if !ok {
		return 0
	}
	proc := console.table.Lookup("con_take_interrupts")
	if !proc.Valid() {
		return 0
	}
	return CallStdcall0Raw(uint32(proc))
}

func ActiveConsoleClosed() bool {
	console, ok := ActiveConsole()
	if !ok {
		return true
	}
	proc := console.table.Lookup("con_get_flags")
	return proc.Valid() && CallStdcall0Raw(uint32(proc))&0x200 != 0
}
