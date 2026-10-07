//go:build kolibrios

// Native equivalent of the upstream term_unix.go terminal operations.
package term

import (
	"kos"
	"os"
	"syscall"
)

type state struct{ raw bool }

func isTerminal(fd uintptr) bool { return fd <= uintptr(kos.StderrFD) && kos.HasActiveConsole() }

func terminalError(fd uintptr) error {
	if !isTerminal(fd) {
		return os.NewSyscallError("terminal", syscall.ENOTTY)
	}
	return nil
}

func makeRaw(fd uintptr) (*State, error) {
	if err := terminalError(fd); err != nil {
		return nil, err
	}
	old, err := kos.SetConsoleInputRaw(true)
	if err != nil {
		return nil, err
	}
	return &State{state{raw: old}}, nil
}

func getState(fd uintptr) (*State, error) {
	if err := terminalError(fd); err != nil {
		return nil, err
	}
	return &State{state{raw: kos.ConsoleInputRaw()}}, nil
}

func setState(fd uintptr, value *State) error {
	if err := terminalError(fd); err != nil {
		return err
	}
	if value == nil {
		return os.ErrInvalid
	}
	_, err := kos.SetConsoleInputRaw(value.raw)
	return err
}

func restore(fd uintptr, value *State) error { return setState(fd, value) }

func getSize(fd uintptr) (width, height int, err error) {
	if err := terminalError(fd); err != nil {
		return 0, 0, err
	}
	return kos.ActiveConsoleSize()
}

type passwordReader struct{}

func (passwordReader) Read(buffer []byte) (int, error) {
	n, err := os.Stdin.Read(buffer)
	for i := 0; i < n; i++ {
		if buffer[i] == '\r' {
			buffer[i] = '\n'
		}
	}
	return n, err
}

func readPassword(fd uintptr) ([]byte, error) {
	if fd != uintptr(kos.StdinFD) {
		return nil, os.NewSyscallError("read", syscall.EBADF)
	}
	old, err := makeRaw(fd)
	if err != nil {
		return nil, err
	}
	defer restore(fd, old)
	return readPasswordLine(passwordReader{})
}
