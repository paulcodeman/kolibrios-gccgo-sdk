//go:build kolibrios && gccgo

package os

import (
	"errors"
	"kos"
	"sync/atomic"
	"syscall"
)

func findProcess(pid int) (*Process, error) {
	if pid <= 0 {
		return nil, NewSyscallError("findprocess", syscall.EINVAL)
	}
	// As with the upstream Unix PID backend, constructing a Process does not
	// prove that it exists or retain the identifier against kernel reuse.
	return &Process{Pid: pid}, nil
}

func (p *Process) kill() error { return p.Signal(Kill) }

func (p *Process) release() error {
	p.sigMu.Lock()
	defer p.sigMu.Unlock()
	if p.Pid > 0 && p.nativeStorage != "" && atomic.CompareAndSwapUint32(&p.nativeWaitStarted, 0, 1) {
		// Releasing a Go handle must not terminate its child. Retain a detached
		// reaper solely for SDK pipes, startup storage and named-area ownership.
		reaper := &Process{Pid: p.Pid, nativeStorage: p.nativeStorage,
			nativeStatusPath: p.nativeStatusPath, nativeStreams: p.nativeStreams,
			nativeControl: p.nativeControl, nativeKilled: p.nativeKilled}
		go func() { _, _ = reaper.Wait() }()
	}
	p.Pid = -1
	return nil
}

func (p *Process) signal(sig Signal) error {
	p.sigMu.Lock()
	defer p.sigMu.Unlock()
	if p.Pid == -1 {
		return errors.New("os: process already released")
	}
	if p.done() || kos.ThreadSlotByIdentifier(p.Pid) == 0 {
		return ErrProcessDone
	}
	s, ok := sig.(syscall.Signal)
	if !ok {
		return NewSyscallError("signal", syscall.EINVAL)
	}
	if s == 0 {
		return nil // Existence query; no signal is delivered.
	}
	if sig != Kill {
		return NewSyscallError("signal", syscall.ENOTSUP)
	}
	p.nativeControl.terminateWorkers(p.Pid)
	if !kos.TerminateByIdentifier(p.Pid) {
		return ErrProcessDone
	}
	p.nativeKilled = true
	return nil
}
