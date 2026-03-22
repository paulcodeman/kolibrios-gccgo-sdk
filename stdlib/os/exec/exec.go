package exec

import (
	"context"
	"errors"
	"io"
	"kos"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("executable file not found in PATH")

type Error struct {
	Name string
	Err  error
}

func (err *Error) Error() string {
	if err == nil {
		return "<nil>"
	}
	if err.Name == "" {
		return err.Err.Error()
	}
	return err.Name + ": " + err.Err.Error()
}

type Process struct {
	Pid int
}

func (process *Process) Kill() error {
	if process == nil || process.Pid <= 0 {
		return errors.New("exec: invalid process")
	}
	if !kos.TerminateByIdentifier(process.Pid) {
		return errors.New("exec: kill failed")
	}
	return nil
}

type ProcessState struct {
	pid    int
	exited bool
}

func (state *ProcessState) Exited() bool {
	return state != nil && state.exited
}

type ExitError struct {
	ProcessState *ProcessState
}

func (err *ExitError) Error() string {
	if err == nil || err.ProcessState == nil {
		return "exec: process exited"
	}
	return "exec: process " + itoa(err.ProcessState.pid) + " exited"
}

type Cmd struct {
	Path string
	Args []string
	Dir  string
	Env  []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Process      *Process
	ProcessState *ProcessState

	ctx context.Context

	mu      sync.Mutex
	started bool
	waited  bool
	waitErr error
	done    chan struct{}
}

func Command(name string, arg ...string) *Cmd {
	path, err := LookPath(name)
	cmd := &Cmd{
		Path: path,
		Args: append([]string{name}, arg...),
		done: make(chan struct{}),
	}
	if err != nil {
		cmd.waitErr = &Error{Name: name, Err: err}
	}
	return cmd
}

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	cmd := Command(name, arg...)
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.ctx = ctx
	return cmd
}

func LookPath(file string) (string, error) {
	if strings.TrimSpace(file) == "" {
		return "", ErrNotFound
	}
	if strings.Contains(file, "/") || strings.Contains(file, "\\") {
		path := file
		if !filepath.IsAbs(path) {
			if wd, err := os.Getwd(); err == nil && wd != "" {
				path = filepath.Join(wd, path)
			}
		}
		if statPath(path) {
			return path, nil
		}
		return "", &Error{Name: file, Err: ErrNotFound}
	}

	candidates := pathEntries(os.Getenv("PATH"))
	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append([]string{wd}, candidates...)
	}
	for i := 0; i < len(candidates); i++ {
		candidate := filepath.Join(candidates[i], file)
		if statPath(candidate) {
			return candidate, nil
		}
	}
	return "", &Error{Name: file, Err: ErrNotFound}
}

func (cmd *Cmd) Run() error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

func (cmd *Cmd) Start() error {
	if cmd == nil {
		return errors.New("exec: nil Cmd")
	}

	cmd.mu.Lock()
	if cmd.started {
		cmd.mu.Unlock()
		return errors.New("exec: already started")
	}
	cmd.started = true
	cmd.mu.Unlock()

	if cmd.waitErr != nil {
		return cmd.waitErr
	}
	if cmd.Path == "" {
		return errors.New("exec: empty executable path")
	}
	if cmd.ctx != nil {
		select {
		case <-cmd.ctx.Done():
			return cmd.ctx.Err()
		default:
		}
	}

	params := buildCommandLine(cmd.Args)
	pid, status := kos.StartApplication(cmd.Path, params, false)
	if status != kos.FileSystemOK || pid <= 0 {
		return &Error{Name: cmd.Path, Err: errors.New("start failed")}
	}
	cmd.Process = &Process{Pid: pid}

	if cmd.ctx != nil {
		go cmd.watchContext()
	}
	return nil
}

func (cmd *Cmd) Wait() error {
	if cmd == nil {
		return errors.New("exec: nil Cmd")
	}

	cmd.mu.Lock()
	if !cmd.started {
		cmd.mu.Unlock()
		return errors.New("exec: not started")
	}
	if cmd.waited {
		err := cmd.waitErr
		cmd.mu.Unlock()
		return err
	}
	cmd.waited = true
	process := cmd.Process
	cmd.mu.Unlock()

	if process == nil || process.Pid <= 0 {
		cmd.finish(errors.New("exec: invalid process"))
		return cmd.waitErr
	}

	for {
		if kos.ThreadSlotByIdentifier(process.Pid) == 0 {
			break
		}
		if cmd.ctx != nil {
			select {
			case <-cmd.ctx.Done():
				_ = process.Kill()
				cmd.finish(cmd.ctx.Err())
				return cmd.waitErr
			default:
			}
		}
		kos.Sleep(1)
	}

	cmd.finish(nil)
	return cmd.waitErr
}

func (cmd *Cmd) watchContext() {
	<-cmd.ctx.Done()
	cmd.mu.Lock()
	process := cmd.Process
	waited := cmd.waited
	cmd.mu.Unlock()
	if waited || process == nil {
		return
	}
	_ = process.Kill()
}

func (cmd *Cmd) finish(err error) {
	cmd.mu.Lock()
	defer cmd.mu.Unlock()
	if cmd.Process != nil {
		cmd.ProcessState = &ProcessState{pid: cmd.Process.Pid, exited: true}
	}
	cmd.waitErr = err
	select {
	case <-cmd.done:
	default:
		close(cmd.done)
	}
}

func statPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info != nil && !info.IsDir()
}

func pathEntries(pathValue string) []string {
	if pathValue == "" {
		return nil
	}
	parts := strings.FieldsFunc(pathValue, func(r rune) bool {
		return r == ';' || r == ':'
	})
	out := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func buildCommandLine(args []string) string {
	if len(args) <= 1 {
		return ""
	}
	parts := make([]string, 0, len(args)-1)
	for i := 1; i < len(args); i++ {
		parts = append(parts, quoteArg(args[i]))
	}
	return strings.Join(parts, " ")
}

func quoteArg(arg string) string {
	if arg == "" {
		return `""`
	}
	needsQuotes := false
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case ' ', '\t', '"':
			needsQuotes = true
		}
	}
	if !needsQuotes {
		return arg
	}
	escaped := strings.ReplaceAll(arg, `"`, `\"`)
	return `"` + escaped + `"`
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [24]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buf[index] = '-'
	}
	return string(buf[index:])
}
