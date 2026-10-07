//go:build kolibrios && gccgo

package os

import "sync"

// Native files are addressed by path. Child descriptor duplication still
// needs an independent File lifetime and the same underlying file position.
type nativeFilePosition struct {
	mu     sync.Mutex
	offset uint64
}

func (file *File) cloneNativeChildFile() (*File, error) {
	if file.localStream != nil {
		return file.cloneNativeLocal()
	}
	if file.closed {
		return nil, ErrClosed
	}
	if !file.fdBacked && file.sharedPosition == nil {
		file.sharedPosition = &nativeFilePosition{offset: file.offset}
	}
	clone := *file
	return &clone, nil
}

func (file *File) lockSharedPosition() func() {
	position := file.sharedPosition
	if position == nil {
		return func() {}
	}
	position.mu.Lock()
	file.offset = position.offset
	return func() { position.offset = file.offset; position.mu.Unlock() }
}
