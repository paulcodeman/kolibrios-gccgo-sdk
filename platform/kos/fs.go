package kos

type StringEncoding uint32
type FileSystemStatus uint32
type FileAttributes uint32

const (
	EncodingDefault StringEncoding = 0
	EncodingCP866   StringEncoding = 1
	EncodingUTF16LE StringEncoding = 2
	EncodingUTF8    StringEncoding = 3
)

const (
	FileSystemOK              FileSystemStatus = 0
	FileSystemUnsupported     FileSystemStatus = 2
	FileSystemNotFound        FileSystemStatus = 5
	FileSystemEOF             FileSystemStatus = 6
	FileSystemBadPointer      FileSystemStatus = 7
	FileSystemDiskFull        FileSystemStatus = 8
	FileSystemInternalError   FileSystemStatus = 9
	FileSystemAccessDenied    FileSystemStatus = 10
	FileSystemDeviceError     FileSystemStatus = 11
	FileSystemNeedsMoreMemory FileSystemStatus = 12
)

const (
	FileAttributeReadOnly  FileAttributes = 1 << 0
	FileAttributeHidden    FileAttributes = 1 << 1
	FileAttributeSystem    FileAttributes = 1 << 2
	FileAttributeLabel     FileAttributes = 1 << 3
	FileAttributeDirectory FileAttributes = 1 << 4
	FileAttributeArchive   FileAttributes = 1 << 5
)

const (
	fileSystemReadFile     = 0
	fileSystemReadFolder   = 1
	fileSystemCreateFile   = 2
	fileSystemWriteFile    = 3
	fileSystemSetEnd       = 4
	fileSystemGetInfo      = 5
	fileSystemSetInfo      = 6
	fileSystemStartApp     = 7
	fileSystemDelete       = 8
	fileSystemCreateFolder = 9
	fileSystemRenameMove   = 10
)

type FileSystemRequest [32]byte

type EncodedFileSystemRequest struct {
	Subfunction       uint32
	Offset            uint32
	OffsetHighOrFlags uint32
	Size              uint32
	Data              uint32
	Encoding          StringEncoding
	Path              uint32
}

type FileTime struct {
	Second byte
	Minute byte
	Hour   byte
}

type FileDate struct {
	Day   byte
	Month byte
	Year  uint16
}

type FileInfo struct {
	Attributes   FileAttributes
	Encoding     StringEncoding
	CreatedTime  FileTime
	CreatedDate  FileDate
	AccessTime   FileTime
	AccessDate   FileDate
	ModifiedTime FileTime
	ModifiedDate FileDate
	Size         uint64
}

type FolderEntry struct {
	Info FileInfo
	Name string
}

type FolderReadResult struct {
	Version uint32
	Count   uint32
	Total   uint32
	Entries []FolderEntry
}

func ReadFile(path string, buffer []byte, offset uint64) (read uint32, status FileSystemStatus) {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return 0, FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction:       fileSystemReadFile,
		Offset:            uint32(offset),
		OffsetHighOrFlags: uint32(offset >> 32),
		Size:              uint32(len(buffer)),
		Data:              byteSliceAddress(buffer),
		Encoding:          EncodingUTF8,
		Path:              pathAddr,
	}

	var secondary uint32
	status = FileSystemStatus(FileSystemEncoded(&request, &secondary))
	freeCString(pathPtr)
	return secondary, status
}

func ReadAllFile(path string) (data []byte, status FileSystemStatus) {
	info, status := GetFileInfo(path)
	if status != FileSystemOK {
		return nil, status
	}

	if info.Size == 0 {
		return []byte{}, FileSystemOK
	}

	data = make([]byte, int(info.Size))
	read, status := ReadFile(path, data, 0)
	if status == FileSystemEOF {
		return data[:read], FileSystemEOF
	}
	if status != FileSystemOK {
		return nil, status
	}

	return data[:read], FileSystemOK
}

