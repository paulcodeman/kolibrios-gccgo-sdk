package main

import (
	"kos"
	"regression/implementation"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode imported interface method tables")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	if implementation.New().Value() != 42 {
		panic("imported private receiver method table")
	}
	console.WriteString("OpenCode imported private receiver and foreign sealed interface PASS\n")
	kos.DebugString("OPENCODE_METHOD_TABLE_PASS")
}
