//go:build kolibrios && gccgo

package os

import (
	"kos"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Go runtime M records know the exact native PID/TID and slot of each worker.
// Parent and child share these records using the documented named-memory API.
// This avoids guessing process membership from virtual addresses or names.
type nativeProcessControl struct {
	words     *[1024]uint32
	closeOnce sync.Once
	stop      chan struct{}
	done      chan struct{}
}

func newNativeProcessControl(statusPath string) (*nativeProcessControl, error) {
	name := kos.ProcessControlName(statusPath)
	if name == "" {
		return nil, syscall.EINVAL
	}
	ready := make(chan *nativeProcessControl, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		address, code := kos.OpenNamedMemory(name, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
		if address == 0 || code != 0 {
			ready <- nil
			return
		}
		control := &nativeProcessControl{words: (*[1024]uint32)(unsafe.Pointer(address)), stop: make(chan struct{}), done: make(chan struct{})}
		for i := range control.words {
			atomic.StoreUint32(&control.words[i], 0)
		}
		ready <- control
		<-control.stop
		kos.CloseNamedMemory(name)
		close(control.done)
	}()
	control := <-ready
	if control == nil {
		return nil, syscall.ENOMEM
	}
	return control, nil
}

func (control *nativeProcessControl) close() {
	if control == nil {
		return
	}
	control.closeOnce.Do(func() { close(control.stop) })
	<-control.done
}

func (control *nativeProcessControl) terminateWorkers(mainPID int) {
	if control == nil {
		return
	}
	atomic.StoreUint32(&control.words[0], 1)
	// Spawn checks the stop flag before creation and each new M checks it
	// after publishing. An in-flight spawn therefore exits itself as well.
	for slot := 1; slot <= 256; slot++ {
		id := int(atomic.LoadUint32(&control.words[slot]))
		if id != 0 && id != mainPID && kos.ThreadSlotByIdentifier(id) == slot {
			kos.TerminateByIdentifier(id)
		}
	}
}

func (control *nativeProcessControl) waitWorkers(mainPID int) {
	if control == nil {
		return
	}
	for {
		alive := false
		for slot := 1; slot <= 256; slot++ {
			id := int(atomic.LoadUint32(&control.words[slot]))
			if id != 0 && id != mainPID && kos.ThreadSlotByIdentifier(id) == slot {
				alive = true
				break
			}
		}
		if !alive {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