func ReadFolder(path string, start uint32, count uint32) (result FolderReadResult, status FileSystemStatus) {
	const folderHeaderSize = 32

	entrySize := folderEntrySize(EncodingUTF8)
	buffer := make([]byte, folderHeaderSize+int(count)*entrySize)
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FolderReadResult{}, FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction:       fileSystemReadFolder,
		Offset:            start,
		OffsetHighOrFlags: uint32(EncodingUTF8),
		Size:              count,
		Data:              byteSliceAddress(buffer),
		Encoding:          EncodingUTF8,
		Path:              pathAddr,
	}

	var secondary uint32
	status = FileSystemStatus(FileSystemEncoded(&request, &secondary))
	freeCString(pathPtr)

	if status != FileSystemOK && status != FileSystemEOF {
		return FolderReadResult{}, status
	}

	result = FolderReadResult{
		Version: littleEndianUint32(buffer, 0),
		Count:   littleEndianUint32(buffer, 4),
		Total:   littleEndianUint32(buffer, 8),
		Entries: make([]FolderEntry, 0, secondary),
	}

	for index := uint32(0); index < secondary; index++ {
		offset := folderHeaderSize + int(index)*entrySize
		block := buffer[offset : offset+entrySize]
		result.Entries = append(result.Entries, FolderEntry{
			Info: parseFileInfoBlock(block),
			Name: zeroTerminatedString(block[40 : 40+fileNameFieldSize(EncodingUTF8)]),
		})
	}

	return result, status
}

func ReadDirectory(path string, start uint32, count uint32) (FolderReadResult, FileSystemStatus) {
	return ReadFolder(path, start, count)
}

func CurrentFolder() string {
	return CurrentFolderWithEncoding(EncodingUTF8)
}

func CurrentFolderWithEncoding(encoding StringEncoding) string {
	var stack [256]byte

	size := GetCurrentFolderRaw(&stack[0], uint32(len(stack)), encoding)
	if size <= 0 {
		return ""
	}
	if size <= len(stack) {
		return zeroTerminatedString(stack[:size])
	}

	buffer := make([]byte, size)
	size = GetCurrentFolderRaw(&buffer[0], uint32(len(buffer)), encoding)
	if size <= 0 {
		return ""
	}
	if size > len(buffer) {
		size = len(buffer)
	}

	return zeroTerminatedString(buffer[:size])
}

func CreateOrRewriteFile(path string, data []byte) (written uint32, status FileSystemStatus) {
	return writeFile(path, data, 0, fileSystemCreateFile)
}

func WriteFile(path string, data []byte, offset uint64) (written uint32, status FileSystemStatus) {
	return writeFile(path, data, offset, fileSystemWriteFile)
}

func SetFileSize(path string, size uint64) FileSystemStatus {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction:       fileSystemSetEnd,
		Offset:            uint32(size),
		OffsetHighOrFlags: uint32(size >> 32),
		Encoding:          EncodingUTF8,
		Path:              pathAddr,
	}

	status := FileSystemStatus(FileSystemEncoded(&request, nil))
	freeCString(pathPtr)
	return status
}

func GetFileInfo(path string) (info FileInfo, status FileSystemStatus) {
	var buffer [40]byte

	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FileInfo{}, FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction: fileSystemGetInfo,
		Data:        pointerValue(&buffer[0]),
		Encoding:    EncodingUTF8,
		Path:        pathAddr,
	}

	status = FileSystemStatus(FileSystemEncoded(&request, nil))
	freeCString(pathPtr)
	if status != FileSystemOK {
		return FileInfo{}, status
	}

	return parseFileInfoBlock(buffer[:]), FileSystemOK
}

func GetPathInfo(path string) (FileInfo, FileSystemStatus) {
	return GetFileInfo(path)
}

func SetFileInfo(path string, info FileInfo) FileSystemStatus {
	var buffer [32]byte

	buffer[0] = byte(info.Attributes)
	buffer[1] = byte(info.Attributes >> 8)
	buffer[2] = byte(info.Attributes >> 16)
	buffer[3] = byte(info.Attributes >> 24)
	buffer[4] = byte(info.Encoding)
	encodeFileTime(buffer[:], 8, info.CreatedTime)
	encodeFileDate(buffer[:], 12, info.CreatedDate)
	encodeFileTime(buffer[:], 16, info.AccessTime)
	encodeFileDate(buffer[:], 20, info.AccessDate)
	encodeFileTime(buffer[:], 24, info.ModifiedTime)
	encodeFileDate(buffer[:], 28, info.ModifiedDate)

	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction: fileSystemSetInfo,
		Data:        pointerValue(&buffer[0]),
		Encoding:    EncodingUTF8,
		Path:        pathAddr,
	}

	status := FileSystemStatus(FileSystemEncoded(&request, nil))
	freeCString(pathPtr)
	return status
}

func SetPathInfo(path string, info FileInfo) FileSystemStatus {
	return SetFileInfo(path, info)
}

