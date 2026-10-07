package os

import (
	"io"
	"io/fs"
	"kos"
	"path"
	"syscall"
	"time"
)

type FileMode = fs.FileMode

const (
	PathSeparator     = '/'
	PathListSeparator = ':'
)

const (
	O_RDONLY int = 0
	O_WRONLY int = 1
	O_RDWR   int = 2

	O_CREATE int = 0x40
	O_EXCL   int = 0x80
	O_TRUNC  int = 0x200
	O_APPEND int = 0x400
	O_SYNC   int = syscall.O_SYNC
)

type osError struct {
	text string
}

func (err *osError) Error() string {
	return err.text
}

var Stdin = &File{
	name:     "stdin",
	fd:       int(kos.StdinFD),
	readable: true,
	writable: false,
	fdBacked: true,
}

var Stdout = &File{
	name:     "stdout",
	fd:       int(kos.StdoutFD),
	readable: false,
	writable: true,
	fdBacked: true,
}

var Stderr = &File{
	name:     "stderr",
	fd:       int(kos.StderrFD),
	readable: false,
	writable: true,
	fdBacked: true,
}

func init() {
	ensureStandardFiles()
	bootstrapArgs()
}

func ensureStandardFiles() {
	if Stdin == nil || Stdin.name == "" {
		Stdin = &File{
			name:     "stdin",
			fd:       int(kos.StdinFD),
			readable: true,
			writable: false,
			fdBacked: true,
		}
	}
	if Stdout == nil || Stdout.name == "" {
		Stdout = &File{
			name:     "stdout",
			fd:       int(kos.StdoutFD),
			readable: false,
			writable: true,
			fdBacked: true,
		}
	}
	if Stderr == nil || Stderr.name == "" {
		Stderr = &File{
			name:     "stderr",
			fd:       int(kos.StderrFD),
			readable: false,
			writable: true,
			fdBacked: true,
		}
	}
}

func DefaultStdin() *File {
	ensureStandardFiles()
	return Stdin
}

func DefaultStdout() *File {
	ensureStandardFiles()
	return Stdout
}

func DefaultStderr() *File {
	ensureStandardFiles()
	return Stderr
}

type LinkError struct {
	Op  string
	Old string
	New string
	Err error
}

func (err *LinkError) Error() string {
	if err == nil {
		return ""
	}
	if err.Err == nil {
		return err.Op + " " + err.Old + " " + err.New
	}

	return err.Op + " " + err.Old + " " + err.New + ": " + err.Err.Error()
}

func (err *LinkError) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Err
}

func (err *LinkError) As(target interface{}) bool {
	if err == nil {
		return false
	}

	switch typed := target.(type) {
	case **LinkError:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}

	return false
}

type statusError struct {
	status kos.FileSystemStatus
	text   string
}

func (err *statusError) Error() string {
	return err.text
}

type FileInfo = fs.FileInfo

type fileInfo struct {
	name string
	path string
	raw  kos.FileInfo
}

func (info fileInfo) Name() string {
	return info.name
}

func (info fileInfo) Size() int64 {
	return int64(info.raw.Size)
}

func (info fileInfo) Mode() FileMode {
	mode := FileMode(0666)
	if info.raw.Attributes&kos.FileAttributeReadOnly != 0 {
		mode = 0444
	}
	if info.raw.Attributes&kos.FileAttributeDirectory != 0 || isVolumeRootPath(info.path) {
		mode |= ModeDir | 0111
	}

	return mode
}

func (info fileInfo) ModTime() time.Time {
	return fileStampTime(info.raw.ModifiedDate, info.raw.ModifiedTime)
}

func (info fileInfo) IsDir() bool {
	return info.Mode().IsDir()
}

func (info fileInfo) Sys() interface{} {
	return info.raw
}

type File struct {
	displayName    string
	name           string
	fd             int
	offset         uint64
	sharedPosition *nativeFilePosition
	readable       bool
	writable       bool
	append         bool
	closed         bool
	fdBacked       bool
	pending        []byte
	pipe           *pipeState
	localStream    *nativeLocalStream
}

