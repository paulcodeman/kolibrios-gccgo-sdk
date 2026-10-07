//go:build kolibrios

package term

import (
	"os"

	consoleterm "github.com/charmbracelet/x/term"
)

// Both upstream terminal packages use the same native console state. The
// KolibriOS backend supplies the OS operations; the portable API stays original.
type state struct{ native *consoleterm.State }

func isTerminal(fd int) bool { return fd >= 0 && consoleterm.IsTerminal(uintptr(fd)) }
func makeRaw(fd int) (*State, error) {
	value, err := consoleterm.MakeRaw(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &State{state{native: value}}, nil
}
func getState(fd int) (*State, error) {
	value, err := consoleterm.GetState(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &State{state{native: value}}, nil
}
func restore(fd int, value *State) error {
	if value == nil {
		return os.ErrInvalid
	}
	return consoleterm.Restore(uintptr(fd), value.native)
}
func getSize(fd int) (int, int, error)    { return consoleterm.GetSize(uintptr(fd)) }
func readPassword(fd int) ([]byte, error) { return consoleterm.ReadPassword(uintptr(fd)) }