func StartApp(path string, params string, debugged bool) int {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return -int(FileSystemNeedsMoreMemory)
	}

	var paramsPtr *byte
	paramsAddr := uint32(0)
	if len(params) > 0 {
		paramsPtr, paramsAddr = stringAddress(params)
		if paramsPtr == nil {
			freeCString(pathPtr)
			return -int(FileSystemNeedsMoreMemory)
		}
	}

	flags := uint32(0)
	if debugged {
		flags = 1
	}

	request := EncodedFileSystemRequest{
		Subfunction:       fileSystemStartApp,
		Offset:            flags,
		OffsetHighOrFlags: paramsAddr,
		Encoding:          EncodingUTF8,
		Path:              pathAddr,
	}

	result := FileSystemEncoded(&request, nil)
	if paramsPtr != nil {
		freeCString(paramsPtr)
	}
	freeCString(pathPtr)
	return result
}

func StartApplication(path string, params string, debugged bool) (pid int, status FileSystemStatus) {
	pid = StartApp(path, params, debugged)
	if pid < 0 {
		return 0, FileSystemStatus(-pid)
	}

	return pid, FileSystemOK
}

func DeletePath(path string) FileSystemStatus {
	return fileSystemPathOnly(path, fileSystemDelete)
}

func CreateFolder(path string) FileSystemStatus {
	return fileSystemPathOnly(path, fileSystemCreateFolder)
}

func CreateDirectory(path string) FileSystemStatus {
	return CreateFolder(path)
}

func RenameOrMove(path string, newPath string) FileSystemStatus {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FileSystemNeedsMoreMemory
	}

	newPathPtr, newPathAddr := stringAddress(newPath)
	if newPathPtr == nil {
		freeCString(pathPtr)
		return FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction: fileSystemRenameMove,
		Data:        newPathAddr,
		Encoding:    EncodingUTF8,
		Path:        pathAddr,
	}

	status := FileSystemStatus(FileSystemEncoded(&request, nil))
	freeCString(newPathPtr)
	freeCString(pathPtr)
	return status
}

func RenamePath(path string, newPath string) FileSystemStatus {
	normalizedPath, normalizedNewPath, ok := normalizeRenamePaths(path, newPath)
	if !ok {
		return FileSystemUnsupported
	}

	return RenameOrMove(normalizedPath, normalizedNewPath)
}

func writeFile(path string, data []byte, offset uint64, subfunction uint32) (written uint32, status FileSystemStatus) {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return 0, FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction:       subfunction,
		Offset:            uint32(offset),
		OffsetHighOrFlags: uint32(offset >> 32),
		Size:              uint32(len(data)),
		Data:              byteSliceAddress(data),
		Encoding:          EncodingUTF8,
		Path:              pathAddr,
	}

	var secondary uint32
	status = FileSystemStatus(FileSystemEncoded(&request, &secondary))
	freeCString(pathPtr)
	return secondary, status
}

func fileSystemPathOnly(path string, subfunction uint32) FileSystemStatus {
	pathPtr, pathAddr := stringAddress(path)
	if pathPtr == nil {
		return FileSystemNeedsMoreMemory
	}

	request := EncodedFileSystemRequest{
		Subfunction: subfunction,
		Encoding:    EncodingUTF8,
		Path:        pathAddr,
	}

	status := FileSystemStatus(FileSystemEncoded(&request, nil))
	freeCString(pathPtr)
	return status
}

func folderEntrySize(encoding StringEncoding) int {
	if encoding == EncodingCP866 {
		return 304
	}

	return 560
}

func fileNameFieldSize(encoding StringEncoding) int {
	if encoding == EncodingCP866 {
		return 264
	}

	return 520
}

func parseFileInfoBlock(buffer []byte) FileInfo {
	return FileInfo{
		Attributes:   FileAttributes(littleEndianUint32(buffer, 0)),
		Encoding:     StringEncoding(littleEndianUint32(buffer, 4)),
		CreatedTime:  parseFileTime(buffer, 8),
		CreatedDate:  parseFileDate(buffer, 12),
		AccessTime:   parseFileTime(buffer, 16),
		AccessDate:   parseFileDate(buffer, 20),
		ModifiedTime: parseFileTime(buffer, 24),
		ModifiedDate: parseFileDate(buffer, 28),
		Size:         littleEndianUint64(buffer, 32),
	}
}

