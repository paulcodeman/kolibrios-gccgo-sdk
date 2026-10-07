package os

import (
	"io"
	"kos"
	"time"
)

// ErrReadCanceled is returned by the KolibriOS console cancellation adapter.
var ErrReadCanceled = &osError{text: "console read canceled"}

// ReadCancelable provides the console equivalent of cancelreader's native
// wakeup pipe. It is supported for stdin while the console is in raw mode.
func (file *File) ReadCancelable(buffer []byte, cancel <-chan struct{}) (int, error) {
	if err := file.ensureReadable("read"); err != nil {
		return 0, err
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	if !file.usesActiveConsoleInput() || !kos.ConsoleInputRaw() {
		return 0, &PathError{Op: "read", Path: file.name, Err: ErrInvalid}
	}
	for {
		n, err := kos.TryReadActiveConsoleInput(buffer, cancel)
		switch err {
		case kos.ErrConsoleReadCanceled:
			return n, ErrReadCanceled
		case kos.ErrConsoleClosed:
			return n, io.EOF
		}
		if n > 0 || err != nil {
			return n, err
		}
		select {
		case <-cancel:
			return 0, ErrReadCanceled
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
