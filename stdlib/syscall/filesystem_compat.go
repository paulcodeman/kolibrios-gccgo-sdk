// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

const (
	EBADFD   = Errno(81)
	EIO      = Errno(0x5)
	EPERM    = Errno(0x1)
	EINTR    = Errno(0x4)
	ENOENT   = Errno(0x2)
	ENOEXEC  = Errno(0x8) // Go Unix executable-format error.
	ENOTDIR  = Errno(0x14)
	EISDIR   = Errno(0x15)
	ESPIPE   = Errno(0x1d)
	ENOSYS   = Errno(88)
	ERANGE   = Errno(0x22)
	O_RDONLY = 0x0
	O_WRONLY = 0x1
	O_RDWR   = 0x2
	O_SYNC   = 0x101000
)

func Chown(path string, uid, gid int) error {
	return ENOSYS
}

func Lchown(path string, uid, gid int) error {
	return ENOSYS
}
