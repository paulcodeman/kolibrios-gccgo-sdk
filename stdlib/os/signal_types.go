// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "syscall"

// Signal represents an operating system signal. KolibriOS does not implement
// POSIX signal delivery; these identifiers exist for source compatibility.
type Signal interface {
	String() string
	Signal()
}

var (
	Interrupt Signal = syscall.SIGINT
	Kill Signal = syscall.SIGKILL
)
