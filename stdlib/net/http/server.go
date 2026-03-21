package http

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	urlpkg "net/url"
	"strconv"
	"strings"
)

var ErrServerClosed = errors.New("http: Server closed")

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (fn HandlerFunc) ServeHTTP(writer ResponseWriter, request *Request) {
	fn(writer, request)
}

type ResponseWriter interface {
	Header() Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

type Server struct {
	Addr    string
	Handler Handler
}

func ListenAndServe(addr string, handler Handler) error {
	return (&Server{
		Addr:    addr,
		Handler: handler,
	}).ListenAndServe()
}

func Serve(listener net.Listener, handler Handler) error {
	return (&Server{Handler: handler}).Serve(listener)
}

func (srv *Server) ListenAndServe() error {
	addr := ":80"
	if srv != nil && srv.Addr != "" {
		addr = srv.Addr
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(listener)
}

func (srv *Server) Serve(listener net.Listener) error {
	if listener == nil {
		return errors.New("http: nil Listener")
	}
	defer listener.Close()

	handler := defaultServerHandler(nil)
	if srv != nil {
		handler = defaultServerHandler(srv.Handler)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return ErrServerClosed
			}
			return err
		}
		// KolibriOS sockets are not yet integrated with a runtime netpoller, so
		// serving inline keeps the single-threaded server responsive.
		serveHTTPConn(conn, handler)
	}
}

func defaultServerHandler(handler Handler) Handler {
	if handler != nil {
		return handler
	}
	return HandlerFunc(NotFound)
}

func serveHTTPConn(conn net.Conn, handler Handler) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	request, err := ReadRequest(reader)
	if err != nil {
		if err != io.EOF {
			writeConnError(conn, StatusBadRequest, "Bad Request\n")
		}
		return
	}
	defer request.Body.Close()

	request.RemoteAddr = addrString(conn.RemoteAddr())
	request.LocalAddr = addrString(conn.LocalAddr())
	request.Close = true

	response := newBufferedResponseWriter(conn, request)
	defer func() {
		if recovered := recover(); recovered != nil {
			response.reset(StatusInternalServerError)
			Error(response, "Internal Server Error", StatusInternalServerError)
		}
		_ = response.finish()
	}()

	handler.ServeHTTP(response, request)
}

func addrString(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}

func ReadRequest(reader *bufio.Reader) (*Request, error) {
	if reader == nil {
		return nil, errors.New("http: nil Reader")
	}

	line, err := readHTTPLine(reader)
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, io.EOF
	}

	method, requestURI, proto, ok := parseRequestLine(line)
	if !ok {
		return nil, fmt.Errorf("malformed HTTP request %s", quote(line))
	}

	parsedURL, err := urlpkg.Parse(requestURI)
	if err != nil {
		return nil, err
	}

	header, err := readHTTPHeaders(reader)
	if err != nil {
		return nil, err
	}

	protoMajor, protoMinor := parseProto(proto)
	if proto == "" || (protoMajor == 0 && protoMinor == 0 && proto != "HTTP/0.0") {
		return nil, fmt.Errorf("malformed HTTP version %s", quote(proto))
	}

	host := parsedURL.Host
	if host == "" {
		host = header.Get("Host")
	}

	bodyData, contentLength, transferEncoding, err := readRequestBody(reader, header)
	if err != nil {
		return nil, err
	}

	body := NoBody
	if len(bodyData) > 0 {
		body = newMemoryBody(bodyData)
	}

	return &Request{
		Method:           method,
		URL:              parsedURL,
		Proto:            proto,
		ProtoMajor:       protoMajor,
		ProtoMinor:       protoMinor,
		Header:           header,
		Body:             body,
		ContentLength:    contentLength,
		TransferEncoding: transferEncoding,
		Close:            shouldCloseRequest(protoMajor, protoMinor, header),
		Host:             host,
		RequestURI:       requestURI,
	}, nil
}

func parseRequestLine(line string) (method string, requestURI string, proto string, ok bool) {
	firstSpace := strings.IndexByte(line, ' ')
	if firstSpace < 0 {
		return "", "", "", false
	}
	secondSpace := strings.IndexByte(line[firstSpace+1:], ' ')
	if secondSpace < 0 {
		return "", "", "", false
	}
	secondSpace += firstSpace + 1

	method = line[:firstSpace]
	requestURI = line[firstSpace+1 : secondSpace]
	proto = line[secondSpace+1:]
	if method == "" || requestURI == "" || proto == "" {
		return "", "", "", false
	}
	return method, requestURI, proto, true
}

func readRequestBody(reader *bufio.Reader, header Header) (bodyData []byte, contentLength int64, transferEncoding []string, err error) {
	transferEncoding = parseTransferEncoding(header)
	switch {
	case hasChunkedEncoding(transferEncoding):
		bodyData, err = readChunkedBody(reader)
		if err != nil {
			return nil, -1, transferEncoding, err
		}
		return bodyData, int64(len(bodyData)), transferEncoding, nil
	default:
		contentLength = parsedContentLength(header)
		if contentLength <= 0 {
			if contentLength < 0 {
				contentLength = 0
			}
			return nil, contentLength, nil, nil
		}

		bodyData, err = io.ReadAll(io.LimitReader(reader, contentLength))
		if err != nil {
			return nil, 0, nil, err
		}
		if int64(len(bodyData)) != contentLength {
			return nil, 0, nil, io.ErrUnexpectedEOF
		}
		return bodyData, contentLength, nil, nil
	}
}