const activeConsoleReadBufferSize = 256

type pipeState struct {
	pending uint64
	readers uint32
	writers uint32
}

var Args = []string{""}

func bootstrapArgs() {
	if startup := kos.CurrentProcessStartup(); startup != nil {
		Args = startup.Args
		streams := make(map[uint32]*File)
		for index, name := range startup.StandardFiles {
			if index > 2 {
				panic("unsupported child standard descriptor")
			}
			file := &File{name: name, fd: -1, readable: index == 0, writable: index != 0}
			if descriptor, ok := kos.LocalSocketStartupDescriptor(name); ok {
				if previous := streams[descriptor]; previous != nil {
					var err error
					file, err = previous.cloneNativeLocal()
					if err != nil {
						panic(err)
					}
				} else {
					file = newNativeLocalFile(name, descriptor, index == 0, index != 0)
					streams[descriptor] = file
				}
			}
			if name == "" {
				file.name = "closed standard descriptor"
				file.fdBacked = true
			}
			switch index {
			case 0:
				Stdin = file
			case 1:
				Stdout = file
			case 2:
				Stderr = file
			}
		}
		standardFiles := []*File{Stdin, Stdout, Stderr}
		kos.ChildProcessCloseStreams = func() {
			for _, file := range standardFiles {
				if file != nil && file.localStream != nil {
					_ = file.Close()
				}
			}
		}
		if err := Chdir(startup.Dir); err != nil {
			panic(err)
		}
		return
	}
	Args = loaderArgs(kos.LoaderPath(), kos.LoaderParameters())
}

func loaderArgs(path string, params string) []string {
	args := splitLoaderCommandLine(params)
	if len(args) == 0 {
		return []string{path}
	}

	result := make([]string, len(args)+1)
	result[0] = path
	for index := 0; index < len(args); index++ {
		result[index+1] = args[index]
	}
	return result
}

func splitLoaderCommandLine(value string) []string {
	args := make([]string, 0, 4)
	token := make([]byte, 0, len(value))
	quotedBy := byte(0)
	escaped := false
	tokenStarted := false

	for index := 0; index < len(value); index++ {
		ch := value[index]

		if escaped {
			token = append(token, ch)
			tokenStarted = true
			escaped = false
			continue
		}

		if ch == '\\' {
			tokenStarted = true
			escaped = true
			continue
		}

		if quotedBy != 0 {
			if ch == quotedBy {
				quotedBy = 0
				continue
			}

			token = append(token, ch)
			tokenStarted = true
			continue
		}

		if ch == '"' || ch == '\'' {
			quotedBy = ch
			tokenStarted = true
			continue
		}

		if isLoaderArgSpace(ch) {
			if tokenStarted {
				args = append(args, string(token))
				token = token[:0]
				tokenStarted = false
			}
			continue
		}

		token = append(token, ch)
		tokenStarted = true
	}

	if escaped {
		token = append(token, '\\')
	}
	if tokenStarted {
		args = append(args, string(token))
	}

	return args
}

func isLoaderArgSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\r', '\n':
		return true
	}

	return false
}

func Getwd() (dir string, err error) {
	dir = kos.CurrentFolder()
	if dir == "" {
		return "", &PathError{Op: "getwd", Path: "", Err: ErrInvalid}
	}

	return dir, nil
}

func Getpid() int {
	id, ok := kos.CurrentThreadID()
	if !ok {
		return 0
	}

	return int(id)
}

func Getppid() int {
	return 0
}

func Exit(code int) {
	kos.ChildProcessExit(code)
	nativeProcessExit()
}

func Stat(name string) (FileInfo, error) {
	info, status := kos.GetPathInfo(name)
	if status == kos.FileSystemOK {
		// The native volume-root metadata leaves the directory attribute
		// unset. Successful root lookup still denotes a directory.
		if isVolumeRootPath(name) {
			info.Attributes |= kos.FileAttributeDirectory
		}
		return fileInfo{
			name: baseName(name),
			path: fileInfoPath(name),
			raw:  info,
		}, nil
	}

	cleaned := path.Clean(name)
	if rootInfo, ok := statVolumeRoot(cleaned); ok {
		return rootInfo, nil
	}

	return nil, wrapPathError("stat", name, status)
}

