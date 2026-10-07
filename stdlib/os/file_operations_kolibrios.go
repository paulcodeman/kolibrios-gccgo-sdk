// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package os

import (
	"io"
	"io/fs"
	"kos"
	"syscall"
	"time"
)

func (f *File) checkValid(op string) error {
	if f == nil {
		return ErrInvalid
	}
	if f.closed {
		return &PathError{Op: op, Path: f.name, Err: ErrClosed}
	}
	return nil
}

func (f *File) wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return &PathError{Op: op, Path: f.name, Err: err}
}

func (f *File) pwrite(data []byte, off int64) (int, error) {
	if !f.writable {
		return 0, ErrPermission
	}
	if f.fdBacked {
		return 0, syscall.ESPIPE
	}
	if off < 0 {
		return 0, syscall.EINVAL
	}
	n, status := kos.WriteFile(f.name, data, uint64(off))
	if n > 0 {
		// POSIX positional writes return a successful short count if bytes
		// were transferred. The upstream loop accounts for these bytes,
		// and the next native write reports the remaining disk-full error.
		return int(n), nil
	}
	if status != kos.FileSystemOK {
		return 0, statusToError(status)
	}
	if len(data) != 0 {
		return 0, io.ErrShortWrite
	}
	return 0, nil
}

func (f *File) truncate(size int64) error {
	if size < 0 {
		return syscall.EINVAL
	}
	if !f.writable {
		return ErrPermission
	}
	if f.fdBacked {
		return syscall.EINVAL
	}
	info, status := kos.GetPathInfo(f.name)
	if status != kos.FileSystemOK {
		return statusToError(status)
	}
	if info.Attributes&kos.FileAttributeDirectory != 0 {
		return syscall.EISDIR
	}
	// Syscall 70/4 guarantees zero filling only for expansions up to
	// 16 MiB. Split larger expansions so every newly exposed byte is zero.
	const maxExpansion = uint64(16 * 1024 * 1024)
	for uint64(size) > info.Size && uint64(size)-info.Size > maxExpansion {
		info.Size += maxExpansion
		if status = kos.SetFileSize(f.name, info.Size); status != kos.FileSystemOK {
			return statusToError(status)
		}
	}
	return statusToError(kos.SetFileSize(f.name, uint64(size)))
}

// Chmod follows Go's DOS filesystem semantics: the owner write bit controls
// the native read-only attribute. KolibriOS exposes no Unix owner/group modes.
func chmod(name string, mode FileMode) error {
	info, status := kos.GetPathInfo(name)
	if status != kos.FileSystemOK {
		return wrapPathError("chmod", name, status)
	}
	if mode&0200 == 0 {
		info.Attributes |= kos.FileAttributeReadOnly
	} else {
		info.Attributes &^= kos.FileAttributeReadOnly
	}
	return fileStatusError("chmod", name, kos.SetPathInfo(name, info))
}

func (f *File) chmod(mode FileMode) error {
	if err := f.checkValid("chmod"); err != nil {
		return err
	}
	if f.fdBacked {
		return f.wrapErr("chmod", syscall.ENOSYS)
	}
	return chmod(f.name, mode)
}

// Chtimes preserves all other DOS metadata and uses filesystem time precision.
func Chtimes(name string, atime, mtime time.Time) error {
	info, status := kos.GetPathInfo(name)
	if status != kos.FileSystemOK {
		return wrapPathError("chtimes", name, status)
	}
	var err error
	info.AccessDate, info.AccessTime, err = fileStamp(atime)
	if err != nil {
		return &PathError{Op: "chtimes", Path: name, Err: err}
	}
	info.ModifiedDate, info.ModifiedTime, err = fileStamp(mtime)
	if err != nil {
		return &PathError{Op: "chtimes", Path: name, Err: err}
	}
	return fileStatusError("chtimes", name, kos.SetPathInfo(name, info))
}

func fileStatusError(op, name string, status kos.FileSystemStatus) error {
	if status == kos.FileSystemOK {
		return nil
	}
	return wrapPathError(op, name, status)
}

func fileStamp(value time.Time) (kos.FileDate, kos.FileTime, error) {
	// GetPathInfo decodes timestamps in UTC, so write the same convention.
	value = value.UTC()
	year, month, day := value.Date()
	hour, minute, second := value.Clock()
	if year < 1980 || year > 2107 {
		return kos.FileDate{}, kos.FileTime{}, syscall.EINVAL
	}
	return kos.FileDate{Year: uint16(year), Month: byte(month), Day: byte(day)},
		kos.FileTime{Hour: byte(hour), Minute: byte(minute), Second: byte(second)}, nil
}

func (f *File) readdir(n int, mode readdirMode) ([]string, []DirEntry, []FileInfo, error) {
	infos, err := f.Readdir(n)
	switch mode {
	case readdirDirEntry:
		entries := make([]DirEntry, len(infos))
		for i, info := range infos {
			entries[i] = fs.FileInfoToDirEntry(info)
		}
		return nil, entries, nil, err
	case readdirName:
		names := make([]string, len(infos))
		for i, info := range infos {
			names[i] = info.Name()
		}
		return names, nil, nil, err
	default:
		return nil, nil, infos, err
	}
}

// No native link-following or readlink operation is exposed by syscall 70.
func Readlink(name string) (string, error) {
	return "", &PathError{Op: "readlink", Path: name, Err: syscall.ENOSYS}
}
