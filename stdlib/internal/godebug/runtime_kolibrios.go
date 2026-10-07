//go:build kolibrios && gccgo

package godebug

import (
	"kos"
	"sync"
	"syscall"
	"unsafe"
)

var nativeMetricReaders sync.Map
var nativeNewIncNonDefault func(string) func()

func setUpdate(update func(string, string)) {
	syscall.RuntimeGodebugChanged = func(environment string) { update("", environment) }
	environment, _ := syscall.Getenv("GODEBUG")
	update("", environment)
}

func registerMetric(name string, read func() uint64) { nativeMetricReaders.Store(name, read) }
func setNewIncNonDefault(create func(string) func()) { nativeNewIncNonDefault = create }

func write(fd uintptr, p unsafe.Pointer, n int32) int32 {
	if n <= 0 { return 0 }
	data := unsafe.Slice((*byte)(p),int(n))
	if kos.HasActiveConsole() {
		written, err := kos.WriteActiveConsole(data)
		if err == nil { return int32(written) }
	}
	kos.DebugString(string(data))
	return n
}

// Get retains the older libgo caller API using the original setting parser.
func Get(name string) string { return New("#" + name).Value() }
