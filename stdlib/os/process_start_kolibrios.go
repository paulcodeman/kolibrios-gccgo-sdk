//go:build kolibrios && gccgo

package os

import (
	"errors"
	"kos"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// NativeProcessStatus is returned by ProcessState.Sys on KolibriOS. The
// loader does not expose Unix wait status or resource usage.
type NativeProcessStatus struct {
	ExitCode      int
	Known, Killed bool
}

func nativeProcessExit() __asm__("runtime_kolibri_exit_process")
type ProcessState struct {
	pid    int
	status NativeProcessStatus
}

func (s *ProcessState) Pid() int                  { return s.pid }
func (s *ProcessState) exited() bool              { return s.status.Known && !s.status.Killed }
func (s *ProcessState) success() bool             { return s.exited() && s.status.ExitCode == 0 }
func (s *ProcessState) sys() any                  { return s.status }
func (s *ProcessState) sysUsage() any             { return nil }
func (s *ProcessState) userTime() time.Duration   { return 0 }
func (s *ProcessState) systemTime() time.Duration { return 0 }
func (s *ProcessState) ExitCode() int {
	if s == nil || !s.exited() {
		return -1
	}
	return s.status.ExitCode
}
func (s *ProcessState) String() string {
	if s == nil {
		return "<nil>"
	}
	if s.status.Killed {
		return "signal: killed"
	}
	if !s.status.Known {
		return "exit status unavailable"
	}
	return "exit status " + strconv.Itoa(s.status.ExitCode)
}

// StartProcess's portable surface is imported from Go. This backend adapts
// Go's child attributes to the SDK loader because the native loader accepts
// only a path and a command line. Streams use native kernel socketpairs.
func startProcess(name string, argv []string, attr *ProcAttr) (*Process, error) {
	if attr == nil {
		attr = &ProcAttr{}
	}
	if len(attr.Files) > 3 {
		return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.ENOTSUP}
	}
	if name == "" || strings.IndexByte(name, 0) >= 0 || strings.IndexByte(attr.Dir, 0) >= 0 {
		return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.EINVAL}
	}
	env := attr.Env
	if env == nil {
		env = Environ()
	}
	for _, list := range [][]string{argv, env} {
		for _, value := range list {
			if strings.IndexByte(value, 0) >= 0 {
				return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.EINVAL}
			}
		}
	}
	dir := attr.Dir
	if dir == "" {
		var err error
		dir, err = Getwd()
		if err != nil {
			return nil, err
		}
	}
	if !path.IsAbs(dir) {
		wd, err := Getwd()
		if err != nil {
			return nil, err
		}
		dir = path.Join(wd, dir)
	}
	info, err := Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &PathError{Op: "chdir", Path: dir, Err: syscall.ENOTDIR}
	}
	if !path.IsAbs(name) {
		wd, err := Getwd()
		if err != nil {
			return nil, err
		}
		name = path.Join(wd, name)
	}
	storage, err := MkdirTemp("", "kos-process-")
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = RemoveAll(storage)
		}
	}()
	statusPath := path.Join(storage, "status")
	if err = WriteFile(statusPath, make([]byte, 8), 0600); err != nil {
		return nil, err
	}
	control, err := newNativeProcessControl(statusPath)
	if err != nil { return nil, err }
	defer func() { if cleanup { control.close() } }()
	standardFiles, streams, err := prepareNativeProcessStreams(attr.Files)
	if err != nil { return nil, &PathError{Op: "fork/exec", Path: name, Err: err} }
	defer func() { if cleanup { abortNativeProcessStreams(streams) } }()
	payload := kos.EncodeProcessStartup(&kos.ProcessStartup{Args: argv, Env: env, Dir: dir, StatusPath: statusPath, StandardFiles: standardFiles})
	if len(payload) > 1024*1024 {
		return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.EINVAL}
	}
	metadata := path.Join(storage, "startup")
	if err = WriteFile(metadata, payload, 0600); err != nil {
		return nil, err
	}
	argument := kos.ProcessStartupArgument(metadata)
	if len(argument) >= 1024 {
		return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.EINVAL}
	}
	pid, status := kos.StartApplication(name, argument, false)
	if status != kos.FileSystemOK {
		return nil, wrapPathError("fork/exec", name, status)
	}
	if pid <= 0 {
		return nil, &PathError{Op: "fork/exec", Path: name, Err: syscall.EIO}
	}
	cleanup = false
	process := &Process{Pid: pid, nativeStorage: storage, nativeStatusPath: statusPath, nativeStreams: streams, nativeControl: control}
	for _, stream := range streams { go stream.copy() }
	return process, nil
}

func (p *Process) wait() (*ProcessState, error) {
	if p == nil {
		return nil, syscall.EINVAL
	}
	if !atomic.CompareAndSwapUint32(&p.nativeWaitStarted, 0, 1) {
		return nil, ErrProcessDone
	}
	p.sigMu.RLock()
	pid, statusPath, storage := p.Pid, p.nativeStatusPath, p.nativeStorage
	p.sigMu.RUnlock()
	if pid <= 0 {
		return nil, syscall.EINVAL
	}
	for kos.ThreadSlotByIdentifier(pid) != 0 {
		time.Sleep(10 * time.Millisecond)
	}
	// A native main-thread fault/exit can bypass the Go process-exit hook.
	// Its remaining Go workers cannot outlive a completed Go process.
	p.nativeControl.terminateWorkers(pid)
	p.nativeControl.waitWorkers(pid)
	p.setDone()
	p.sigMu.RLock()
	killed := p.nativeKilled
	p.sigMu.RUnlock()
	state := &ProcessState{pid: pid, status: NativeProcessStatus{ExitCode: -1, Killed: killed}}
	if statusPath != "" {
		data, err := ReadFile(statusPath)
		if err == nil && len(data) == 8 && string(data[:4]) == "KGS1" {
			state.status.ExitCode = int(int32(uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24))
			state.status.Known = true
		}
	}
	streamErr := p.finishNativeStreams(state.status.Known)
	p.nativeControl.close()
	if storage != "" { _ = RemoveAll(storage) }
	if !state.status.Known && !killed {
		return state, errors.New("os: native child exit status unavailable")
	}
	return state, streamErr
}
