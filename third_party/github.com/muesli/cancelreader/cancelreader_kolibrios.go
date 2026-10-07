//go:build kolibrios

// KolibriOS equivalent of the upstream wakeup-pipe reader. It uses the native
// console's nonblocking input query and the SDK's cancellable raw-mode reads.
package cancelreader

import (
	"io"
	"kos"
	"os"
	"sync"
)

type consoleCancelReader struct {
	file   *os.File
	cancel chan struct{}
	once   sync.Once
}

func NewReader(reader io.Reader) (CancelReader, error) {
	if file, ok := reader.(*os.File); ok && file.Fd() == uintptr(kos.StdinFD) && kos.ConsoleInputRaw() {
		return &consoleCancelReader{file: file, cancel: make(chan struct{})}, nil
	}
	return newFallbackCancelReader(reader)
}

func (reader *consoleCancelReader) Read(buffer []byte) (int, error) {
	select {
	case <-reader.cancel:
		return 0, ErrCanceled
	default:
	}
	n, err := reader.file.ReadCancelable(buffer, reader.cancel)
	if err == os.ErrReadCanceled {
		return n, ErrCanceled
	}
	return n, err
}

func (reader *consoleCancelReader) Cancel() bool {
	reader.once.Do(func() { close(reader.cancel) })
	return true
}

func (reader *consoleCancelReader) Close() error { reader.Cancel(); return nil }
