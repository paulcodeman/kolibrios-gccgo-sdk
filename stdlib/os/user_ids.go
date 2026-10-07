// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "syscall"

// Getuid returns the numeric user id of the caller.
//
// On Windows, it returns -1.
func Getuid() int { return syscall.Getuid() }

// Geteuid returns the numeric effective user id of the caller.
//
// On Windows, it returns -1.
func Geteuid() int { return syscall.Geteuid() }

// Getgid returns the numeric group id of the caller.
//
// On Windows, it returns -1.
func Getgid() int { return syscall.Getgid() }

// Getegid returns the numeric effective group id of the caller.
//
// On Windows, it returns -1.
func Getegid() int { return syscall.Getegid() }

// Getgroups returns a list of the numeric ids of groups that the caller belongs to.
//
// On Windows, it returns syscall.EWINDOWS. See the os/user package
// for a possible alternative.
func Getgroups() ([]int, error) {
	gids, e := syscall.Getgroups()
	return gids, NewSyscallError("getgroups", e)
}

