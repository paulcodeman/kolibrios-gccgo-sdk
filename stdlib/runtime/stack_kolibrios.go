package runtime

// Version identifies the SDK runtime. gccgo builds do not embed cmd/go's
// buildVersion or module metadata in their flat KolibriOS executables.
func Version() string { return "gccgo (KolibriOS SDK bootstrap)" }

func collectCallers(pcs *uintptr, capacity int) int __asm__("runtime_kolibri_collect_callers")

// Stack writes a trace of the calling goroutine. Program counters are collected
// by libgcc's DWARF unwinder; the bootstrap SDK has no Go PC-to-line table yet.
// Capturing suspended goroutines requires scheduler support that is not present.
func Stack(buf []byte, all bool) int {
	if all {
		panic("runtime.Stack: all-goroutine traces are unsupported on KolibriOS")
	}
	var pcs [128]uintptr
	count := collectCallers(&pcs[0], len(pcs))
	trace := []byte("goroutine [running] (program counters):\n")
	const hex = "0123456789abcdef"
	for i := 0; i < count; i++ {
		trace = append(trace, '\t', '0', 'x')
		pc := pcs[i]
		for shift := 28; shift >= 0; shift -= 4 {
			trace = append(trace, hex[(pc>>uint(shift))&15])
		}
		trace = append(trace, '\n')
	}
	return copy(buf, trace)
}
