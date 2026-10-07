package kos

// SDK kernel extension ABI: docs/KERNEL_FILESYSTEM_EXTENSIONS.md.
// Original syscall 70/80 parameter layouts are preserved. Old kernels and
// filesystems that do not implement operations 14/15 report status 2.
// Operations 11/12 are upstream symbolic-link operations; 13 is volume info.
const FileSystemAlreadyExists FileSystemStatus = 17

func CreateExclusiveFile(path string, data []byte) (uint32, FileSystemStatus) {
	return writeFile(path, data, 0, 14)
}

func CreateExclusiveDirectory(path string) FileSystemStatus {
	return fileSystemPathOnly(path, 15)
}
