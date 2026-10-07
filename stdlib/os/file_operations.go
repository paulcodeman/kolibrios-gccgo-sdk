// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"syscall"
)

var errWriteAtInAppendMode = errors.New("os: invalid use of WriteAt on file opened with O_APPEND")

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	if err := f.checkValid("write"); err != nil {
		return 0, err
	}
	if f.append {
		return 0, errWriteAtInAppendMode
	}

	if off < 0 {
		return 0, &PathError{Op: "writeat", Path: f.name, Err: errors.New("negative offset")}
	}

	for len(b) > 0 {
		m, e := f.pwrite(b, off)
		if e != nil {
			err = f.wrapErr("write", e)
			break
		}
		n += m
		b = b[m:]
		off += int64(m)
	}
	return
}

func (f *File) Truncate(size int64) error {
	if err := f.checkValid("truncate"); err != nil {
		return err
	}
	if e := f.truncate(size); e != nil {
		return f.wrapErr("truncate", e)
	}
	return nil
}

func Truncate(name string, size int64) error {
	f, e := OpenFile(name, O_WRONLY, 0666)
	if e != nil {
		return e
	}
	defer f.Close()
	e1 := f.Truncate(size)
	if e1 != nil {
		return e1
	}
	return nil
}

func Chmod(name string, mode FileMode) error { return chmod(name, mode) }

func (f *File) Chmod(mode FileMode) error { return f.chmod(mode) }

func Chown(name string, uid, gid int) error {
	e := ignoringEINTR(func() error {
		return syscall.Chown(name, uid, gid)
	})
	if e != nil {
		return &PathError{Op: "chown", Path: name, Err: e}
	}
	return nil
}

func Lchown(name string, uid, gid int) error {
	e := ignoringEINTR(func() error {
		return syscall.Lchown(name, uid, gid)
	})
	if e != nil {
		return &PathError{Op: "lchown", Path: name, Err: e}
	}
	return nil
}

func ignoringEINTR(fn func() error) error {
	for {
		err := fn()
		if err != syscall.EINTR {
			return err
		}
	}
}

func Link(oldname, newname string) error {
	e := ignoringEINTR(func() error {
		return syscall.Link(oldname, newname)
	})
	if e != nil {
		return &LinkError{"link", oldname, newname, e}
	}
	return nil
}

func Symlink(oldname, newname string) error {
	e := ignoringEINTR(func() error {
		return syscall.Symlink(oldname, newname)
	})
	if e != nil {
		return &LinkError{"symlink", oldname, newname, e}
	}
	return nil
}
