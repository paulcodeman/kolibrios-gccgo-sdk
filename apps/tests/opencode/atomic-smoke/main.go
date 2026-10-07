package main

import (
	"kos"
	"runtime"
	"sync/atomic"
	"unsafe"
)

func check(ok bool, why string) {
	if !ok {
		kos.DebugString(why)
		panic(why)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode atomic port test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_ATOMIC_START")
	runtime.GOMAXPROCS(2)
	var total atomic.Int32
	done := make(chan bool)
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 5000; j++ {
				total.Add(1)
			}
			done <- true
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	check(total.Load() == 40000, "concurrent atomic increments")
	check(total.Swap(3) == 40000 && total.CompareAndSwap(3, 5), "atomic swap and CAS")
	check(total.Or(2) == 5 && total.Load() == 7, "atomic OR returns old value")
	check(total.And(6) == 7 && total.Load() == 6, "atomic AND returns old value")
	kos.DebugString("ATOMIC32_OK")
	var value atomic.Value
	check(value.Load() == nil, "empty atomic.Value")
	check(value.CompareAndSwap(nil, "first"), "first atomic.Value CAS")
	check(value.Swap("second") == "first" && value.Load() == "second", "atomic.Value swap")
	check(!value.CompareAndSwap("first", "third") && value.CompareAndSwap("second", "third"), "atomic.Value comparison")
	var pointer atomic.Pointer[int]
	n := 42
	pointer.Store(&n)
	check(pointer.Load() == &n && pointer.CompareAndSwap(&n, nil), "atomic pointer")
	kos.DebugString("ATOMIC_VALUE_OK")
	check(unsafe.Alignof(atomic.Int64{}) >= 8 && unsafe.Alignof(atomic.Uint64{}) >= 8, "typed 64-bit atomic alignment")
	type align64 struct{}
	check(unsafe.Alignof(align64{}) == 1, "unrelated empty struct alignment")
	type wrapped struct {
		pad   uint32
		value atomic.Int64
	}
	var w wrapped
	check(uintptr(unsafe.Pointer(&w.value))%8 == 0, "embedded atomic alignment")
	w.value.Store(1 << 40)
	check(w.value.Add(1) == 1<<40|1 && w.value.Or(2) == 1<<40|1 && w.value.Load() == 1<<40|3, "atomic 64-bit operations")
	values := make([]wrapped, 25)
	for i := range values {
		check(uintptr(unsafe.Pointer(&values[i].value))%8 == 0, "slice atomic alignment")
		values[i].value.Store(int64(i))
	}
	var wide atomic.Int64
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 5000; j++ {
				wide.Add(1)
			}
			done <- true
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	check(wide.Load() == 40000, "concurrent 64-bit increments")
	unaligned := false
	func() {
		defer func() { unaligned = recover() != nil }()
		atomic.LoadInt64((*int64)(unsafe.Pointer(uintptr(unsafe.Pointer(&w.value)) + 4)))
	}()
	check(unaligned, "unaligned 64-bit operation must panic")
	kos.DebugString("OPENCODE_ATOMIC_PASS")
	console.WriteString("OPENCODE_ATOMIC_PASS\n")
}
