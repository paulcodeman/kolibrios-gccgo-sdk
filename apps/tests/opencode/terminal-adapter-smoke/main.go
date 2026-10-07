package main

import (
	"errors"
	"kos"
	"os"
	"syscall"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

func main() {
	console, ok := kos.LoadConsole()
	if !ok || !console.Init(83, 27, 83, 300, "OpenCode terminal adapters") {
		panic("native console initialization")
	}
	defer console.Exit(false)
	if !term.IsTerminal(0) || !term.IsTerminal(1) || !term.IsTerminal(2) || term.IsTerminal(3) {
		panic("terminal descriptor identification")
	}
	if _, err := term.MakeRaw(3); !errors.Is(err, syscall.ENOTTY) {
		panic("nonterminal raw-mode error")
	}
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width != 83 || height != 27 {
		panic("native terminal-size adapter")
	}
	old, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil || !kos.ConsoleInputRaw() {
		panic("raw input mode")
	}
	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		panic(err)
	}
	readDone := make(chan error, 1)
	go func() {
		var buffer [1]byte
		n, err := reader.Read(buffer[:])
		if n != 0 {
			panic("canceled read consumed data")
		}
		readDone <- err
	}()
	time.Sleep(30 * time.Millisecond)
	if !reader.Cancel() {
		panic("native read cancellation unavailable")
	}
	select {
	case err := <-readDone:
		if err != cancelreader.ErrCanceled {
			panic("canceled read error")
		}
	case <-time.After(time.Second):
		panic("canceled read still blocked")
	}
	if n, err := reader.Read(nil); n != 0 || err != cancelreader.ErrCanceled {
		panic("read after cancellation")
	}
	reader.Close()
	if os.Stdin.Fd() != 0 {
		panic("cancellation closed caller's stdin")
	}
	console.WriteString("Waiting for raw input through os.Stdin...\n")
	var received []byte
	for len(received) < 6 {
		var buffer [1]byte
		n, err := os.Stdin.Read(buffer[:])
		if err != nil || n != 1 {
			panic("small raw read")
		}
		received = append(received, buffer[0])
	}
	if string(received) != "a\x1b[A\r\x03" {
		panic("raw escape-sequence preservation")
	}
	if err := term.Restore(0, old); err != nil || kos.ConsoleInputRaw() {
		panic("restore canonical mode")
	}
	console.WriteString("Waiting for a test password (no echo)...\n")
	password, err := term.ReadPassword(0)
	if err != nil || string(password) != "ac" || kos.ConsoleInputRaw() {
		panic("password editing or mode restoration")
	}
	console.WriteString("OpenCode terminal adapters PASS\n")
	kos.DebugString("OPENCODE_TERMINAL_ADAPTER_PASS")
}
