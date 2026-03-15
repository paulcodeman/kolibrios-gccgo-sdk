package main

import (
	"bufio"
	"fmt"
	"os"

	"kos"
)

const consoleDemoTitle = "KolibriOS Console Demo"
const consoleExitKey = 27

func main() {
	console, ok := kos.OpenConsole(consoleDemoTitle)
	if !ok {
		kos.DebugString("console demo: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}

	if _, err := fmt.Println("KolibriOS console demo"); err != nil {
		kos.DebugString("console demo: stdout write failed")
		os.Exit(1)
		return
	}
	_, _ = fmt.Printf("Loaded %s and resolved required exports.\n", kos.ConsoleDLLPath)
	_, _ = fmt.Println("fmt.Print* now routes through os.Stdout into CONSOLE.OBJ.")
	_, _ = fmt.Printf("export table: 0x%x / version: 0x%x\n", uint32(console.ExportTable()), console.Version())
	if console.SupportsTitle() {
		console.SetTitle(consoleDemoTitle + " / ready")
	}
	if console.SupportsLineInput() {
		runConsoleLineDemo()
	}

	if console.SupportsInput() {
		_, _ = fmt.Println("Press Esc to close this console.")
		waitForConsoleExitKey(console)
	} else {
		_, _ = fmt.Println("Input export missing, closing in three seconds.")
		kos.SleepSeconds(3)
	}

	os.Exit(0)
}

func runConsoleLineDemo() {
	reader := bufio.NewReader(os.Stdin)

	_, _ = fmt.Print("Type a full line and press Enter: ")
	line, err := reader.ReadString('\n')
	if err != nil {
		_, _ = fmt.Printf("ReadString failed: %v\n", err)
		return
	}

	line = trimConsoleLine(line)
	_, _ = fmt.Printf("You typed: %s\n", line)
}

func trimConsoleLine(line string) string {
	for len(line) > 0 {
		last := line[len(line)-1]
		if last != '\r' && last != '\n' {
			break
		}
		line = line[:len(line)-1]
	}

	return line
}

func waitForConsoleExitKey(console kos.Console) {
	for {
		key := console.Getch()
		if key == consoleExitKey {
			return
		}
	}
}
