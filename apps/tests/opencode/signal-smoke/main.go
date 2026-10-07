package main

import (
	"context"
	"kos"
	"os"
	"os/signal"
	"time"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode native signal delivery")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	wait := func(ch <-chan os.Signal) {
		select {
		case got := <-ch:
			if got != os.Interrupt {
				panic("wrong signal")
			}
		case <-time.After(15 * time.Second):
			panic("native signal timeout")
		}
	}
	first := make(chan os.Signal, 1)
	second := make(chan os.Signal, 1)
	signal.Notify(first, os.Interrupt)
	signal.Notify(second, os.Interrupt)
	wait(first)
	wait(second)
	if console.KeyHit() {
		panic("canonical interrupt leaked into stdin")
	}
	signal.Stop(first)
	signal.Stop(second)
	console.WriteString("PASS two subscribers and Stop\n")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			panic("context cancellation")
		}
	case <-time.After(15 * time.Second):
		panic("native NotifyContext timeout")
	}
	stop()
	console.WriteString("PASS NotifyContext\n")

	signal.Notify(first, os.Interrupt)
	if _, err := kos.SetConsoleInputRaw(true); err != nil {
		panic(err)
	}
	var input [1]byte
	if n, err := os.Stdin.Read(input[:]); n != 1 || err != nil || input[0] != 3 {
		panic("raw interrupt byte")
	}
	select {
	case <-first:
		panic("raw input emitted signal")
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := kos.SetConsoleInputRaw(false); err != nil {
		panic(err)
	}
	signal.Ignore(os.Interrupt)
	if !signal.Ignored(os.Interrupt) {
		panic("ignored state")
	}
	time.Sleep(4 * time.Second)
	if console.KeyHit() {
		panic("ignored interrupt leaked into stdin")
	}
	select {
	case <-first:
		panic("ignored interrupt delivered")
	default:
	}
	signal.Reset(os.Interrupt)
	if signal.Ignored(os.Interrupt) {
		panic("reset ignored state")
	}
	console.WriteString("PASS raw mode and Ignore/Reset\n")

	signal.Notify(first, os.Interrupt)
	signal.Stop(first)
	if _, err := kos.SetConsoleInputRaw(true); err != nil {
		panic(err)
	}
	if n, err := os.Stdin.Read(input[:]); n != 1 || err != nil || input[0] != 3 {
		panic("input after Stop")
	}
	time.Sleep(100 * time.Millisecond)
	select {
	case <-first:
		panic("delivery after Stop")
	default:
	}
	kos.SetConsoleInputRaw(false)
	console.WriteString("OpenCode native signal delivery PASS\n")
	kos.DebugString("OPENCODE_SIGNAL_PASS")
}
