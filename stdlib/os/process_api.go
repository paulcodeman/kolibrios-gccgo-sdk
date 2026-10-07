// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrProcessDone indicates a Process has finished.
var ErrProcessDone = errors.New("os: process already finished")

// Process stores the information about a process created by StartProcess.
type Process struct {
	Pid               int
	handle            uintptr      // handle is accessed atomically on Windows
	isdone            uint32       // process has been successfully waited on, non zero if true
	sigMu             sync.RWMutex // avoid race between wait and signal
	nativeStorage     string
	nativeStatusPath  string
	nativeWaitStarted uint32
	nativeKilled      bool
	nativeStreams     []*nativeProcessStream
	nativeControl     *nativeProcessControl
}

func (p *Process) setDone() {
	atomic.StoreUint32(&p.isdone, 1)
}

func (p *Process) done() bool {
	return atomic.LoadUint32(&p.isdone) > 0
}

func FindProcess(pid int) (*Process, error) {
	return findProcess(pid)
}

func (p *Process) Kill() error {
	return p.kill()
}

func (p *Process) Release() error {
	return p.release()
}

func (p *Process) Signal(sig Signal) error {
	return p.signal(sig)
}