func parseTransferEncoding(header Header) []string {
	values := header.Values("Transfer-Encoding")
	if len(values) == 0 {
		return nil
	}

	encodings := make([]string, 0, len(values))
	for valueIndex := 0; valueIndex < len(values); valueIndex++ {
		parts := strings.Split(values[valueIndex], ",")
		for partIndex := 0; partIndex < len(parts); partIndex++ {
			encoding := strings.TrimSpace(parts[partIndex])
			if encoding != "" {
				encodings = append(encodings, encoding)
			}
		}
	}
	if len(encodings) == 0 {
		return nil
	}
	return encodings
}

func hasChunkedEncoding(encodings []string) bool {
	for index := 0; index < len(encodings); index++ {
		if asciiEqualFold(encodings[index], "chunked") {
			return true
		}
	}
	return false
}

func shouldCloseRequest(protoMajor int, protoMinor int, header Header) bool {
	connectionValues := header.Values("Connection")
	for valueIndex := 0; valueIndex < len(connectionValues); valueIndex++ {
		parts := strings.Split(connectionValues[valueIndex], ",")
		for partIndex := 0; partIndex < len(parts); partIndex++ {
			value := strings.TrimSpace(parts[partIndex])
			if asciiEqualFold(value, "close") {
				return true
			}
			if asciiEqualFold(value, "keep-alive") {
				return false
			}
		}
	}

	return protoMajor < 1 || (protoMajor == 1 && protoMinor == 0)
}

type bufferedResponseWriter struct {
	conn        net.Conn
	request     *Request
	header      Header
	status      int
	wroteHeader bool
	finished    bool
	body        bytes.Buffer
}

func newBufferedResponseWriter(conn net.Conn, request *Request) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		conn:    conn,
		request: request,
		header:  make(Header),
		status:  StatusOK,
	}
}

func (writer *bufferedResponseWriter) Header() Header {
	if writer.header == nil {
		writer.header = make(Header)
	}
	return writer.header
}

func (writer *bufferedResponseWriter) WriteHeader(statusCode int) {
	if writer.finished || writer.wroteHeader {
		return
	}
	if statusCode <= 0 {
		statusCode = StatusOK
	}
	writer.status = statusCode
	writer.wroteHeader = true
}

func (writer *bufferedResponseWriter) Write(data []byte) (int, error) {
	if writer.finished {
		return 0, errors.New("http: write after response finished")
	}
	if !writer.wroteHeader {
		writer.WriteHeader(StatusOK)
	}
	return writer.body.Write(data)
}

func (writer *bufferedResponseWriter) reset(statusCode int) {
	writer.header = make(Header)
	writer.status = statusCode
	writer.wroteHeader = false
	writer.finished = false
	writer.body.Reset()
}

func (writer *bufferedResponseWriter) finish() error {
	if writer.finished {
		return nil
	}
	writer.finished = true
	if !writer.wroteHeader {
		writer.WriteHeader(StatusOK)
	}

	sendBody := true
	headRequest := false
	if writer.request != nil {
		headRequest = normalizeMethod(writer.request.Method) == MethodHead
		sendBody = !responseHasNoBody(normalizeMethod(writer.request.Method), writer.status)
	}

	header := writer.Header()
	if header.Get("Connection") == "" {
		header.Set("Connection", "close")
	}
	if header.Get("Content-Type") == "" && writer.body.Len() > 0 {
		header.Set("Content-Type", "text/html; charset=utf-8")
	}
	if !sendBody {
		if header.Get("Content-Length") == "" {
			if headRequest && writer.status != StatusNoContent && writer.status != 304 &&
				(writer.status < 100 || writer.status >= 200) {
				header.Set("Content-Length", strconv.Itoa(writer.body.Len()))
			} else {
				header.Set("Content-Length", "0")
			}
		}
	} else if header.Get("Content-Length") == "" {
		header.Set("Content-Length", strconv.Itoa(writer.body.Len()))
	}

	buffered := bufio.NewWriter(writer.conn)
	if _, err := buffered.WriteString(statusLine(writer.status)); err != nil {
		return err
	}
	if err := writeResponseHeaders(buffered, header); err != nil {
		return err
	}
	if _, err := buffered.WriteString("\r\n"); err != nil {
		return err
	}
	if sendBody && writer.body.Len() > 0 {
		if _, err := buffered.Write(writer.body.Bytes()); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func statusLine(statusCode int) string {
	statusText := StatusText(statusCode)
	if statusText == "" {
		return "HTTP/1.1 " + strconv.Itoa(statusCode) + "\r\n"
	}
	return "HTTP/1.1 " + strconv.Itoa(statusCode) + " " + statusText + "\r\n"
}

func writeResponseHeaders(writer io.Writer, header Header) error {
	if len(header) == 0 {
		return nil
	}

	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sortStrings(keys)

	for index := 0; index < len(keys); index++ {
		values := header[keys[index]]
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			if _, err := io.WriteString(writer, keys[index]); err != nil {
				return err
			}
			if _, err := io.WriteString(writer, ": "); err != nil {
				return err
			}
			if _, err := io.WriteString(writer, values[valueIndex]); err != nil {
				return err
			}
			if _, err := io.WriteString(writer, "\r\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func Error(writer ResponseWriter, text string, statusCode int) {
	header := writer.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/plain; charset=utf-8")
	}
	writer.WriteHeader(statusCode)
	if text == "" {
		return
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, _ = io.WriteString(writer, text)
}

func NotFound(writer ResponseWriter, request *Request) {
	Error(writer, "Not Found", StatusNotFound)
}

func writeConnError(conn net.Conn, statusCode int, text string) {
	response := newBufferedResponseWriter(conn, nil)
	Error(response, text, statusCode)
	_ = response.finish()
}
