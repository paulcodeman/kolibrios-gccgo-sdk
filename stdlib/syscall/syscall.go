package syscall

import "kos"

type Errno int

const (
	EBADF  Errno = 9
	ENOTTY Errno = 25
	EINVAL Errno = 22
	EFAULT Errno = 14
	ENFILE Errno = 23
	EMFILE Errno = 24
	EPIPE  Errno = 32
)

const O_CLOEXEC = 0x40000

func (errno Errno) Error() string {
	switch errno {
	case ENOMEM:
		return "not enough space"
	case EBADF:
		return "bad file descriptor"
	case EINVAL:
		return "invalid argument"
	case EFAULT:
		return "bad address"
	case ENFILE:
		return "file table overflow"
	case EMFILE:
		return "too many open files"
	case EPIPE:
		return "broken pipe"
	}

	return "errno " + formatInt(int(errno))
}

func (errno Errno) As(target interface{}) bool {
	switch typed := target.(type) {
	case *Errno:
		if typed == nil {
			return false
		}
		*typed = errno
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = errno
		return true
	}

	return false
}

func Read(fd int, buffer []byte) (n int, err error) {
	result := kos.FDRead(uint32(fd), buffer)
	if result < 0 {
		return 0, errnoFromKolibri(-result)
	}

	return result, nil
}

func Write(fd int, buffer []byte) (n int, err error) {
	result := kos.FDWrite(uint32(fd), buffer)
	if result < 0 {
		return 0, errnoFromKolibri(-result)
	}

	return result, nil
}

func Pipe(pipefd []int) error {
	return Pipe2(pipefd, 0)
}

func Pipe2(pipefd []int, flags int) error {
	if len(pipefd) < 2 {
		return EINVAL
	}
	if flags < 0 {
		return EINVAL
	}

	readFD, writeFD, result := kos.CreatePipe(uint32(flags))
	if result < 0 {
		return errnoFromKolibri(-result)
	}

	pipefd[0] = int(readFD)
	pipefd[1] = int(writeFD)
	return nil
}

// The native descriptor ABI (sysfuncs.txt, syscall 77/13) returns -11 for
// EINVAL. The Go compatibility API uses the KolibriOS newlib errno namespace,
// where EINVAL is 22 and 11 is EAGAIN. Other documented FD errors have the
// same values in both namespaces. Socket and syscall 70 errors are separate.
func errnoFromKolibri(code int) Errno {
	if code == 11 {
		return EINVAL
	}
	return Errno(code)
}

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

func formatInt(value int) string {
	if value < 0 {
		return "-" + formatUint(uint(^value)+1)
	}

	return formatUint(uint(value))
}

func formatUint(value uint) string {
	if value < 10 {
		return decimalDigits[value]
	}

	return formatUint(value/10) + decimalDigits[value%10]
}