func parseFileTime(buffer []byte, offset int) FileTime {
	return FileTime{
		Second: buffer[offset],
		Minute: buffer[offset+1],
		Hour:   buffer[offset+2],
	}
}

func parseFileDate(buffer []byte, offset int) FileDate {
	return FileDate{
		Day:   buffer[offset],
		Month: buffer[offset+1],
		Year:  littleEndianUint16(buffer, offset+2),
	}
}

func encodeFileTime(buffer []byte, offset int, value FileTime) {
	buffer[offset] = value.Second
	buffer[offset+1] = value.Minute
	buffer[offset+2] = value.Hour
	buffer[offset+3] = 0
}

func encodeFileDate(buffer []byte, offset int, value FileDate) {
	buffer[offset] = value.Day
	buffer[offset+1] = value.Month
	buffer[offset+2] = byte(value.Year)
	buffer[offset+3] = byte(value.Year >> 8)
}

func normalizeRenamePaths(path string, newPath string) (normalizedPath string, normalizedNewPath string, ok bool) {
	absolutePath, ok := absoluteFSPath(path)
	if !ok {
		return "", "", false
	}

	absoluteNewPath, ok := absoluteFSPath(newPath)
	if !ok {
		return "", "", false
	}

	pathRoot, ok := volumeRootPath(absolutePath)
	if !ok {
		return "", "", false
	}

	newPathRoot, ok := volumeRootPath(absoluteNewPath)
	if !ok {
		return "", "", false
	}

	if !equalFoldASCII(pathRoot, newPathRoot) {
		return "", "", false
	}

	normalizedNewPath = absoluteNewPath[len(newPathRoot):]
	if normalizedNewPath == "" {
		normalizedNewPath = "/"
	}

	return absolutePath, normalizedNewPath, true
}

func absoluteFSPath(name string) (string, bool) {
	if name == "" {
		return "", false
	}

	if name[0] == '/' {
		return cleanSlashPath(name), true
	}

	currentFolder := CurrentFolder()
	if currentFolder == "" {
		return "", false
	}

	return cleanSlashPath(currentFolder + "/" + name), true
}

func cleanSlashPath(name string) string {
	if name == "" {
		return "."
	}

	rooted := name[0] == '/'
	parts := make([]string, 0, 8)
	index := 0

	for index < len(name) {
		for index < len(name) && name[index] == '/' {
			index++
		}
		if index >= len(name) {
			break
		}

		next := index
		for next < len(name) && name[next] != '/' {
			next++
		}

		part := name[index:next]
		switch part {
		case "", ".":
		case "..":
			if rooted {
				if len(parts) > 0 {
					parts = parts[:len(parts)-1]
				}
			} else if len(parts) > 0 && parts[len(parts)-1] != ".." {
				parts = parts[:len(parts)-1]
			} else {
				parts = append(parts, part)
			}
		default:
			parts = append(parts, part)
		}

		index = next
	}

	if rooted {
		if len(parts) == 0 {
			return "/"
		}

		cleaned := "/" + parts[0]
		for index = 1; index < len(parts); index++ {
			cleaned += "/" + parts[index]
		}
		return cleaned
	}

	if len(parts) == 0 {
		return "."
	}

	cleaned := parts[0]
	for index = 1; index < len(parts); index++ {
		cleaned += "/" + parts[index]
	}
	return cleaned
}

func volumeRootPath(name string) (string, bool) {
	if len(name) < 2 || name[0] != '/' {
		return "", false
	}

	firstEnd := indexSlash(name, 1)
	if firstEnd < 0 {
		return "", false
	}

	firstSegment := name[1:firstEnd]
	if firstSegment == "" {
		return "", false
	}

	if equalFoldASCII(firstSegment, "sys") {
		return "/" + firstSegment, true
	}

	secondStart := firstEnd + 1
	if secondStart >= len(name) {
		return "/" + firstSegment, true
	}

	secondEnd := indexSlash(name, secondStart)
	if secondEnd < 0 {
		return name, true
	}

	return name[:secondEnd], true
}

func indexSlash(name string, start int) int {
	for index := start; index < len(name); index++ {
		if name[index] == '/' {
			return index
		}
	}

	return -1
}

func equalFoldASCII(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := 0; index < len(left); index++ {
		if foldASCII(left[index]) != foldASCII(right[index]) {
			return false
		}
	}

	return true
}

func foldASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}

	return value
}
