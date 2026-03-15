package readline

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	"kos"
)

var ErrInterrupt = errors.New("readline: interrupt")

// AutoCompleter is invoked when the user presses tab.
type AutoCompleter interface {
	Do(line []rune, pos int) ([][]rune, int)
}

// Config mirrors the subset of gopkg.in/readline.v1 used by Otto.
type Config struct {
	Prompt       string
	AutoComplete AutoCompleter
}

type Instance struct {
	prompt       string
	autoComplete AutoCompleter
	reader       *bufio.Reader
	stdout       io.Writer
	stderr       io.Writer

	mu            sync.Mutex
	history       []string
	historyPos    int
	savedLine     []rune
	line          []rune
	pos           int
	lastRenderLen int
}

func NewEx(config *Config) (*Instance, error) {
	prompt := ""
	var auto AutoCompleter
	if config != nil {
		prompt = config.Prompt
		auto = config.AutoComplete
	}
	return &Instance{
		prompt:       prompt,
		autoComplete: auto,
		reader:       bufio.NewReader(os.Stdin),
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		historyPos:   -1,
	}, nil
}

func (rl *Instance) Close() error {
	return nil
}

func (rl *Instance) SetPrompt(prompt string) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	rl.prompt = prompt
	rl.mu.Unlock()
}

func (rl *Instance) Refresh() error {
	if rl == nil {
		return nil
	}
	rl.mu.Lock()
	line := append([]rune(nil), rl.line...)
	pos := rl.pos
	prompt := rl.prompt
	last := rl.lastRenderLen
	rl.mu.Unlock()

	if len(line) == 0 && last == 0 && prompt == "" {
		return nil
	}
	rl.redraw(line, pos)
	return nil
}

func (rl *Instance) Stdout() io.Writer {
	if rl == nil {
		return os.Stdout
	}
	return rl.stdout
}

func (rl *Instance) Stderr() io.Writer {
	if rl == nil {
		return os.Stderr
	}
	return rl.stderr
}

func (rl *Instance) Readline() (string, error) {
	if rl == nil {
		return "", io.EOF
	}
	console, ok := kos.ActiveConsole()
	if ok && console.SupportsInputFull() {
		return rl.readlineConsole(console)
	}
	return rl.readlineBuffered()
}