func statVolumeRoot(name string) (FileInfo, bool) {
	if !isVolumeRootPath(name) {
		return nil, false
	}

	_, status := kos.ReadFolder(name, 0, 1)
	if status != kos.FileSystemOK && status != kos.FileSystemEOF {
		return nil, false
	}

	return fileInfo{
		name: baseName(name),
		path: fileInfoPath(name),
		raw: kos.FileInfo{
			Attributes: kos.FileAttributeDirectory,
		},
	}, true
}

func isVolumeRootPath(name string) bool {
	name = path.Clean(name)
	if name == "/" {
		return true
	}
	if len(name) < 2 || name[0] != '/' {
		return false
	}
	if equalFoldASCII(name[1:], "sys") {
		return true
	}

	firstEnd := indexSlash(name, 1)
	if firstEnd < 0 {
		return false
	}
	first := name[1:firstEnd]
	secondStart := firstEnd + 1
	if secondStart >= len(name) {
		return false
	}
	if indexSlash(name, secondStart) >= 0 {
		return false
	}

	second := name[secondStart:]
	if !isASCIIUnsignedDecimal(second) {
		return false
	}
	if equalFoldASCII(first, "rd") || equalFoldASCII(first, "fd") {
		return true
	}
	if hasASCIIPrefixFold(first, "tmp") {
		return len(first) > 3 && isASCIIUnsignedDecimal(first[3:])
	}
	if hasASCIIPrefixFold(first, "hd") || hasASCIIPrefixFold(first, "cd") {
		return len(first) > 2 && isASCIIUnsignedDecimal(first[2:])
	}
	return false
}

func indexSlash(value string, start int) int {
	for index := start; index < len(value); index++ {
		if value[index] == '/' {
			return index
		}
	}
	return -1
}

