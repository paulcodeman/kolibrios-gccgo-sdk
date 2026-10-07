// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import (
	"internal/bytealg"
	"kos"
)

// These are the syscall 70 adapters for the Unix filesystem API. Native
// filesystem statuses belong to a different namespace than newlib errno.
func filesystemErrno(status kos.FileSystemStatus) error {
	switch status {
	case kos.FileSystemOK:
		return nil
	case kos.FileSystemUnsupported:
		return ENOTSUP
	case kos.FileSystemAlreadyExists:
		return EEXIST
	case kos.FileSystemNotFound:
		return ENOENT
	case kos.FileSystemBadPointer:
		return EFAULT
	case kos.FileSystemAccessDenied:
		return EACCES
	case kos.FileSystemDiskFull:
		return Errno(28) // newlib ENOSPC
	case kos.FileSystemNeedsMoreMemory:
		return ENOMEM
	default:
		return EIO
	}
}

func Unlink(path string) error {
	if bytealg.IndexByteString(path, 0) != -1 {
		return EINVAL
	}
	info, status := kos.GetPathInfo(path)
	if status != kos.FileSystemOK {
		return filesystemErrno(status)
	}
	if info.Attributes&kos.FileAttributeDirectory != 0 {
		return EISDIR
	}
	return filesystemErrno(kos.DeletePath(path))
}

func Rmdir(path string) error {
	if bytealg.IndexByteString(path, 0) != -1 {
		return EINVAL
	}
	info, status := kos.GetPathInfo(path)
	if status != kos.FileSystemOK {
		return filesystemErrno(status)
	}
	if info.Attributes&kos.FileAttributeDirectory == 0 {
		return ENOTDIR
	}
	status = kos.DeletePath(path)
	if status == kos.FileSystemAccessDenied {
		// The native API reports both permission failures and a nonempty
		// directory as status 10. Inspect entries to disambiguate them.
		entries, readStatus := kos.ReadDirectory(path, 0, 3)
		if readStatus == kos.FileSystemOK || readStatus == kos.FileSystemEOF {
			for _, entry := range entries.Entries {
				if entry.Name != "." && entry.Name != ".." {
					return ENOTEMPTY
				}
			}
		}
	}
	return filesystemErrno(status)
}

func Rename(oldpath, newpath string) error {
	if bytealg.IndexByteString(oldpath, 0) != -1 || bytealg.IndexByteString(newpath, 0) != -1 {
		return EINVAL
	}
	return filesystemErrno(kos.RenamePath(oldpath, newpath))
}

// Syscall 77 provides blocking pipes and no fcntl operation. Report this
// limitation for both directions instead of claiming that flags changed.
func SetNonblock(fd int, nonblocking bool) error {
	if fd < 0 {
		return EBADF
	}
	return ENOTSUP
}
