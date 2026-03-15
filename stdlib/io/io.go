package io

type ioError struct {
	text string
}

func (err *ioError) Error() string {
	return err.text
}

var EOF = &ioError{text: "EOF"}
var ErrShortWrite = &ioError{text: "short write"}
var ErrShortBuffer = &ioError{text: "short buffer"}
var ErrUnexpectedEOF = &ioError{text: "unexpected EOF"}
var ErrNoProgress = &ioError{text: "multiple Read calls return no data or error"}

type Reader interface {
	Read(p []byte) (n int, err error)
}

type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

type ByteReader interface {
	ReadByte() (byte, error)
}

type ByteWriter interface {
	WriteByte(c byte) error
}

type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

type RuneScanner interface {
	RuneReader
	UnreadRune() error
}

type Seeker interface {
	Seek(offset int64, whence int) (int64, error)
}

type Closer interface {
	Close() error
}

type ReadSeeker interface {
	Reader
	Seeker
}

type ReadWriter interface {
	Reader
	Writer
}

type ReadCloser interface {
	Reader
	Closer
}

type WriteCloser interface {
	Writer
	Closer
}

type StringWriter interface {
	WriteString(s string) (n int, err error)
}

const (
	SeekStart   = 0
	SeekCurrent = 1
	SeekEnd     = 2
)

func ReadAll(r Reader) ([]byte, error) {
	data := make([]byte, 0, 512)
	buffer := make([]byte, 512)

	for {
		read, err := r.Read(buffer)
		if read > 0 {
			data = append(data, buffer[:read]...)
		}

		if err != nil {
			if err == EOF {
				return data, nil
			}

			return data, err
		}
	}
}

func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		return n, err
	}
	if n > 0 && err == EOF {
		err = ErrUnexpectedEOF
	}
	return n, err
}

func ReadFull(r Reader, buf []byte) (n int, err error) {
	return ReadAtLeast(r, buf, len(buf))
}

func Copy(dst Writer, src Reader) (written int64, err error) {
	if writerTo, ok := src.(WriterTo); ok {
		return writerTo.WriteTo(dst)
	}
	if readerFrom, ok := dst.(ReaderFrom); ok {
		return readerFrom.ReadFrom(src)
	}

	return CopyBuffer(dst, src, nil)
}

func CopyBuffer(dst Writer, src Reader, buffer []byte) (written int64, err error) {
	if writerTo, ok := src.(WriterTo); ok {
		return writerTo.WriteTo(dst)
	}
	if readerFrom, ok := dst.(ReaderFrom); ok {
		return readerFrom.ReadFrom(src)
	}

	if len(buffer) == 0 {
		buffer = make([]byte, 512)
	}

	for {
		read, readErr := src.Read(buffer)
		if read > 0 {
			wrote, writeErr := dst.Write(buffer[:read])
			written += int64(wrote)

			if writeErr != nil {
				return written, writeErr
			}
			if wrote != read {
				return written, ErrShortWrite
			}
		}

		if readErr != nil {
			if readErr == EOF {
				return written, nil
			}

			return written, readErr
		}
	}
}

func WriteString(w Writer, s string) (n int, err error) {
	return w.Write([]byte(s))
}