func hasASCIIPrefixFold(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	return equalFoldASCII(value[:len(prefix)], prefix)
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

func isASCIIUnsignedDecimal(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func TempDir() string {
	if value, ok := LookupEnv("TMPDIR"); ok && value != "" {
		return value
	}
	if value, ok := LookupEnv("TEMP"); ok && value != "" {
		return value
	}
	return "/tmp0/1"
}

func ReadFile(name string) ([]byte, error) {
	data, status := kos.ReadAllFile(name)
	if status == kos.FileSystemOK || status == kos.FileSystemEOF {
		return data, nil
	}

	return nil, wrapPathError("read", name, status)
}

func WriteFile(name string, data []byte, perm FileMode) error {
	written, status := kos.CreateOrRewriteFile(name, data)
	if status != kos.FileSystemOK {
		return wrapPathError("write", name, status)
	}
	if int(written) != len(data) {
		return &PathError{Op: "write", Path: name, Err: io.ErrShortWrite}
	}

	return nil
}

func Mkdir(name string, perm FileMode) error {
	status := kos.CreateExclusiveDirectory(name)
	if status == kos.FileSystemUnsupported {
		return &PathError{Op: "mkdir", Path: name, Err: ErrExclusiveCreateUnsupported}
	}
	if status != kos.FileSystemOK {
		return wrapPathError("mkdir", name, status)
	}

	return nil
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits are ignored.

func Remove(name string) error {
	status := kos.DeletePath(name)
	if status != kos.FileSystemOK {
		return wrapPathError("remove", name, status)
	}

	return nil
}

func Rename(oldpath string, newpath string) error {
	status := kos.RenamePath(oldpath, newpath)
	if status != kos.FileSystemOK {
		return &LinkError{
			Op:  "rename",
			Old: oldpath,
			New: newpath,
			Err: statusToError(status),
		}
	}

	return nil
}

func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}

func Create(name string) (*File, error) {
	return OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0)
}

func Pipe() (reader *File, writer *File, err error) {
	first, second, code := kos.CreateLocalSocketPair()
	if code != 0 {
		return nil, nil, NewSyscallError("pipe", syscall.ENOMEM)
	}
	return newNativeLocalFile("|0", first, true, false), newNativeLocalFile("|1", second, false, true), nil
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	if name == DevNull {
		return &File{name: DevNull, fd: -1, readable: flag&3 != O_WRONLY, writable: flag&3 != O_RDONLY}, nil
	}
	originalName := name
	if name != "" && !path.IsAbs(name) {
		wd, err := Getwd()
		if err != nil {
			return nil, err
		}
		name = path.Join(wd, name)
	}

	accessMode := flag & 3
	readable := accessMode == O_RDONLY || accessMode == O_RDWR
	writable := accessMode == O_WRONLY || accessMode == O_RDWR

	if flag&O_TRUNC != 0 && !writable {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}
	if flag&O_APPEND != 0 && !writable {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}

	exclusive := flag&(O_CREATE|O_EXCL) == O_CREATE|O_EXCL
	if exclusive {
		_, status := kos.CreateExclusiveFile(name, nil)
		if status == kos.FileSystemUnsupported {
			return nil, &PathError{Op: "open", Path: originalName, Err: ErrExclusiveCreateUnsupported}
		}
		if status != kos.FileSystemOK {
			return nil, wrapPathError("open", originalName, status)
		}
	} else if flag&O_CREATE != 0 {
		_, status := kos.GetPathInfo(name)
		if status == kos.FileSystemNotFound {
			_, status = kos.CreateOrRewriteFile(name, nil)
			if status != kos.FileSystemOK {
				return nil, wrapPathError("open", name, status)
			}
		} else if status != kos.FileSystemOK {
			return nil, wrapPathError("open", name, status)
		}
	}

	if flag&O_TRUNC != 0 && !exclusive {
		_, status := kos.CreateOrRewriteFile(name, nil)
		if status != kos.FileSystemOK {
			return nil, wrapPathError("open", name, status)
		}
	}

	info, status := kos.GetPathInfo(name)
	if status != kos.FileSystemOK {
		return nil, wrapPathError("open", name, status)
	}

	file := &File{
		name:        name,
		displayName: originalName,
		readable:    readable,
		writable:    writable,
		append:      flag&O_APPEND != 0,
	}
	if file.append {
		file.offset = info.Size
	}

	return file, nil
}

func (file *File) Name() string {
	if file == nil {
		return ""
	}

	if file.displayName != "" {
		return file.displayName
	}
	return file.name
}

func (file *File) Stat() (FileInfo, error) {
	if file == nil {
		return nil, &PathError{Op: "stat", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return nil, &PathError{Op: "stat", Path: file.name, Err: ErrClosed}
	}
	if file.fdBacked {
		return nil, &PathError{Op: "stat", Path: file.name, Err: ErrInvalid}
	}

	return Stat(file.name)
}

func (file *File) Readdir(n int) ([]FileInfo, error) {
	if file == nil {
		return nil, &PathError{Op: "readdir", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return nil, &PathError{Op: "readdir", Path: file.name, Err: ErrClosed}
	}
	if file.fdBacked {
		return nil, &PathError{Op: "readdir", Path: file.name, Err: ErrInvalid}
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &PathError{Op: "readdir", Path: file.name, Err: ErrInvalid}
	}

	readAll := n <= 0
	remaining := n
	result := make([]FileInfo, 0, 16)

	for {
		batchSize := uint32(64)
		if !readAll && remaining < int(batchSize) {
			batchSize = uint32(remaining)
		}
		if batchSize == 0 {
			break
		}

		read, status := kos.ReadFolder(file.name, uint32(file.offset), batchSize)
		if status != kos.FileSystemOK && status != kos.FileSystemEOF {
			return result, wrapPathError("readdir", file.name, status)
		}

		before := len(result)
		for index := 0; index < len(read.Entries); index++ {
			entry := read.Entries[index]
			// Match upstream os/dir_gccgo.go: special entries consume the
			// native directory cursor but not the caller's requested count.
			if entry.Name == "." || entry.Name == ".." {
				continue
			}
			result = append(result, fileInfo{
				name: entry.Name,
				path: fileInfoPath(path.Join(file.name, entry.Name)),
				raw:  entry.Info,
			})
		}
		file.offset += uint64(len(read.Entries))

		if !readAll {
			remaining -= len(result) - before
			if remaining <= 0 {
				return result, nil
			}
		}

		if status == kos.FileSystemEOF || len(read.Entries) == 0 {
			if !readAll && len(result) == 0 {
				return nil, io.EOF
			}
			return result, nil
		}
	}

	if !readAll && len(result) == 0 {
		return nil, io.EOF
	}
	return result, nil
}

func (file *File) Close() error {
	if file == nil {
		return &PathError{Op: "close", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return &PathError{Op: "close", Path: file.name, Err: ErrClosed}
	}
	if file.localStream != nil {
		return file.closeNativeLocal()
	}

	file.releasePipeEndpoint()
	file.closed = true
	return nil
}

func (file *File) Seek(offset int64, whence int) (int64, error) {
	if file == nil {
		return 0, &PathError{Op: "seek", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return 0, &PathError{Op: "seek", Path: file.name, Err: ErrClosed}
	}
	if file.fdBacked || file.localStream != nil {
		return 0, &PathError{Op: "seek", Path: file.name, Err: ErrInvalid}
	}
	unlock := file.lockSharedPosition()
	defer unlock()

	base := int64(0)
	switch whence {
	case io.SeekStart:
		base = 0
	case io.SeekCurrent:
		base = int64(file.offset)
	case io.SeekEnd:
		info, status := kos.GetPathInfo(file.name)
		if status != kos.FileSystemOK {
			return 0, wrapPathError("seek", file.name, status)
		}
		base = int64(info.Size)
	default:
		return int64(file.offset), &PathError{Op: "seek", Path: file.name, Err: ErrInvalid}
	}

	position := base + offset
	if position < 0 {
		return int64(file.offset), &PathError{Op: "seek", Path: file.name, Err: ErrInvalid}
	}

	file.offset = uint64(position)
	return position, nil
}

func (file *File) ReadAt(buffer []byte, off int64) (int, error) {
	if err := file.ensureReadable("read"); err != nil {
		return 0, err
	}
	if off < 0 {
		return 0, &PathError{Op: "read", Path: file.name, Err: ErrInvalid}
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	if file.fdBacked || file.localStream != nil {
		return 0, &PathError{Op: "read", Path: file.name, Err: ErrInvalid}
	}

	read, status := kos.ReadFile(file.name, buffer, uint64(off))
	switch status {
	case kos.FileSystemOK:
		if read == 0 {
			return 0, io.EOF
		}
		return int(read), nil
	case kos.FileSystemEOF:
		if read > 0 {
			return int(read), io.EOF
		}
		return 0, io.EOF
	default:
		return int(read), wrapPathError("read", file.name, status)
	}
}

func (file *File) Read(buffer []byte) (int, error) {
	if err := file.ensureReadable("read"); err != nil {
		return 0, err
	}
	if file.localStream != nil {
		return file.readNativeLocal(buffer)
	}
	if file.name == DevNull {
		if len(buffer) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	if file.fdBacked {
		if file.usesActiveConsoleInput() && kos.HasActiveConsole() {
			return file.readActiveConsole(buffer)
		}
		if file.pipe != nil && file.pipe.pending == 0 && file.pipe.writers == 0 {
			return 0, io.EOF
		}

		read, err := syscall.Read(file.fd, buffer)
		if err != nil {
			return read, &PathError{Op: "read", Path: file.name, Err: err}
		}
		if file.pipe != nil && read > 0 {
			file.pipe.consume(uint64(read))
		}
		if read == 0 {
			return 0, io.EOF
		}
		return read, nil
	}

	unlock := file.lockSharedPosition()
	defer unlock()
	read, status := kos.ReadFile(file.name, buffer, file.offset)
	file.offset += uint64(read)

	switch status {
	case kos.FileSystemOK:
		if read == 0 {
			return 0, io.EOF
		}
		return int(read), nil
	case kos.FileSystemEOF:
		if read > 0 {
			return int(read), io.EOF
		}
		return 0, io.EOF
	default:
		return int(read), wrapPathError("read", file.name, status)
	}
}

func (file *File) Write(buffer []byte) (int, error) {
	if err := file.ensureWritable("write"); err != nil {
		return 0, err
	}
	if file.localStream != nil {
		return file.writeNativeLocal(buffer)
	}
	if file.name == DevNull {
		return len(buffer), nil
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	if file.fdBacked {
		if file.usesActiveConsole() && kos.HasActiveConsole() {
			written, err := kos.WriteActiveConsole(buffer)
			if err != nil {
				return written, &PathError{Op: "write", Path: file.name, Err: err}
			}
			if written != len(buffer) {
				return written, io.ErrShortWrite
			}

			return written, nil
		}
		if file.pipe != nil && file.pipe.readers == 0 {
			return 0, &PathError{Op: "write", Path: file.name, Err: syscall.EPIPE}
		}

		written, err := syscall.Write(file.fd, buffer)
		if err != nil {
			return written, &PathError{Op: "write", Path: file.name, Err: err}
		}
		if file.pipe != nil && written > 0 {
			file.pipe.pending += uint64(written)
		}
		if written != len(buffer) {
			return written, io.ErrShortWrite
		}
		return written, nil
	}

	unlock := file.lockSharedPosition()
	defer unlock()
	if file.append {
		info, status := kos.GetPathInfo(file.name)
		if status != kos.FileSystemOK {
			return 0, wrapPathError("write", file.name, status)
		}
		file.offset = info.Size
	}

	written, status := kos.WriteFile(file.name, buffer, file.offset)
	file.offset += uint64(written)

	if status != kos.FileSystemOK {
		return int(written), wrapPathError("write", file.name, status)
	}
	if int(written) != len(buffer) {
		return int(written), io.ErrShortWrite
	}

	return int(written), nil
}

func (file *File) WriteString(value string) (int, error) {
	return file.Write([]byte(value))
}

func (file *File) Sync() error {
	if file == nil {
		return &PathError{Op: "sync", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return &PathError{Op: "sync", Path: file.name, Err: ErrClosed}
	}
	return nil
}

func (file *File) readActiveConsole(buffer []byte) (int, error) {
	if kos.ConsoleInputRaw() {
		return file.ReadCancelable(buffer, nil)
	}
	if len(file.pending) == 0 {
		line := make([]byte, activeConsoleReadBufferSize)
		read, err := kos.ReadActiveConsoleLine(line)
		if err != nil {
			return 0, io.EOF
		}
		file.pending = line[:read]
	}

	read := copy(buffer, file.pending)
	file.pending = file.pending[read:]
	if read == 0 {
		return 0, io.EOF
	}

	return read, nil
}

func (file *File) ensureReadable(op string) error {
	if file == nil {
		return &PathError{Op: op, Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return &PathError{Op: op, Path: file.name, Err: ErrClosed}
	}
	if !file.readable {
		return &PathError{Op: op, Path: file.name, Err: ErrPermission}
	}

	return nil
}

func (file *File) ensureWritable(op string) error {
	if file == nil {
		return &PathError{Op: op, Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return &PathError{Op: op, Path: file.name, Err: ErrClosed}
	}
	if !file.writable {
		return &PathError{Op: op, Path: file.name, Err: ErrPermission}
	}

	return nil
}

func (file *File) usesActiveConsole() bool {
	if file == nil || !file.fdBacked {
		return false
	}

	return file.fd == int(kos.StdoutFD) || file.fd == int(kos.StderrFD)
}

func (file *File) usesActiveConsoleInput() bool {
	return file != nil && file.fdBacked && file.fd == int(kos.StdinFD)
}

func wrapPathError(op string, name string, status kos.FileSystemStatus) error {
	return &PathError{
		Op:   op,
		Path: name,
		Err:  statusToError(status),
	}
}

func statusToError(status kos.FileSystemStatus) error {
	switch status {
	case kos.FileSystemOK:
		return nil
	case kos.FileSystemAlreadyExists:
		return ErrExist
	case kos.FileSystemNotFound:
		return ErrNotExist
	case kos.FileSystemAccessDenied:
		return ErrPermission
	case kos.FileSystemUnsupported, kos.FileSystemBadPointer:
		return ErrInvalid
	case kos.FileSystemDiskFull:
		return &statusError{status: status, text: "disk full"}
	case kos.FileSystemInternalError:
		return &statusError{status: status, text: "internal error"}
	case kos.FileSystemDeviceError:
		return &statusError{status: status, text: "device error"}
	case kos.FileSystemNeedsMoreMemory:
		return &statusError{status: status, text: "not enough memory"}
	case kos.FileSystemEOF:
		return io.EOF
	}

	return &statusError{
		status: status,
		text:   "filesystem status " + formatStatus(status),
	}
}

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

func formatStatus(status kos.FileSystemStatus) string {
	return formatUint32(uint32(status))
}

func formatUint32(value uint32) string {
	if value < 10 {
		return decimalDigits[value]
	}

	return formatUint32(value/10) + decimalDigits[value%10]
}

func newDescriptorFile(name string, fd int, readable bool, writable bool) *File {
	return &File{
		name:     name,
		fd:       fd,
		readable: readable,
		writable: writable,
		fdBacked: true,
	}
}

func newPipeFile(name string, fd int, readable bool, writable bool, pipe *pipeState) *File {
	file := newDescriptorFile(name, fd, readable, writable)
	file.pipe = pipe
	return file
}

func (file *File) releasePipeEndpoint() {
	if file == nil || file.pipe == nil {
		return
	}

	if file.readable && file.pipe.readers > 0 {
		file.pipe.readers--
	}
	if file.writable && file.pipe.writers > 0 {
		file.pipe.writers--
	}
}

func (pipe *pipeState) consume(count uint64) {
	if pipe == nil {
		return
	}
	if count >= pipe.pending {
		pipe.pending = 0
		return
	}

	pipe.pending -= count
}

func baseName(name string) string {
	if name == "" {
		return "."
	}

	end := len(name)
	for end > 1 && name[end-1] == '/' {
		end--
	}
	name = name[:end]
	if name == "" {
		return "/"
	}

	lastSlash := -1
	for index := len(name) - 1; index >= 0; index-- {
		if name[index] == '/' {
			lastSlash = index
			break
		}
	}

	if lastSlash < 0 {
		return name
	}
	if lastSlash == 0 && len(name) == 1 {
		return "/"
	}

	return name[lastSlash+1:]
}

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	daysPer400Years  = 146097
	unixEpochDays    = 719468
)

func fileStampTime(date kos.FileDate, clock kos.FileTime) time.Time {
	if date.Year == 0 || date.Month == 0 || date.Day == 0 {
		return time.Time{}
	}

	days := daysFromCivil(int(date.Year), int(date.Month), int(date.Day))
	seconds := int64(days*secondsPerDay +
		int(clock.Hour)*secondsPerHour +
		int(clock.Minute)*secondsPerMinute +
		int(clock.Second))

	return time.Unix(seconds, 0)
}

func daysFromCivil(year int, month int, day int) int {
	if month <= 2 {
		year--
	}

	era := year / 400
	if year < 0 && year%400 != 0 {
		era--
	}
	yearOfEra := year - era*400
	monthPrime := month
	if monthPrime > 2 {
		monthPrime -= 3
	} else {
		monthPrime += 9
	}

	dayOfYear := ((153 * monthPrime) + 2) / 5
	dayOfYear += day - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return era*daysPer400Years + dayOfEra - unixEpochDays
}
