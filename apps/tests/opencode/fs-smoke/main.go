// Exercise upstream file algorithms against the temporary QEMU FAT disk.
package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"kos"
)

func main() {
	kos.DebugString("OPENCODE_FS_START")
	console, ok := kos.OpenConsole("OpenCode filesystem port test")
	if !ok {
		panic("filesystem test console unavailable")
	}
	defer console.Close()
	const root = "/hd0/1/FSCASE"
	if err := os.RemoveAll(root); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(root+"/tree/child", 0700); err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	checkSyscallFS(root)
	const name = root + "/данные-資料.txt"
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	if _, err = f.WriteString("abcdef"); err != nil {
		panic(err)
	}
	if _, err = f.Seek(2, io.SeekStart); err != nil {
		panic(err)
	}
	var writer io.WriterAt = f
	if n, err := writer.WriteAt([]byte("X"), 4); n != 1 || err != nil {
		panic("positional write failed")
	}
	if offset, _ := f.Seek(0, io.SeekCurrent); offset != 2 {
		panic("WriteAt changed the sequential offset")
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "abcdXf" {
		panic("WriteAt modified the wrong bytes")
	}
	if _, err = writer.WriteAt([]byte("bad"), -1); err == nil {
		panic("negative positional write accepted")
	}
	if err = f.Truncate(-1); err == nil {
		panic("negative truncate accepted")
	}
	if err = f.Truncate(3); err != nil {
		panic(err)
	}
	if err = f.Truncate(8); err != nil {
		panic(err)
	}
	data, err = os.ReadFile(name)
	if err != nil || !bytes.Equal(data, []byte{'a', 'b', 'c', 0, 0, 0, 0, 0}) {
		panic("truncate shrink or zero-filled expansion failed")
	}
	if offset, _ := f.Seek(0, io.SeekCurrent); offset != 2 {
		panic("Truncate changed the sequential offset")
	}
	const largeSize = int64(16*1024*1024 + 4096)
	if err = f.Truncate(largeSize); err != nil {
		panic(err)
	}
	var zeros [32]byte
	for _, off := range []int64{8, 16*1024*1024 - 16, largeSize - 32} {
		var got [32]byte
		if n, err := f.ReadAt(got[:], off); n != len(got) || err != nil || got != zeros {
			panic("large expansion did not expose zero-filled storage")
		}
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
	if _, err = f.WriteAt([]byte("closed"), 0); !errors.Is(err, os.ErrClosed) {
		panic("closed positional write lost ErrClosed")
	}
	if err = os.Truncate(name, 3); err != nil {
		panic(err)
	}
	appendFile, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		panic(err)
	}
	if _, err = appendFile.WriteAt([]byte("append"), 0); err == nil {
		panic("WriteAt accepted an append handle")
	}
	appendFile.Close()
	readOnly, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	if err = readOnly.Truncate(0); err == nil {
		panic("Truncate accepted a read-only handle")
	}
	readOnly.Close()
	kos.DebugString("FILE_POSITIONAL_OK")
	atime := time.Date(2000, time.January, 3, 4, 5, 6, 0, time.UTC)
	mtime := time.Date(2001, time.September, 9, 12, 30, 10, 0, time.UTC)
	if err = os.Chtimes(name, atime, mtime); err != nil {
		panic(err)
	}
	info, err := os.Stat(name)
	if err != nil || info.Size() != 3 || info.ModTime().Unix() != mtime.Unix() {
		panic("Chtimes modified size or returned the wrong timestamp")
	}
	if err = os.Chmod(name, 0444); err != nil {
		panic(err)
	}
	info, err = os.Stat(name)
	if err != nil || info.Mode().Perm() != 0444 || info.ModTime().Unix() != mtime.Unix() {
		panic("Chmod lost the read-only bit or changed file timestamps")
	}
	if err = os.Chmod(name, 0600); err != nil {
		panic(err)
	}
	info, err = os.Stat(name)
	if err != nil || info.Mode().Perm() != 0666 {
		panic("Chmod did not clear the native read-only bit")
	}
	if err = os.Chown(name, 1, 2); !errors.Is(err, syscall.ENOSYS) {
		panic("native ownership unsupported error lost")
	}
	if err = os.Symlink(name, root+"/link"); !errors.Is(err, syscall.ENOSYS) {
		panic("native link unsupported error lost")
	}
	kos.DebugString("FILE_METADATA_OK")
	for _, path := range []string{root + "/z", root + "/a", root + "/tree/child/data"} {
		if err = os.WriteFile(path, []byte("tree"), 0600); err != nil {
			panic(err)
		}
	}
	kos.DebugString("BEFORE_READDIR")
	entries, err := os.ReadDir(root)
	kos.DebugString("AFTER_READDIR")
	if err != nil || len(entries) != 4 {
		panic("ReadDir did not return all entries")
	}
	for i, entry := range entries {
		if i > 0 && entries[i-1].Name() >= entry.Name() {
			panic("ReadDir did not sort names")
		}
		if _, err := entry.Info(); err != nil {
			panic(err)
		}
	}
	if entries[len(entries)-1].Name() != "данные-資料.txt" {
		panic("ReadDir lost Unicode filename characters")
	}
	directory, err := os.Open(root)
	if err != nil {
		panic(err)
	}
	first, err := directory.ReadDir(1)
	if err != nil || len(first) != 1 {
		panic("bounded ReadDir failed")
	}
	next, err := directory.Readdirnames(1)
	if err != nil || len(next) != 1 || next[0] == first[0].Name() {
		panic("directory APIs lost the shared cursor")
	}
	remaining, err := directory.ReadDir(-1)
	if err != nil || len(remaining) != 2 {
		panic("ReadDir did not resume the directory cursor")
	}
	if last, err := directory.ReadDir(1); len(last) != 0 || err != io.EOF {
		panic("bounded ReadDir lost EOF")
	}
	directory.Close()
	var walked []string
	if err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		walked = append(walked, path)
		return nil
	}); err != nil || len(walked) != 7 {
		panic("Walk failed to visit the directory tree")
	}
	var skipped []string
	if err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		skipped = append(skipped, path)
		// The native FAT backend returns short DOS names in upper case.
		if strings.EqualFold(entry.Name(), "tree") {
			return filepath.SkipDir
		}
		return nil
	}); err != nil || len(skipped) != 5 {
		kos.DebugString("WALKDIR_COUNT=" + strconv.Itoa(len(skipped)))
		if err != nil {
			kos.DebugString("WALKDIR_ERROR=" + err.Error())
		}
		for _, path := range skipped {
			kos.DebugString("WALKDIR_PATH=" + path)
		}
		panic("WalkDir did not skip the subtree")
	}
	visits := 0
	if err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		visits++
		if visits == 2 {
			return filepath.SkipAll
		}
		return err
	}); err != nil || visits != 2 {
		panic("WalkDir did not stop at SkipAll")
	}
	missingCalled := false
	if err = filepath.WalkDir(root+"/absent", func(path string, entry os.DirEntry, err error) error {
		missingCalled = entry == nil && os.IsNotExist(err)
		return nil
	}); err != nil || !missingCalled {
		panic("WalkDir did not report the missing root to the callback")
	}
	kos.DebugString("WALK_OK")
	if err = os.RemoveAll(root + "/."); !errors.Is(err, syscall.EINVAL) {
		panic("RemoveAll accepted a final dot component")
	}
	if err = os.RemoveAll(root); err != nil {
		panic(err)
	}
	if _, err = os.Stat(root); !os.IsNotExist(err) {
		panic("RemoveAll left the directory tree")
	}
	if err = os.RemoveAll(root); err != nil {
		panic("RemoveAll failed on an absent path")
	}
	kos.DebugString("DIRECTORY_OKOPENCODE_FS_PASS")
	console.WriteString("OPENCODE_FS_PASS\n")
}

