package main

import (
	"kos"
	"regexp"
	"strings"
)

var deepPattern = regexp.MustCompile(strings.Repeat("(", 700) + "x" + strings.Repeat(")", 700))

func main() {
	console, ok := kos.OpenConsole("OpenCode original regexp stack")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	if deepPattern.NumSubexp() != 700 {
		panic("nested capture count")
	}
	matches := deepPattern.FindStringSubmatch("x")
	if len(matches) != 701 {
		panic("nested match count")
	}
	for _, match := range matches {
		if match != "x" {
			panic("nested capture content")
		}
	}
	console.WriteString("Original regexp with 700 nested captures compiled before main and matched PASS\n")
	kos.DebugString("OPENCODE_REGEXP_STACK_PASS")
}
