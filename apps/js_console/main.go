package main

import (
	"fmt"
	"os"

	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/repl"
	"kos"
)

const consoleTitle = "KolibriOS JS Console"

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("js console: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}

	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	vm := otto.New()
	_, _ = vm.Run("var print = console.log;")

	prelude := "KolibriOS JavaScript Console (Otto)\n" +
		"Enter JavaScript and press Enter.\n" +
		"Press Ctrl+C or Esc to exit."

	if err := repl.RunWithOptions(vm, repl.Options{
		Prompt:  "> ",
		Prelude: prelude,
	}); err != nil {
		_, _ = fmt.Printf("REPL error: %v\n", err)
	}
}