func checkSyscallFS(root string) {
	name := root + "/syscall-file"
	if err := os.WriteFile(name, []byte("native rename"), 0600); err != nil {
		panic(err)
	}
	first, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	second, err := os.Stat(root + "/tree/../syscall-file")
	if err != nil || !os.SameFile(first, second) {
		panic("SameFile canonical path")
	}
	other, err := os.Stat(root + "/tree")
	if err != nil || os.SameFile(first, other) || os.SameFile(first, nil) {
		panic("SameFile distinct files")
	}
	if !errors.Is(syscall.Rmdir(name), syscall.ENOTDIR) {
		panic("Rmdir accepted file")
	}
	if !errors.Is(syscall.Unlink(name+"\x00suffix"), syscall.EINVAL) {
		panic("Unlink accepted NUL")
	}
	if !errors.Is(syscall.Rmdir(root+"/tree\x00suffix"), syscall.EINVAL) {
		panic("Rmdir accepted NUL")
	}
	if !errors.Is(syscall.Rename(name, root+"/bad\x00suffix"), syscall.EINVAL) ||
		!errors.Is(syscall.Rename(name+"\x00suffix", root+"/bad"), syscall.EINVAL) {
		panic("Rename accepted NUL")
	}
	if !errors.Is(syscall.Unlink(root+"/tree"), syscall.EISDIR) {
		panic("Unlink accepted directory")
	}
	if !errors.Is(syscall.Rmdir(root+"/tree"), syscall.ENOTEMPTY) {
		panic("nonempty Rmdir errno")
	}
	if err := syscall.Rename(name, root+"/renamed"); err != nil {
		panic(err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		panic("Rename retained old path")
	}
	data, err := os.ReadFile(root + "/renamed")
	if err != nil || string(data) != "native rename" {
		panic("Rename lost contents")
	}
	if err := syscall.Unlink(root + "/renamed"); err != nil {
		panic(err)
	}
	if !errors.Is(syscall.Unlink(root+"/renamed"), syscall.ENOENT) {
		panic("missing Unlink errno")
	}
	if err := os.Mkdir(root+"/empty", 0700); err != nil {
		panic(err)
	}
	if err := syscall.Rmdir(root + "/empty"); err != nil {
		panic(err)
	}
	if !errors.Is(syscall.SetNonblock(-1, true), syscall.EBADF) {
		panic("negative fd nonblock")
	}
	if !errors.Is(syscall.SetNonblock(1, true), syscall.ENOTSUP) {
		panic("unsupported nonblock reported success")
	}
	kos.DebugString("SYSCALL_FILESYSTEM_OK")
}
