//go:build kolibrios

package dotlk

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"kos"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// The original VFS dot-lock backend serializes processes, while permitting
// its own connections to share the database and WAL index. KolibriOS supplies
// atomic exclusive creation through named memory, rather than O_EXCL files.
// Each lock belongs to a dedicated OS thread so kernel thread-exit cleanup
// releases it when the owning process dies. No stale files remain on disk.
type areaLock struct {
	release chan struct{}
	closed  chan struct{}
}

var areaLocks = struct {
	sync.Mutex
	items map[string]*areaLock
}{items: make(map[string]*areaLock)}

func areaLockName(name string) (string, error) {
	path, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	// FAT names are case insensitive. A conservative collision blocks another
	// database rather than granting conflicting ownership.
	sum := sha256.Sum256([]byte(strings.ToLower(path)))
	return "sqlk-" + hex.EncodeToString(sum[:12]), nil
}

// TryLock preserves the upstream nil/ErrExist contract with syscall 68/22.
// It requires the documented SHM_CREATE error return fixed by the SDK's
// isolated kernel builder.
func TryLock(name string) error {
	area, err := areaLockName(name)
	if err != nil {
		return err
	}
	areaLocks.Lock()
	defer areaLocks.Unlock()
	if areaLocks.items[area] != nil {
		return fs.ErrExist
	}
	lock := &areaLock{release: make(chan struct{}), closed: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		address, status := kos.OpenNamedMemory(area, 4096, kos.SharedMemoryCreate|kos.SharedMemoryWrite)
		if address == 0 {
			if status == 10 {
				result <- fs.ErrExist
				return
			}
			code := syscall.EINVAL
			if status == 30 {
				code = syscall.ENOMEM
			}
			result <- &os.PathError{Op: "lock", Path: name, Err: code}
			return
		}
		result <- nil
		<-lock.release
		kos.CloseNamedMemory(area)
		close(lock.closed)
	}()
	if err = <-result; err != nil {
		return err
	}
	areaLocks.items[area] = lock
	return nil
}

// LockShm excludes other processes from the upstream in-process WAL index.
func LockShm(name string) error { return TryLock(name) }

func Unlock(name string) error {
	area, err := areaLockName(name)
	if err != nil {
		return err
	}
	areaLocks.Lock()
	defer areaLocks.Unlock()
	lock := areaLocks.items[area]
	if lock == nil {
		return nil
	}
	delete(areaLocks.items, area)
	close(lock.release)
	<-lock.closed
	return nil
}
