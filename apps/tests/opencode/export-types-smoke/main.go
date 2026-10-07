package main

import (
	"kos"
	"regression/exporttypes"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode compiler function exports")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	exporttypes.Callback()
	if exporttypes.Clamp(-4, 1, 7) != 1 || exporttypes.Clamp(12, 1, 7) != 7 || exporttypes.Clamp(3, 1, 7) != 3 {
		panic("imported inline min/max")
	}
	if exporttypes.Variadic(1, 2, 3) != 3 || exporttypes.Slice([]int{1, 2}) != 2 {
		panic("function export signatures")
	}
	console.WriteString("OpenCode function export types PASS\n")
	kos.DebugString("OPENCODE_EXPORT_TYPES_PASS")
}
