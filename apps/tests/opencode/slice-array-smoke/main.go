package main

import (
	"kos"
	"runtime"
)

type pair [2]int
type empty [0]int
type numbers []int

var evaluations int

func input() numbers { evaluations++; return nil }

func short() (message string) {
	defer func() {
		value := recover()
		err, ok := value.(runtime.Error)
		if !ok {
			panic("bounds panic did not implement runtime.Error")
		}
		message = err.Error()
	}()
	_ = [2]int(make([]int, 1, 3))
	return
}

func main() {
	console, ok := kos.OpenConsole("OpenCode native slice-to-array conversion")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	s := numbers{3, 5, 7}
	v := pair(s)
	s[0] = 99
	if v != (pair{3, 5}) {
		panic("array value retained slice alias")
	}
	p := (*pair)(s)
	p[0] = 11
	if s[0] != 11 {
		panic("array pointer lost alias")
	}
	if empty(input()) != (empty{}) || evaluations != 1 {
		panic("nil empty array conversion")
	}
	_ = [0]int([]int{})
	message := short()
	if message != "runtime error: cannot convert slice with length 1 to pointer to array with length 2" {
		panic("wrong bounds error: " + message)
	}
	console.WriteString("OpenCode slice-to-array conversion PASS\n")
	kos.DebugString("OPENCODE_SLICE_ARRAY_PASS")
}
