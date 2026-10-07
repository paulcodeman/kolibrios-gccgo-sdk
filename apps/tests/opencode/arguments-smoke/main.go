package main

import (
	"kos"
	"os"
)

func main() {
	want := []string{"--help", "two words", "Привет", ""}
	if len(os.Args) != len(want)+1 {
		panic("loader argument count")
	}
	for i, value := range want {
		if os.Args[i+1] != value {
			panic("loader argument value")
		}
	}
	kos.DebugString("OPENCODE_ARGUMENTS_PASS")
}
