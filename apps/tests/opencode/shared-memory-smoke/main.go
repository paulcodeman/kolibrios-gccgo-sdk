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
	const name = "opencode-shm-native-test"
	if kos.LoaderParameters() == "shm-child" {
		runtime.LockOSThread()
		duplicate, err := kos.OpenNamedMemory(name, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
		check(duplicate == 0 && err == 10, "cross-process exclusive creation")
		other, size := kos.OpenNamedMemory(name, 0, kos.SharedMemoryOpen|kos.SharedMemoryWrite)
		check(other != 0 && size == 4096, "cross-process open")
		check(*(*byte)(unsafe.Pointer(other)) == 42, "cross-process shared data")
		owned, status := kos.OpenNamedMemory("opencode-shm-child-only", 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
		check(owned != 0 && status == 0, "child creates automatic cleanup area")
		atomic.StoreUint32((*uint32)(unsafe.Pointer(other+4)), 2026)
		kos.DebugString("SHM_CHILD_OK")
		// Both mappings are deliberately left for kernel thread-exit cleanup.
		return
	}
	console, ok := kos.OpenConsole("OpenCode shared memory test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_SHARED_MEMORY_START")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	address, result := kos.OpenNamedMemory(name, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
	check(address != 0 && result == 0, "create named area")
	data := unsafe.Slice((*byte)(unsafe.Pointer(address)), 4096)
	data[0] = 42
	done := make(chan bool)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		duplicate, err := kos.OpenNamedMemory(name, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
		kos.DebugString("SHM_DUPLICATE=")
		kos.DebugOutHex(uint32(duplicate))
		kos.DebugOutHex(err)
		check(duplicate == 0 && err == 10, "exclusive area creation")
		other, size := kos.OpenNamedMemory(name, 0, kos.SharedMemoryOpen|kos.SharedMemoryWrite)
		check(other != 0 && size == 4096, "open existing area")
		shared := unsafe.Slice((*byte)(unsafe.Pointer(other)), 4096)
		check(shared[0] == 42, "shared data visibility")
		shared[1] = 99
		kos.CloseNamedMemory(name)
		done <- true
	}()
	<-done
	check(data[1] == 99, "shared writes")
	child, status := kos.StartApplication(kos.LoaderPath(), "shm-child", false)
	check(child > 0 && status == kos.FileSystemOK, "start child process")
	for i := 0; i < 300 && kos.ThreadSlotByIdentifier(child) != 0; i++ {
		kos.Sleep(1)
	}
	check(kos.ThreadSlotByIdentifier(child) == 0, "child exited")
	check(atomic.LoadUint32((*uint32)(unsafe.Pointer(address+4))) == 2026, "cross-process writes")
	leftover, err := kos.OpenNamedMemory("opencode-shm-child-only", 0, kos.SharedMemoryOpen)
	check(leftover == 0 && err == 5, "thread exit automatically removes last mapping")
	kos.CloseNamedMemory(name)
	address, result = kos.OpenNamedMemory(name, 0, kos.SharedMemoryOpen)
	check(address == 0 && result == 5, "last close removes area")
	address, result = kos.OpenNamedMemory(name, 1, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
	check(address != 0 && result == 0, "recreate after close")
	kos.CloseNamedMemory(name)
	address, result = kos.OpenNamedMemory("invalid\x00name", 4096, kos.SharedMemoryCreate)
	check(address == 0 && result == 33, "reject embedded NUL")
	kos.DebugString("OPENCODE_SHARED_MEMORY_PASS")
	console.WriteString("OPENCODE_SHARED_MEMORY_PASS\n")
}
