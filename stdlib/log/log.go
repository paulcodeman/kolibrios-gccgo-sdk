package log

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const (
	Ldate         = 1 << iota
	Ltime
	Lmicroseconds
	Llongfile
	Lshortfile
	LUTC
	Lmsgprefix
)

const LstdFlags = Ldate | Ltime

type Logger struct {
	out    io.Writer
	prefix string
	flag   int
}

var std Logger

func New(out io.Writer, prefix string, flag int) *Logger {
	if out == nil {
		out = os.DefaultStderr()
	}

	return &Logger{
		out:    out,
		prefix: prefix,
		flag:   flag,
	}
}

func Default() *Logger {
	if std.out == nil {
		std.out = os.DefaultStderr()
		std.flag = LstdFlags
	}

	return &std
}

func (logger *Logger) Flags() int {
	return logger.flag
}

func (logger *Logger) SetFlags(flag int) {
	logger.flag = flag
}

func (logger *Logger) Prefix() string {
	return logger.prefix
}

func (logger *Logger) SetPrefix(prefix string) {
	logger.prefix = prefix
}

func (logger *Logger) Writer() io.Writer {
	if logger.out == nil {
		logger.out = os.DefaultStderr()
	}

	return logger.out
}

func (logger *Logger) SetOutput(writer io.Writer) {
	if writer == nil {
		writer = os.DefaultStderr()
	}

	logger.out = writer
}

func (logger *Logger) Output(calldepth int, text string) error {
	_ = calldepth

	writer := logger.Writer()
	line := logger.formatHeader() + text
	if len(line) == 0 || line[len(line)-1] != '\n' {
		line += "\n"
	}

	written, err := io.WriteString(writer, line)
	if err != nil {
		return err
	}
	if written != len(line) {
		return io.ErrShortWrite
	}

	return nil
}

func (logger *Logger) Print(values ...interface{}) {
	_ = logger.Output(2, fmt.Sprint(values...))
}

func (logger *Logger) Printf(format string, values ...interface{}) {
	_ = logger.Output(2, fmt.Sprintf(format, values...))
}

func (logger *Logger) Println(values ...interface{}) {
	_ = logger.Output(2, fmt.Sprintln(values...))
}

func Flags() int {
	return Default().Flags()
}

func SetFlags(flag int) {
	Default().SetFlags(flag)
}

func Prefix() string {
	return Default().Prefix()
}

func SetPrefix(prefix string) {
	Default().SetPrefix(prefix)
}

func Writer() io.Writer {
	return Default().Writer()
}

func SetOutput(writer io.Writer) {
	Default().SetOutput(writer)
}

func Output(calldepth int, text string) error {
	return Default().Output(calldepth+1, text)
}

func Print(values ...interface{}) {
	Default().Print(values...)
}

func Printf(format string, values ...interface{}) {
	Default().Printf(format, values...)
}

func Println(values ...interface{}) {
	Default().Println(values...)
}

func (logger *Logger) formatHeader() string {
	header := make([]byte, 0, 64)
	flags := logger.flag

	if flags&Lmsgprefix == 0 {
		header = append(header, logger.prefix...)
	}

	wroteHeader := false
	now := time.Now()
	if flags&LUTC != 0 {
		now = now.UTC()
	}

	if flags&Ldate != 0 {
		header = appendPaddedInt(header, now.Year(), 4)
		header = append(header, '/')
		header = appendPaddedInt(header, int(now.Month()), 2)
		header = append(header, '/')
		header = appendPaddedInt(header, now.Day(), 2)
		wroteHeader = true
	}

	if flags&(Ltime|Lmicroseconds) != 0 {
		if wroteHeader {
			header = append(header, ' ')
		}
		header = appendPaddedInt(header, now.Hour(), 2)
		header = append(header, ':')
		header = appendPaddedInt(header, now.Minute(), 2)
		header = append(header, ':')
		header = appendPaddedInt(header, now.Second(), 2)
		if flags&Lmicroseconds != 0 {
			header = append(header, '.')
			header = appendPaddedInt(header, now.Nanosecond()/1000, 6)
		}
		wroteHeader = true
	}

	// File-location flags exist for compatibility, but bootstrap runtime does
	// not yet provide caller file/line metadata for log headers.
	_ = flags & (Llongfile | Lshortfile)

	if wroteHeader {
		header = append(header, ' ')
	}

	if flags&Lmsgprefix != 0 {
		header = append(header, logger.prefix...)
	}

	return string(header)
}

func appendPaddedInt(dst []byte, value int, width int) []byte {
	text := strconv.AppendInt(nil, int64(value), 10)
	for index := len(text); index < width; index++ {
		dst = append(dst, '0')
	}

	return append(dst, text...)
}
