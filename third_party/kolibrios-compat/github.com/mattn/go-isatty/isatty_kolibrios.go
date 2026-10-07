//go:build kolibrios

package isatty

import "kos"

// Standard descriptors use the SDK's active KolibriOS console. Pipe and
// file descriptors are not terminals.
func IsTerminal(fd uintptr) bool {
	return fd <= uintptr(kos.StderrFD) && kos.HasActiveConsole()
}

func IsCygwinTerminal(fd uintptr) bool { return false }
