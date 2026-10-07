//go:build kolibrios && gccgo

package os

import (
	"io"
	"kos"
	"sync"
	"syscall"
	"time"
)

// Adapt the existing native net polling/close pattern to File. The original
// kernel AF_LOCAL socketpair owns the byte buffers and EOF state.
type nativeLocalStream struct {
	mu         sync.Mutex
	descriptor uint32
	references int
}

func newNativeLocalFile(name string, descriptor uint32, readable, writable bool) *File {
	return &File{name: name, fd: int(descriptor), readable: readable, writable: writable,
		localStream: &nativeLocalStream{descriptor: descriptor, references: 1}}
}

func (file *File) cloneNativeLocal() (*File, error) {
	stream := file.localStream
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if file.closed || stream.references == 0 {
		return nil, ErrClosed
	}
	stream.references++
	copy := *file
	return &copy, nil
}

func (file *File) closeNativeLocal() error {
	stream := file.localStream
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if file.closed {
		return &PathError{Op: "close", Path: file.name, Err: ErrClosed}
	}
	file.closed = true
	stream.references--
	if stream.references == 0 {
		if kos.CloseLocalSocket(stream.descriptor) != 0 {
			return &PathError{Op: "close", Path: file.name, Err: syscall.EBADF}
		}
	}
	return nil
}

func (file *File) readNativeLocal(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	stream := file.localStream
	for {
		stream.mu.Lock()
		if file.closed || stream.references == 0 {
			stream.mu.Unlock()
			return 0, &PathError{Op: "read", Path: file.name, Err: ErrClosed}
		}
		n, code := kos.ReadLocalSocket(stream.descriptor, buffer, kos.LocalSocketDontWait)
		stream.mu.Unlock()
		if code == kos.LocalSocketWouldBlock {
			time.Sleep(time.Millisecond)
			continue
		}
		if code != 0 {
			return 0, &PathError{Op: "read", Path: file.name, Err: syscall.EBADF}
		}
		if n == 0 {
			return 0, io.EOF
		}
		return n, nil
	}
}

func (file *File) writeNativeLocal(buffer []byte) (int, error) {
	stream := file.localStream
	written := 0
	for written < len(buffer) {
		stream.mu.Lock()
		if file.closed || stream.references == 0 {
			stream.mu.Unlock()
			return written, &PathError{Op: "write", Path: file.name, Err: ErrClosed}
		}
		n, code := kos.WriteLocalSocket(stream.descriptor, buffer[written:], kos.LocalSocketDontWait)
		stream.mu.Unlock()
		if code != 0 && code != kos.LocalSocketWouldBlock {
			return written, &PathError{Op: "write", Path: file.name, Err: syscall.EPIPE}
		}
		if n > 0 {
			written += n
		} else {
			time.Sleep(time.Millisecond)
		}
	}
	return written, nil
}
