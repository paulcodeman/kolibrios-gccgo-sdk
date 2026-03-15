package main

import "kos"

func main() {
	console, ok := kos.OpenConsole("Hello")
	if !ok {
		kos.Exit()
		return
	}

	console.WriteString("Console test")
	console.Exit(true)
	kos.Exit()
}
