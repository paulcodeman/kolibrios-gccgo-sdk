package os

import (
	"io"
	"kos"
	"syscall"
	"time"
)

type FileMode uint32

const (
	PathSeparator              = '/'
	PathListSeparator          = ':'
	ModeDir           FileMode = 1 << 31
)

func (mode FileMode) IsDir() bool {
	return mode&ModeDir != 0
}

const (
	O_RDONLY int = 0
	O_WRONLY int = 1
	O_RDWR   int = 2

	O_CREATE int = 0x40
	O_TRUNC  int = 0x200
	O_APPEND int = 0x400
)

type osError struct {
	text string
}

func (err *osError) Error() string {
	return err.text
}

var ErrInvalid = &osError{text: "invalid argument"}
var ErrPermission = &osError{text: "permission denied"}
var ErrExist = &osError{text: "file already exists"}
var ErrNotExist = &osError{text: "file does not exist"}
var ErrClosed = &osError{text: "file already closed"}

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

type PathError struct {
	Op   string
	Path string
	Err  error
}

func (err *PathError) Error() string {
	if err == nil {
		return ""
	}
	if err.Err == nil {
		return err.Op + " " + err.Path
	}

	return err.Op + " " + err.Path + ": " + err.Err.Error()
}

func (err *PathError) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Err
}

func (err *PathError) As(target interface{}) bool {
	if err == nil {
		return false
	}

	switch typed := target.(type) {
	case **PathError:
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

type FileInfo interface {
	Name() string
	Size() int64
	Mode() FileMode
	ModTime() time.Time
	IsDir() bool
	Sys() interface{}
}

type fileInfo struct {
	name string
	raw  kos.FileInfo
}

func (info fileInfo) Name() string {
	return info.name
}

func (info fileInfo) Size() int64 {
	return int64(info.raw.Size)
}

func (info fileInfo) Mode() FileMode {
	mode := FileMode(0)
	if info.raw.Attributes&kos.FileAttributeDirectory != 0 {
		mode |= ModeDir
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
	name     string
	fd       int
	offset   uint64
	readable bool
	writable bool
	append   bool
	closed   bool
	fdBacked bool
	pending  []byte
	pipe     *pipeState
}

const activeConsoleReadBufferSize = 256

type pipeState struct {
	pending uint64
	readers uint32
	writers uint32
}

type envEntry struct {
	key   string
	value string
}

var Args = []string{""}

var envEntries []envEntry
var errInvalidEnv = &osError{text: "invalid environment variable"}

func bootstrapArgs() {
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
	kos.Exit()
}

func Getenv(key string) string {
	value, _ := LookupEnv(key)
	return value
}

func LookupEnv(key string) (string, bool) {
	index := findEnvEntry(key)
	if index < 0 {
		return "", false
	}

	return envEntries[index].value, true
}

func Setenv(key string, value string) error {
	if !validEnvKey(key) || containsNUL(value) {
		return errInvalidEnv
	}

	index := findEnvEntry(key)
	if index >= 0 {
		envEntries[index].value = value
		return nil
	}

	envEntries = append(envEntries, envEntry{
		key:   key,
		value: value,
	})
	return nil
}

func Unsetenv(key string) error {
	if !validEnvKey(key) {
		return errInvalidEnv
	}

	index := findEnvEntry(key)
	if index < 0 {
		return nil
	}

	for current := index; current+1 < len(envEntries); current++ {
		envEntries[current] = envEntries[current+1]
	}
	envEntries = envEntries[:len(envEntries)-1]
	return nil
}

func Clearenv() {
	envEntries = nil
}

func Environ() []string {
	values := make([]string, len(envEntries))
	for index := 0; index < len(envEntries); index++ {
		values[index] = envEntries[index].key + "=" + envEntries[index].value
	}

	return values
}

func Stat(name string) (FileInfo, error) {
	info, status := kos.GetPathInfo(name)
	if status != kos.FileSystemOK {
		return nil, wrapPathError("stat", name, status)
	}

	return fileInfo{
		name: baseName(name),
		raw:  info,
	}, nil
}

func IsNotExist(err error) bool {
	return errorMatches(err, ErrNotExist)
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
	status := kos.CreateDirectory(name)
	if status != kos.FileSystemOK {
		return wrapPathError("mkdir", name, status)
	}

	return nil
}

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
	var pipefd [2]int

	if err = syscall.Pipe(pipefd[:]); err != nil {
		return nil, nil, err
	}

	pipe := &pipeState{
		readers: 1,
		writers: 1,
	}
	reader = newPipeFile("pipe[0]", pipefd[0], true, false, pipe)
	writer = newPipeFile("pipe[1]", pipefd[1], false, true, pipe)
	return reader, writer, nil
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	accessMode := flag & 3
	readable := accessMode == O_RDONLY || accessMode == O_RDWR
	writable := accessMode == O_WRONLY || accessMode == O_RDWR

	if flag&O_TRUNC != 0 && !writable {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}
	if flag&O_APPEND != 0 && !writable {
		return nil, &PathError{Op: "open", Path: name, Err: ErrInvalid}
	}

	if flag&O_CREATE != 0 {
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

	if flag&O_TRUNC != 0 {
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
		name:     name,
		readable: readable,
		writable: writable,
		append:   flag&O_APPEND != 0,
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

func (file *File) Close() error {
	if file == nil {
		return &PathError{Op: "close", Path: "", Err: ErrInvalid}
	}
	if file.closed {
		return &PathError{Op: "close", Path: file.name, Err: ErrClosed}
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
	if file.fdBacked {
		return 0, &PathError{Op: "seek", Path: file.name, Err: ErrInvalid}
	}

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
	if file.fdBacked {
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

func (file *File) readActiveConsole(buffer []byte) (int, error) {
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

func errorMatches(err error, target error) bool {
	for err != nil {
		if err == target {
			return true
		}

		unwrapper, ok := interface{}(err).(interface{ Unwrap() error })
		if !ok {
			return false
		}

		err = unwrapper.Unwrap()
	}

	return target == nil
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}

	for index := 0; index < len(key); index++ {
		if key[index] == '=' || key[index] == 0 {
			return false
		}
	}

	return true
}

func containsNUL(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] == 0 {
			return true
		}
	}

	return false
}

func findEnvEntry(key string) int {
	for index := 0; index < len(envEntries); index++ {
		if envEntries[index].key == key {
			return index
		}
	}

	return -1
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
