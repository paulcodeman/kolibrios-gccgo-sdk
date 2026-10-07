//go:build kolibrios

// Adapted from upstream Bubble Tea tty_unix.go input initialization;
// job-control capability follows the upstream Windows backend.
package tea

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
)

func (p *Program) initInput() (err error) {
	// Check if input is a terminal
	if f, ok := p.input.(term.File); ok && term.IsTerminal(f.Fd()) {
		p.ttyInput = f
		p.previousTtyInputState, err = term.MakeRaw(p.ttyInput.Fd())
		if err != nil {
			return fmt.Errorf("error entering raw mode: %w", err)
		}
	}

	if f, ok := p.output.(term.File); ok && term.IsTerminal(f.Fd()) {
		p.ttyOutput = f
	}

	return nil
}

func openInputTTY() (*os.File, error) {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return nil, fmt.Errorf("could not open a new TTY: %w", err)
	}
	return f, nil
}

// The native terminal does not support Unix job-control suspension.
// Keep the same declared capability and behavior as the Windows backend.
const suspendSupported = false

func suspendProcess() {}
