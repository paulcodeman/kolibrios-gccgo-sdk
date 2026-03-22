package io

import "sync"

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
var ErrClosedPipe = &ioError{text: "io: read/write on closed pipe"}

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

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

type ReadCloser interface {
	Reader
	Closer
}

type LimitedReader struct {
	R Reader
	N int64
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

func LimitReader(r Reader, n int64) Reader { return &LimitedReader{r, n} }

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

func (limited *LimitedReader) Read(p []byte) (n int, err error) {
	if limited.N <= 0 {
		return 0, EOF
	}
	if int64(len(p)) > limited.N {
		p = p[0:limited.N]
	}
	n, err = limited.R.Read(p)
	limited.N -= int64(n)
	return
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

func CopyN(dst Writer, src Reader, n int64) (written int64, err error) {
	if n <= 0 {
		return 0, nil
	}
	buffer := make([]byte, 512)
	for n > 0 {
		toRead := len(buffer)
		if int64(toRead) > n {
			toRead = int(n)
		}
		read, readErr := src.Read(buffer[:toRead])
		if read > 0 {
			wrote, writeErr := dst.Write(buffer[:read])
			written += int64(wrote)
			if writeErr != nil {
				return written, writeErr
			}
			if wrote != read {
				return written, ErrShortWrite
			}
			n -= int64(wrote)
		}
		if readErr != nil {
			if readErr == EOF && n == 0 {
				return written, nil
			}
			if readErr == EOF {
				return written, ErrUnexpectedEOF
			}
			return written, readErr
		}
	}
	return written, nil
}

func WriteString(w Writer, s string) (n int, err error) {
	return w.Write([]byte(s))
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, EOF
}

type multiReader struct {
	readers []Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*multiReader); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF {
			mr.readers[0] = eofReader{}
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF {
			if err == EOF && len(mr.readers) > 0 {
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

func (mr *multiReader) WriteTo(w Writer) (sum int64, err error) {
	buffer := make([]byte, 1024*32)
	for i, r := range mr.readers {
		var n int64
		if sub, ok := r.(*multiReader); ok {
			n, err = sub.WriteTo(w)
		} else {
			n, err = CopyBuffer(w, r, buffer)
		}
		sum += n
		if err != nil {
			mr.readers = mr.readers[i:]
			return sum, err
		}
		mr.readers[i] = nil
	}
	mr.readers = nil
	return sum, nil
}

func MultiReader(readers ...Reader) Reader {
	r := make([]Reader, len(readers))
	copy(r, readers)
	return &multiReader{readers: r}
}

type multiWriter struct {
	writers []Writer
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			return n, ErrShortWrite
		}
	}
	return len(p), nil
}

func (mw *multiWriter) WriteString(s string) (n int, err error) {
	var p []byte
	for _, w := range mw.writers {
		if sw, ok := w.(StringWriter); ok {
			n, err = sw.WriteString(s)
		} else {
			if p == nil {
				p = []byte(s)
			}
			n, err = w.Write(p)
		}
		if err != nil {
			return
		}
		if n != len(s) {
			return n, ErrShortWrite
		}
	}
	return len(s), nil
}

func MultiWriter(writers ...Writer) Writer {
	allWriters := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter{writers: allWriters}
}

type teeReader struct {
	r Reader
	w Writer
}

func (tr *teeReader) Read(p []byte) (n int, err error) {
	n, err = tr.r.Read(p)
	if n > 0 {
		if _, writeErr := tr.w.Write(p[:n]); writeErr != nil {
			return n, writeErr
		}
	}
	return
}

func TeeReader(r Reader, w Writer) Reader {
	return &teeReader{r: r, w: w}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

// Discard is a Writer on which all Write calls succeed without doing anything.
var Discard Writer = discard{}

// PipeReader is the read half of a pipe.
type PipeReader struct {
	p *pipe
}

// PipeWriter is the write half of a pipe.
type PipeWriter struct {
	p *pipe
}

type pipe struct {
	mu       sync.Mutex
	done     chan struct{}
	doneOnce sync.Once
	ch       chan []byte
	buf      []byte
	rerr     error
	werr     error
	rclosed  bool
	wclosed  bool
}

// Pipe creates a synchronous in-memory pipe. It can be used to connect
// code expecting an io.Reader with code expecting an io.Writer.
func Pipe() (*PipeReader, *PipeWriter) {
	p := &pipe{
		done: make(chan struct{}),
		ch:   make(chan []byte),
	}
	return &PipeReader{p: p}, &PipeWriter{p: p}
}

func (p *pipe) closeDone() {
	if p == nil {
		return
	}
	p.doneOnce.Do(func() {
		close(p.done)
	})
}

func (p *pipe) read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	for {
		p.mu.Lock()
		if len(p.buf) > 0 {
			n := copy(b, p.buf)
			p.buf = p.buf[n:]
			p.mu.Unlock()
			return n, nil
		}
		if p.rclosed {
			p.mu.Unlock()
			return 0, ErrClosedPipe
		}
		if p.wclosed {
			err := p.werr
			p.mu.Unlock()
			if err != nil {
				return 0, err
			}
			return 0, EOF
		}
		ch := p.ch
		done := p.done
		p.mu.Unlock()

		select {
		case data := <-ch:
			if len(data) == 0 {
				continue
			}
			n := copy(b, data)
			if n < len(data) {
				p.mu.Lock()
				p.buf = append(p.buf, data[n:]...)
				p.mu.Unlock()
			}
			return n, nil
		case <-done:
			p.mu.Lock()
			err := p.werr
			p.mu.Unlock()
			if err != nil {
				return 0, err
			}
			return 0, EOF
		}
	}
}

func (p *pipe) write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	p.mu.Lock()
	if p.wclosed || p.rclosed {
		p.mu.Unlock()
		return 0, ErrClosedPipe
	}
	ch := p.ch
	done := p.done
	p.mu.Unlock()

	data := make([]byte, len(b))
	copy(data, b)

	select {
	case <-done:
		return 0, ErrClosedPipe
	case ch <- data:
		return len(b), nil
	}
}

func (p *pipe) closeRead(err error) error {
	p.mu.Lock()
	if p.rclosed {
		p.mu.Unlock()
		return ErrClosedPipe
	}
	p.rclosed = true
	if err == nil {
		err = ErrClosedPipe
	}
	p.rerr = err
	p.mu.Unlock()
	p.closeDone()
	return nil
}

func (p *pipe) closeWrite(err error) error {
	p.mu.Lock()
	if p.wclosed {
		p.mu.Unlock()
		return ErrClosedPipe
	}
	p.wclosed = true
	if err == nil {
		err = EOF
	}
	p.werr = err
	p.mu.Unlock()
	p.closeDone()
	return nil
}

// Read reads data from the pipe.
func (r *PipeReader) Read(b []byte) (int, error) {
	if r == nil || r.p == nil {
		return 0, ErrClosedPipe
	}
	return r.p.read(b)
}

// Close closes the reader; subsequent writes will return ErrClosedPipe.
func (r *PipeReader) Close() error {
	return r.CloseWithError(nil)
}

// CloseWithError closes the reader with the provided error.
func (r *PipeReader) CloseWithError(err error) error {
	if r == nil || r.p == nil {
		return ErrClosedPipe
	}
	return r.p.closeRead(err)
}

// Write writes data to the pipe.
func (w *PipeWriter) Write(b []byte) (int, error) {
	if w == nil || w.p == nil {
		return 0, ErrClosedPipe
	}
	return w.p.write(b)
}

// Close closes the writer; subsequent reads will return EOF.
func (w *PipeWriter) Close() error {
	return w.CloseWithError(nil)
}

// CloseWithError closes the writer with the provided error.
func (w *PipeWriter) CloseWithError(err error) error {
	if w == nil || w.p == nil {
		return ErrClosedPipe
	}
	return w.p.closeWrite(err)
}
