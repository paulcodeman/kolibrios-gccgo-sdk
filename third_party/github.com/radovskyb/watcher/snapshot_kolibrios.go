//go:build kolibrios

package watcher

import (
	"hash/crc32"
	"io"
	"os"
	"runtime"
)

// FAT's two-second timestamps can hide same-size edits. Keep the real metadata
// and additionally compare a content fingerprint in each upstream snapshot.
type nativeFileSnapshot struct {
	os.FileInfo
	checksum uint32
	valid    bool
}

// The bootstrap scheduler is cooperative. Yield between upstream io.Copy
// chunks so hashing a large workspace does not stop terminal and LSP traffic.
type nativeSnapshotReader struct{ file *os.File }

func (reader nativeSnapshotReader) Read(buffer []byte) (int, error) {
	n, err := reader.file.Read(buffer)
	runtime.Gosched()
	return n, err
}

func nativeSnapshot(path string, info os.FileInfo) os.FileInfo {
	if info.IsDir() {
		return info
	}
	snapshot := &nativeFileSnapshot{FileInfo: info}
	file, err := os.Open(path)
	if err != nil {
		return snapshot
	}
	defer file.Close()
	hash := crc32.NewIEEE()
	_, err = io.Copy(hash, nativeSnapshotReader{file})
	snapshot.valid = err == nil
	snapshot.checksum = hash.Sum32()
	return snapshot
}

func nativeChanged(old, current os.FileInfo) bool {
	if old.ModTime() != current.ModTime() || old.Size() != current.Size() {
		return true
	}
	before, a := old.(*nativeFileSnapshot)
	after, b := current.(*nativeFileSnapshot)
	return a && b && before.valid && after.valid && before.checksum != after.checksum
}
