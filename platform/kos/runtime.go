// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package kos

import "runtime"

// Gosched yields the processor, allowing other goroutines to run.
func Gosched() {
	runtime.Gosched()
}