func (rl *Instance) readlineBuffered() (string, error) {
	prompt := rl.getPrompt()
	if prompt != "" {
		_, _ = rl.stdout.Write([]byte(prompt))
	}
	line, err := rl.reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

func (rl *Instance) readlineConsole(console kos.Console) (string, error) {
	rl.mu.Lock()
	rl.line = rl.line[:0]
	rl.pos = 0
	rl.lastRenderLen = 0
	rl.historyPos = -1
	rl.savedLine = nil
	rl.mu.Unlock()

	rl.redraw(nil, 0)

	for {
		key := console.Getch2()
		if key == 0 {
			return "", io.EOF
		}
		ascii := byte(key & 0xff)
		ext := byte((key >> 8) & 0xff)

		if ascii != 0 {
			line, done, err := rl.handleASCII(ascii)
			if err != nil {
				return "", err
			}
			if done {
				return line, nil
			}
			rl.redraw(rl.line, rl.pos)
			continue
		}

		if rl.handleExtended(ext) {
			rl.redraw(rl.line, rl.pos)
		}
	}
}

func (rl *Instance) handleASCII(ascii byte) (string, bool, error) {
	switch ascii {
	case 3, 27:
		return "", true, ErrInterrupt
	case 4:
		if len(rl.line) == 0 {
			return "", true, io.EOF
		}
		return "", false, nil
	case 13, 10:
		line := string(rl.line)
		_, _ = rl.stdout.Write([]byte("\r\n"))
		rl.lastRenderLen = 0
		rl.mu.Lock()
		if line != "" {
			rl.history = append(rl.history, line)
		}
		rl.line = rl.line[:0]
		rl.pos = 0
		rl.mu.Unlock()
		return line, true, nil
	case 8, 127:
		if rl.pos > 0 {
			rl.line = append(rl.line[:rl.pos-1], rl.line[rl.pos:]...)
			rl.pos--
		}
		return "", false, nil
	case 9:
		changed := rl.applyCompletion()
		if !changed {
			return "", false, nil
		}
		return "", false, nil
	default:
		if ascii < 32 {
			return "", false, nil
		}
		ch := rune(ascii)
		rl.line = append(rl.line[:rl.pos], append([]rune{ch}, rl.line[rl.pos:]...)...)
		rl.pos++
		return "", false, nil
	}
}

func (rl *Instance) handleExtended(ext byte) bool {
	switch ext {
	case 75: // left
		if rl.pos > 0 {
			rl.pos--
			return true
		}
	case 77: // right
		if rl.pos < len(rl.line) {
			rl.pos++
			return true
		}
	case 72, 73: // up / pgup
		return rl.historyUp()
	case 80, 81: // down / pgdn
		return rl.historyDown()
	case 71: // home
		if rl.pos != 0 {
			rl.pos = 0
			return true
		}
	case 79: // end
		if rl.pos != len(rl.line) {
			rl.pos = len(rl.line)
			return true
		}
	case 83: // delete
		if rl.pos < len(rl.line) {
			rl.line = append(rl.line[:rl.pos], rl.line[rl.pos+1:]...)
			return true
		}
	}
	return false
}

func (rl *Instance) historyUp() bool {
	if len(rl.history) == 0 {
		return false
	}
	if rl.historyPos == -1 {
		rl.savedLine = append([]rune(nil), rl.line...)
		rl.historyPos = len(rl.history) - 1
	} else if rl.historyPos > 0 {
		rl.historyPos--
	} else {
		return false
	}
	rl.line = []rune(rl.history[rl.historyPos])
	rl.pos = len(rl.line)
	return true
}

func (rl *Instance) historyDown() bool {
	if rl.historyPos == -1 {
		return false
	}
	if rl.historyPos < len(rl.history)-1 {
		rl.historyPos++
		rl.line = []rune(rl.history[rl.historyPos])
		rl.pos = len(rl.line)
		return true
	}
	if rl.savedLine != nil {
		rl.line = append([]rune(nil), rl.savedLine...)
	} else {
		rl.line = rl.line[:0]
	}
	rl.pos = len(rl.line)
	rl.historyPos = -1
	return true
}

func (rl *Instance) applyCompletion() bool {
	if rl.autoComplete == nil {
		return false
	}
	line := append([]rune(nil), rl.line...)
	pos := rl.pos
	candidates, _ := rl.autoComplete.Do(line, pos)
	if len(candidates) == 0 {
		return false
	}
	insert := commonPrefix(candidates)
	if len(insert) == 0 {
		rl.printCompletions(candidates)
		return true
	}
	rl.line = append(rl.line[:pos], append(insert, rl.line[pos:]...)...)
	rl.pos = pos + len(insert)
	return true
}

func commonPrefix(items [][]rune) []rune {
	if len(items) == 0 {
		return nil
	}
	prefix := append([]rune(nil), items[0]...)
	for _, item := range items[1:] {
		if len(prefix) == 0 {
			return nil
		}
		max := len(prefix)
		if len(item) < max {
			max = len(item)
		}
		idx := 0
		for idx < max && prefix[idx] == item[idx] {
			idx++
		}
		prefix = prefix[:idx]
	}
	return prefix
}

func (rl *Instance) printCompletions(candidates [][]rune) {
	if len(candidates) == 0 {
		return
	}
	_, _ = rl.stdout.Write([]byte("\r\n"))
	for i, item := range candidates {
		if i > 0 {
			_, _ = rl.stdout.Write([]byte(" "))
		}
		_, _ = rl.stdout.Write([]byte(string(item)))
	}
	_, _ = rl.stdout.Write([]byte("\r\n"))
}

func (rl *Instance) redraw(line []rune, pos int) {
	if rl == nil {
		return
	}
	if line == nil {
		line = rl.line
		pos = rl.pos
	}
	prompt := rl.getPrompt()
	promptRunes := []rune(prompt)
	out := rl.stdout

	_, _ = out.Write([]byte("\r"))
	_, _ = out.Write([]byte(prompt))
	_, _ = out.Write([]byte(string(line)))

	totalLen := len(promptRunes) + len(line)
	if rl.lastRenderLen > totalLen {
		pad := rl.lastRenderLen - totalLen
		_, _ = out.Write([]byte(strings.Repeat(" ", pad)))
	}

	_, _ = out.Write([]byte("\r"))
	_, _ = out.Write([]byte(prompt))
	if pos > 0 {
		_, _ = out.Write([]byte(string(line[:pos])))
	}

	rl.lastRenderLen = totalLen
}

func (rl *Instance) getPrompt() string {
	rl.mu.Lock()
	prompt := rl.prompt
	rl.mu.Unlock()
	return prompt
}
