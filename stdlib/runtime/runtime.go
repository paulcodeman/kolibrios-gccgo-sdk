// Package runtime provides access to the Go runtime.
//
// This is a minimal KolibriOS wrapper that exposes a subset of the standard
// runtime API for porting convenience.
package runtime

const GOOS = "kolibrios"
const GOARCH = "386"

// Gosched yields the processor, allowing other goroutines to run.
func Gosched() __asm__("runtime.Gosched")

// LockOSThread wires the calling goroutine to its current OS thread.
func LockOSThread() __asm__("runtime.LockOSThread")

// UnlockOSThread undoes an earlier LockOSThread call.
func UnlockOSThread() __asm__("runtime.UnlockOSThread")

// GOMAXPROCS sets the maximum number of OS threads that can execute Go code
// simultaneously and returns the previous setting. If n < 1, it returns the
// current setting without changing it.
func GOMAXPROCS(n int) int {
	prev := int(getRuntimeThreads())
	if n < 1 {
		return prev
	}
	setRuntimeThreads(uint32(n))
	return prev
}

// NumCPU returns the number of logical CPUs usable by the runtime.
func NumCPU() int {
	return int(getRuntimeThreads())
}

func setRuntimeThreads(count uint32) uint32 __asm__("runtime_kolibri_set_threads")
func getRuntimeThreads() uint32 __asm__("runtime_kolibri_get_threads")
