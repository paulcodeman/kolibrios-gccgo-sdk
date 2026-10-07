//go:build kolibrios

// Adapted from libgo's signal_unix.go. Delivery comes from native console
// Ctrl-C capture and actual terminal dimension changes. Other signal numbers
// have no native event source; window close retains upstream console behavior.
package signal

import (
	"kos"
	"os"
	"sync"
	"syscall"
	"time"
)

const numSig = 32

var nativeSignals struct {
	sync.Mutex
	enabled uint32
	ignored uint32
}
var nativeSignalDispatch sync.Mutex
var nativeSignalWake = make(chan struct{}, 1)

func init() { watchSignalLoop = nativeSignalLoop }

func signum(sig os.Signal) int {
	if value, ok := sig.(syscall.Signal); ok {
		if value >= 0 && int(value) < numSig {
			return int(value)
		}
	}
	return -1
}

func enableSignal(sig int) {
	if sig < 0 || sig >= numSig {
		return
	}
	if sig == int(syscall.SIGINT) {
		if err := kos.SetConsoleSignalMode(1); err != nil {
			panic(err)
		}
	}
	nativeSignals.Lock()
	nativeSignals.enabled |= 1 << uint(sig)
	nativeSignals.ignored &^= 1 << uint(sig)
	nativeSignals.Unlock()
	select {
	case nativeSignalWake <- struct{}{}:
	default:
	}
}

func disableSignal(sig int) {
	if sig == int(syscall.SIGINT) {
		kos.SetConsoleSignalMode(0)
		kos.TakeConsoleInterrupts()
	}
	nativeSignals.Lock()
	nativeSignals.enabled &^= 1 << uint(sig)
	nativeSignals.ignored &^= 1 << uint(sig)
	nativeSignals.Unlock()
}

func ignoreSignal(sig int) {
	if sig == int(syscall.SIGINT) {
		if err := kos.SetConsoleSignalMode(2); err != nil {
			panic(err)
		}
		kos.TakeConsoleInterrupts()
	}
	nativeSignals.Lock()
	nativeSignals.enabled &^= 1 << uint(sig)
	nativeSignals.ignored |= 1 << uint(sig)
	nativeSignals.Unlock()
}

func signalIgnored(sig int) bool {
	nativeSignals.Lock()
	defer nativeSignals.Unlock()
	return nativeSignals.ignored&(1<<uint(sig)) != 0
}

func signalWaitUntilIdle() {
	// Stop invokes this after releasing handlers.Lock. Waiting here also
	// covers a native event already dequeued by the delivery goroutine.
	nativeSignalDispatch.Lock()
	nativeSignalDispatch.Unlock()
}

func nativeSignalLoop() {
	var table kos.DLLExportTable
	var columns, rows int
	for {
		nativeSignals.Lock()
		enabled := nativeSignals.enabled
		nativeSignals.Unlock()
		if enabled == 0 {
			<-nativeSignalWake
			continue
		}
		nativeSignalDispatch.Lock()
		// Refresh under the delivery barrier: Stop may have disabled a source
		// after the initial snapshot but before this iteration acquired it.
		nativeSignals.Lock()
		enabled = nativeSignals.enabled
		nativeSignals.Unlock()
		console, active := kos.ActiveConsole()
		if active {
			current := console.ExportTable()
			if current != table {
				table, columns, rows = current, 0, 0
			}
			if enabled&(1<<uint(syscall.SIGINT)) != 0 && kos.TakeConsoleInterrupts() > 0 {
				process(syscall.SIGINT)
			}
			if enabled&(1<<uint(syscall.SIGWINCH)) != 0 && !kos.ActiveConsoleClosed() {
				width, height, err := kos.ActiveConsoleSize()
				if err == nil {
					if columns != 0 && (columns != width || rows != height) {
						process(syscall.SIGWINCH)
					}
					columns, rows = width, height
				}
			}
		}
		nativeSignalDispatch.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
}
