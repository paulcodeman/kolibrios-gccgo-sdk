// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "syscall"

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func Chdir(dir string) error {
	if e := syscall.Chdir(dir); e != nil {
		return &PathError{Op: "chdir", Path: dir, Err: e}
	}
	return nil
}
