package kos

const (
	StdinFD  uint32 = 0
	StdoutFD uint32 = 1
	StderrFD uint32 = 2

	PipeCloseOnExec uint32 = 0x40000
)

func FDRead(fd uint32, buffer []byte) int {
	if len(buffer) == 0 {
		return 0
	}

	return PosixReadRaw(fd, &buffer[0], uint32(len(buffer)))
}

func FDWrite(fd uint32, buffer []byte) int {
	if len(buffer) == 0 {
		return 0
	}

	return PosixWriteRaw(fd, &buffer[0], uint32(len(buffer)))
}

func CreatePipe(flags uint32) (readFD uint32, writeFD uint32, result int) {
	var pipefd [2]uint32

	result = PosixPipe2Raw(&pipefd[0], flags)
	if result != 0 {
		return 0, 0, result
	}

	return pipefd[0], pipefd[1], 0
}
