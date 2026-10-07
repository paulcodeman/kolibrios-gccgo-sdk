package main

import "kos"
import "strings"

type seq func(func(int) bool)
type seq2 func(func(int, string) bool)

var trace string

func pair(y func(int, string) bool) {
	defer func() { trace += "P" }()
	for i := 1; i <= 4; i++ {
		if !y(i, "a") {
			return
		}
	}
}
func many(y func(int) bool) {
	for i := 1; i <= 4; i++ {
		if !y(i) {
			return
		}
	}
}
func answer() (n int) {
	defer func() { n++; trace += "D" }()
	for i := range seq(many) {
		for j := range seq(many) {
			if i == 2 && j == 3 {
				n := 99
				_ = n
				return i*10 + j
			}
		}
	}
	return 0
}
func bare() (n int) {
	for i := range seq(many) {
		n = i
		if i == 3 {
			return
		}
	}
	return
}
func tuples() (int, string) {
	for i, s := range seq2(pair) {
		if i == 2 {
			return i, s
		}
	}
	return 0, ""
}
func mixed() string {
	for range seq(many) {
		for s := range strings.SplitSeq("a,b", ",") {
			return s
		}
	}
	return ""
}
func storedMixed() string {
	for range seq(many) {
		split := strings.SplitSeq("a,b", ",")
		for s := range split {
			return s
		}
	}
	return ""
}
func storedSingle() string {
	split := strings.SplitSeq("a,b", ",")
	for s := range split {
		return s
	}
	return ""
}
func returned() {
	defer func() { trace += "R" }()
	for i := range seq2(pair) {
		if i == 2 {
			return
		}
	}
	panic("return lost")
}
func brokenIterator(y func(int) bool) { y(1); y(2) }
func swallowed(y func(int) bool)      { defer func() { recover() }(); y(1) }
func resumed(y func(int) bool)        { func() { defer func() { recover() }(); y(1) }(); y(2) }
func getPanic(f func()) (text string) {
	defer func() {
		e, ok := recover().(interface {
			Error() string
			RuntimeError()
		})
		if !ok {
			panic("runtime.Error expected")
		}
		text = e.Error()
	}()
	f()
	panic("panic missing")
}
func runRangeChecks() {
	var capture []func() int
	sum := 0
	for i, s := range seq2(pair) {
		if s != "a" {
			panic("second value")
		}
		if i == 2 {
			continue
		}
		capture = append(capture, func() int { return i })
		sum += i
		if i == 3 {
			break
		}
	}
	if sum != 4 || capture[0]() != 1 || capture[1]() != 3 || trace != "P" {
		panic("break/continue/capture")
	}
	var i int
	var s string
	for i, s = range seq2(pair) {
		if i == 2 {
			break
		}
	}
	if i != 2 || s != "a" {
		panic("assignment range")
	}
	hits := 0
	for i := range seq(many) {
		switch i {
		case 1:
			break
		case 2:
			continue
		}
		for j := 0; j < 4; j++ {
			if j == 1 {
				continue
			}
			if j == 2 {
				break
			}
			hits++
		}
	}
	if hits != 3 {
		panic("nested ordinary control flow")
	}
	if answer() != 24 || bare() != 3 {
		panic("nested return")
	}
	if mixed() != "a" || storedMixed() != "a" || storedSingle() != "a" {
		panic("mixed slice intrinsic return")
	}
	n, s2 := tuples()
	if n != 2 || s2 != "a" {
		panic("tuple return")
	}
	returned()
	count := 0
	for range func(y func() bool) { y(); y() } {
		count++
	}
	if count != 2 {
		panic("zero values")
	}
	evaluations := 0
	for range func() seq { evaluations++; return many }() {
		break
	}
	if evaluations != 1 {
		panic("iterator evaluated twice")
	}
	stopped := getPanic(func() {
		for range brokenIterator {
			break
		}
	})
	if stopped != "runtime error: range function continued iteration after function for loop body returned false" {
		panic(stopped)
	}
	missing := getPanic(func() {
		for range swallowed {
			panic("body")
		}
	})
	if missing != "runtime error: range function recovered a loop body panic and did not resume panicking" {
		panic(missing)
	}
	resumedPanic := getPanic(func() {
		for range resumed {
			panic("body")
		}
	})
	if resumedPanic != "runtime error: range function continued iteration after loop body panic" {
		panic(resumedPanic)
	}
	var saved func(int) bool
	for range func(y func(int) bool) { saved = y } {
	}
	exhausted := getPanic(func() { saved(1) })
	if exhausted != "runtime error: range function continued iteration after whole loop exit" {
		panic(exhausted)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode Go 1.23 function ranges")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	runRangeChecks()
	console.WriteString("OpenCode upstream function range control flow and panic checks PASS\n")
	kos.DebugString("OPENCODE_RANGEFUNC_PASS")
}
