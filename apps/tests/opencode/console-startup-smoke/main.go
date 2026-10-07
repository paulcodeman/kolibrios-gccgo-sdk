package main

import (
	"kos"
	"os"
)

// This fixture tests console startup. Public model discovery is covered by the
// native full-CLI test, with the real bootstrap/zen_public.go implementation.
func loadZenPublicModels() error { return nil }

func main() {
	if !kos.HasActiveConsole() || len(os.Args) != 2 || os.Args[1] != "--help" {
		panic("native CLI startup")
	}
	const text = "OpenCode console bootstrap PASS\n"
	if n, err := os.Stdout.Write([]byte(text)); err != nil || n != len(text) {
		panic("native stdout")
	}
	kos.DebugString("OPENCODE_CONSOLE_STARTUP_PASS")
}
