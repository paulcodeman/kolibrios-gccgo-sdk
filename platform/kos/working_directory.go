package kos

import (
	"runtime"
	"sync"
)

// Native function 30 stores a directory per OS thread. Go's os.Chdir
// changes the directory for the application, so filesystem wrappers resolve
// relative paths against this shared value before entering the kernel.
var applicationDirectory struct {
	sync.RWMutex
	path string
}

func ChangeCurrentFolder(name string) FileSystemStatus {
	for i := 0; i < len(name); i++ {
		if name[i] == 0 {
			return FileSystemBadPointer
		}
	}
	absolute, ok := absoluteFSPath(name)
	if !ok {
		return FileSystemNotFound
	}
	info, status := GetPathInfo(absolute)
	if status != FileSystemOK || info.Attributes&FileAttributeDirectory == 0 {
		// Native volume roots and /sys can be listed but do not always
		// implement metadata lookup. Verify them through the directory API.
		_, readStatus := ReadDirectory(absolute, 0, 1)
		if readStatus != FileSystemOK && readStatus != FileSystemEOF {
			if status == FileSystemOK {
				return FileSystemAccessDenied
			}
			return status
		}
	}
	applicationDirectory.Lock()
	applicationDirectory.path = absolute
	applicationDirectory.Unlock()
	return FileSystemOK
}

func fsStringAddress(name string) (*byte, uint32) {
	if name != "" {
		if absolute, ok := absoluteFSPath(name); ok {
			name = absolute
		}
	}
	return stringAddress(name)
}

// Function 30/4 has no status result. Callers pass an absolute directory
// already checked through function 70, using the documented UTF-8 encoding.
func setNativeThreadDirectory(name string) {
	ptr, address := stringAddress(name)
	if ptr == nil {
		return
	}
	regs := SyscallRegs{EAX: 30, EBX: 4, ECX: address, EDX: uint32(EncodingUTF8)}
	SyscallRaw(&regs)
	freeCString(ptr)
}

func withApplicationDirectory(start func() int) int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous := nativeCurrentFolderWithEncoding(EncodingUTF8)
	current := CurrentFolder()
	if previous != current {
		setNativeThreadDirectory(current)
		defer setNativeThreadDirectory(previous)
	}
	return start()
}
