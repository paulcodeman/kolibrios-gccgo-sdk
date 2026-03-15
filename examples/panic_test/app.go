package main

import (
	"fmt"
	"os"

	"kos"
)

const panicTestTitle = "KolibriOS Panic Test"

type testResult struct {
	name   string
	ok     bool
	detail string
}

func main() {
	console, ok := kos.OpenConsole(panicTestTitle)
	if !ok {
		kos.DebugString("panic test: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}

	if console.SupportsTitle() {
		console.SetTitle(panicTestTitle + " / running")
	}

	_, _ = fmt.Println("KolibriOS panic/recover test")
	_, _ = fmt.Println("Running checks...")

	results := runTests()
	passed := true
	for _, result := range results {
		status := "PASS"
		if !result.ok {
			status = "FAIL"
			passed = false
		}
		if result.detail != "" {
			_, _ = fmt.Printf("%s: %s (%s)\n", status, result.name, result.detail)
		} else {
			_, _ = fmt.Printf("%s: %s\n", status, result.name)
		}
	}

	if passed {
		_, _ = fmt.Println("Summary: all tests passed")
		if console.SupportsTitle() {
			console.SetTitle(panicTestTitle + " / ok")
		}
	} else {
		_, _ = fmt.Println("Summary: failures detected")
		if console.SupportsTitle() {
			console.SetTitle(panicTestTitle + " / failures")
		}
	}

	if console.SupportsInput() {
		_, _ = fmt.Println("Press P to trigger a fatal panic, any other key to exit.")
		key := console.Getch()
		if key == 'p' || key == 'P' {
			panic("fatal panic test")
		}
	} else {
		_, _ = fmt.Println("Input export missing, closing in three seconds.")
		kos.SleepSeconds(3)
	}

	os.Exit(0)
}

func runTests() []testResult {
	return []testResult{
		safeTest("recover string", testRecoverString),
		safeTest("recover int", testRecoverInt),
		safeTest("recover only in defer", testRecoverOnlyInDefer),
		safeTest("defer runs on panic", testDeferRunsOnPanic),
	}
}

func safeTest(name string, fn func() (bool, string)) testResult {
	result := testResult{name: name}
	defer func() {
		if recovered := recover(); recovered != nil {
			result.ok = false
			result.detail = fmt.Sprintf("panic during test: %v", recovered)
		}
	}()
	result.ok, result.detail = fn()
	return result
}

func testRecoverString() (bool, string) {
	var recovered interface{}
	func() {
		defer func() {
			recovered = recover()
		}()
		panic("boom")
	}()

	if recovered == nil {
		return false, "recover returned nil"
	}
	if value, ok := recovered.(string); !ok || value != "boom" {
		return false, fmt.Sprintf("unexpected value: %v", recovered)
	}
	return true, ""
}

func testRecoverInt() (bool, string) {
	var recovered interface{}
	func() {
		defer func() {
			recovered = recover()
		}()
		panic(123)
	}()

	if recovered == nil {
		return false, "recover returned nil"
	}
	value, ok := recovered.(int)
	if !ok || value != 123 {
		return false, fmt.Sprintf("unexpected value: %v", recovered)
	}
	return true, ""
}

func testRecoverOnlyInDefer() (bool, string) {
	if func() interface{} { return recover() }() != nil {
		_, _ = fmt.Println("recover outside defer should be nil")
		return false, "recover outside defer should be nil"
	}

	var recovered interface{}
	func() {
		defer func() {
			recovered = recover()
		}()
		panic("inside")
	}()

	if recovered == nil {
		_, _ = fmt.Println("recover in defer returned nil")
		return false, "recover in defer returned nil"
	}
	return true, ""
}

func testDeferRunsOnPanic() (bool, string) {
	ran := false
	func() {
		defer func() {
			ran = true
			_ = recover()
		}()
		panic("defer")
	}()

	if !ran {
		return false, "defer did not run"
	}
	return true, ""
}
