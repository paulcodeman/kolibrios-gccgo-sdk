// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"runtime"
	"syscall"
)

// Adapt libgo net/rawconn.go to the SDK's native connection and polling loop.
// The callback receives a syscall 75 socket handle, not a syscall 77 pipe FD.
type rawConn struct{ conn *TCPConn }

func (c *TCPConn) SyscallConn() (syscall.RawConn, error) {
	if c == nil {
		return nil, syscall.EINVAL
	}
	return &rawConn{conn: c}, nil
}

func (c *rawConn) Control(f func(uintptr)) error {
	if c == nil || c.conn == nil {
		return syscall.EINVAL
	}
	conn := c.conn
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.closed {
		return &OpError{Op: "raw-control", Net: "tcp", Addr: conn.laddr, Err: ErrClosed}
	}
	f(uintptr(conn.fd))
	runtime.KeepAlive(conn)
	return nil
}

func (c *rawConn) Read(f func(uintptr) bool) error  { return c.io(f, false) }
func (c *rawConn) Write(f func(uintptr) bool) error { return c.io(f, true) }

func (c *rawConn) io(f func(uintptr) bool, write bool) error {
	if c == nil || c.conn == nil {
		return syscall.EINVAL
	}
	conn := c.conn
	op := "raw-read"
	if write {
		op = "raw-write"
	}
	spins := 0
	for {
		if err := conn.checkIO(write); err != nil {
			return &OpError{Op: op, Net: "tcp", Source: conn.laddr, Addr: conn.raddr, Err: err}
		}
		done, err := c.callback(f)
		if err != nil {
			return &OpError{Op: op, Net: "tcp", Source: conn.laddr, Addr: conn.raddr, Err: err}
		}
		if done {
			return nil
		}
		spins = yieldSocketWait(spins)
	}
}

func (c *rawConn) callback(f func(uintptr) bool) (bool, error) {
	conn := c.conn
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.closed {
		return false, ErrClosed
	}
	done := f(uintptr(conn.fd))
	runtime.KeepAlive(conn)
	return done, nil
}
