package syscall

import (
	"internal/bytealg"
	"kos"
)

func Chdir(path string) error {
	if bytealg.IndexByteString(path, 0) != -1 {
		return EINVAL
	}
	info, status := kos.GetPathInfo(path)
	if status != kos.FileSystemOK || info.Attributes&kos.FileAttributeDirectory == 0 {
		_, readStatus := kos.ReadDirectory(path, 0, 1)
		if readStatus != kos.FileSystemOK && readStatus != kos.FileSystemEOF {
			if status == kos.FileSystemOK {
				return ENOTDIR
			}
			return filesystemErrno(status)
		}
	}
	return filesystemErrno(kos.ChangeCurrentFolder(path))
}
