package http

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"kos"
	"net"
	urlpkg "net/url"
	"strconv"
	"strings"
)

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodPatch   = "PATCH"
	MethodOptions = "OPTIONS"
	MethodConnect = "CONNECT"
	MethodTrace   = "TRACE"
)

const (
	StatusCreated             = 201
	StatusAccepted            = 202
	StatusNoContent           = 204
	StatusOK                  = 200
	StatusMovedPermanently    = 301
	StatusFound               = 302
	StatusSeeOther            = 303
	StatusTemporaryRedirect   = 307
	StatusPermanentRedirect   = 308
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusConflict            = 409
	StatusGone                = 410
	StatusRequestTimeout      = 408
	StatusUnprocessableEntity = 422
	StatusInternalServerError = 500
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
)

type Header map[string][]string

type Request struct {
	Method        string
	URL           *urlpkg.URL
	Header        Header
	Body          io.ReadCloser
	ContentLength int64

	bodyData []byte
}

type Response struct {
	Status        string
	StatusCode    int
	Proto         string
	ProtoMajor    int
	ProtoMinor    int
	Header        Header
	Body          io.ReadCloser
	ContentLength int64
	Request       *Request
}

type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}

type Transport struct {
	TLSClientConfig *tls.Config
}

type Client struct {
	Transport RoundTripper
}

var DefaultTransport RoundTripper = &Transport{}
var DefaultClient = &Client{}
var NoBody io.ReadCloser = noBodyReader{}

func (header Header) Add(key string, value string) {
	storedKey := headerStoredKey(header, key)
	header[storedKey] = append(header[storedKey], value)
}

func (header Header) Set(key string, value string) {
	storedKey := headerStoredKey(header, key)
	header[storedKey] = []string{value}
}

func (header Header) Get(key string) string {
	values := header.Values(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (header Header) Values(key string) []string {
	if header == nil {
		return nil
	}

	if values, ok := header[key]; ok {
		return values
	}

	for existingKey := range header {
		if asciiEqualFold(existingKey, key) {
			return header[existingKey]
		}
	}

	return nil
}

func (header Header) Del(key string) {
	if header == nil {
		return
	}

	if _, ok := header[key]; ok {
		delete(header, key)
	}

	keys := make([]string, 0, len(header))
	for existingKey := range header {
		if asciiEqualFold(existingKey, key) {
			keys = append(keys, existingKey)
		}
	}
	for index := 0; index < len(keys); index++ {
		delete(header, keys[index])
	}
}

func (header Header) Clone() Header {
	if header == nil {
		return nil
	}

	cloned := make(Header)
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	for index := 0; index < len(keys); index++ {
		key := keys[index]
		values := header[key]
		copied := make([]string, len(values))
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			copied[valueIndex] = values[valueIndex]
		}
		cloned[key] = copied
	}

	return cloned
}

func NewRequest(method string, rawURL string, body io.Reader) (*Request, error) {
	if method == "" {
		method = MethodGet
	}
	method = normalizeMethod(method)

	parsedURL, err := urlpkg.Parse(rawURL)
	if err != nil {
		return nil, &urlpkg.Error{
			Op:  httpOperationName(method),
			URL: rawURL,
			Err: err,
		}
	}

	request := &Request{
		Method: method,
		URL:    parsedURL,
		Header: make(Header),
		Body:   NoBody,
	}

	if body != nil {
		data, readErr := io.ReadAll(body)
		if readErr != nil {
			return nil, &urlpkg.Error{
				Op:  httpOperationName(method),
				URL: rawURL,
				Err: readErr,
			}
		}

		request.bodyData = data
		request.ContentLength = int64(len(data))
		request.Body = newMemoryBody(data)
	}

	return request, nil
}

func Get(rawURL string) (*Response, error) {
	return DefaultClient.Get(rawURL)
}

func Head(rawURL string) (*Response, error) {
	return DefaultClient.Head(rawURL)
}

func Post(rawURL string, contentType string, body io.Reader) (*Response, error) {
	return DefaultClient.Post(rawURL, contentType, body)
}

func PostForm(rawURL string, data urlpkg.Values) (*Response, error) {
	return DefaultClient.PostForm(rawURL, data)
}

func (client *Client) Get(rawURL string) (*Response, error) {
	request, err := NewRequest(MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(request)
}

func (client *Client) Head(rawURL string) (*Response, error) {
	request, err := NewRequest(MethodHead, rawURL, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(request)
}

func (client *Client) Post(rawURL string, contentType string, body io.Reader) (*Response, error) {
	request, err := NewRequest(MethodPost, rawURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	return client.Do(request)
}

func (client *Client) PostForm(rawURL string, data urlpkg.Values) (*Response, error) {
	encoded := ""
	if data != nil {
		encoded = data.Encode()
	}

	return client.Post(rawURL, "application/x-www-form-urlencoded", strings.NewReader(encoded))
}

func (client *Client) Do(request *Request) (*Response, error) {
	if client == nil {
		client = DefaultClient
	}

	transport := client.Transport
	if transport == nil {
		transport = DefaultTransport
	}
	if transport == nil {
		return nil, &urlpkg.Error{
			Op:  "request",
			URL: "",
			Err: errors.New("nil Transport"),
		}
	}

	response, err := transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (transport *Transport) RoundTrip(request *Request) (*Response, error) {
	if transport == nil {
		transport = &Transport{}
	}

	if request == nil || request.URL == nil {
		return nil, &urlpkg.Error{
			Op:  "request",
			URL: "",
			Err: errors.New("nil Request"),
		}
	}

	rawURL := request.URL.String()
	scheme := strings.ToLower(request.URL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, &urlpkg.Error{
			Op:  httpOperationName(request.Method),
			URL: rawURL,
			Err: errors.New("unsupported protocol scheme " + quote(scheme)),
		}
	}
	if request.URL.Host == "" {
		return nil, &urlpkg.Error{
			Op:  httpOperationName(request.Method),
			URL: rawURL,
			Err: errors.New("missing host"),
		}
	}
	method := normalizeMethod(request.Method)
	if !supportsTransferMethod(method) {
		return nil, &urlpkg.Error{
			Op:  httpOperationName(method),
			URL: rawURL,
			Err: errors.New("unsupported method " + quote(method)),
		}
	}

	var (
		response *Response
		err      error
	)
	switch scheme {
	case "http":
		response, err = doHTTPObj(request, rawURL, method)
	case "https":
		response, err = transport.doHTTPS(request, method)
	}
	if err != nil {
		return nil, &urlpkg.Error{
			Op:  httpOperationName(method),
			URL: rawURL,
			Err: err,
		}
	}
	return response, nil
}

func (transport *Transport) doHTTPS(request *Request, method string) (*Response, error) {
	serverName, dialAddress, requestHost, requestPath, err := resolveHTTPSRequestTarget(request.URL)
	if err != nil {
		return nil, err
	}

	config := cloneTLSConfig(transport.TLSClientConfig)
	if config.ServerName == "" {
		config.ServerName = serverName
	}

	conn, err := tls.DialWithDialer(new(net.Dialer), "tcp", dialAddress, config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := writeHTTPSRequest(conn, request, method, requestHost, requestPath); err != nil {
		return nil, err
	}

	return readHTTPSResponse(bufio.NewReader(conn), method, request)
}

func StatusText(code int) string {
	switch code {
	case StatusCreated:
		return "Created"
	case StatusAccepted:
		return "Accepted"
	case StatusNoContent:
		return "No Content"
	case StatusOK:
		return "OK"
	case StatusMovedPermanently:
		return "Moved Permanently"
	case StatusFound:
		return "Found"
	case StatusSeeOther:
		return "See Other"
	case StatusTemporaryRedirect:
		return "Temporary Redirect"
	case StatusPermanentRedirect:
		return "Permanent Redirect"
	case StatusBadRequest:
		return "Bad Request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusForbidden:
		return "Forbidden"
	case StatusNotFound:
		return "Not Found"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	case StatusConflict:
		return "Conflict"
	case StatusGone:
		return "Gone"
	case StatusRequestTimeout:
		return "Request Timeout"
	case StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case StatusInternalServerError:
		return "Internal Server Error"
	case StatusBadGateway:
		return "Bad Gateway"
	case StatusServiceUnavailable:
		return "Service Unavailable"
	}

	return ""
}

type noBodyReader struct{}

func (noBodyReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (noBodyReader) Close() error {
	return nil
}

type memoryBody struct {
	reader *bytes.Reader
	closed bool
}

func newMemoryBody(data []byte) *memoryBody {
	return &memoryBody{reader: bytes.NewReader(data)}
}

func (body *memoryBody) Read(buffer []byte) (int, error) {
	if body == nil || body.closed {
		return 0, errors.New("http: read on closed body")
	}

	return body.reader.Read(buffer)
}

func (body *memoryBody) Close() error {
	if body != nil {
		body.closed = true
	}
	return nil
}

func responseFromTransfer(transfer kos.HTTPTransfer, method string, request *Request) *Response {
	statusLine, header := parseHeaderBlock(transfer.HeaderString())
	statusCode, status, proto, protoMajor, protoMinor := parseStatusLine(statusLine, int(transfer.Status()))

	contentLength := int64(-1)
	if transfer.Flags().Has(kos.HTTPFlagContentLength) {
		contentLength = int64(transfer.ContentLength())
	}

	bodyData := []byte{}
	if method != MethodHead {
		bodyData = transfer.ContentBytes()
		if contentLength < 0 {
			contentLength = int64(len(bodyData))
		}
	}

	return &Response{
		Status:        status,
		StatusCode:    statusCode,
		Proto:         proto,
		ProtoMajor:    protoMajor,
		ProtoMinor:    protoMinor,
		Header:        header,
		Body:          newMemoryBody(bodyData),
		ContentLength: contentLength,
		Request:       request,
	}
}

func doHTTPObj(request *Request, rawURL string, method string) (*Response, error) {
	var (
		http     kos.HTTP
		transfer kos.HTTPTransfer
		ok       bool
	)

	http, ok = kos.LoadHTTP()
	if !ok {
		return nil, errors.New("http.obj unavailable")
	}
	if !http.Ready() {
		return nil, errors.New("http.obj transfer unavailable")
	}

	addHeader := headerLines(request.Header, method == MethodPost)

	switch method {
	case MethodGet:
		transfer, ok = http.Get(rawURL, 0, 0, addHeader)
	case MethodHead:
		transfer, ok = http.Head(rawURL, 0, 0, addHeader)
	case MethodPost:
		contentType := request.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		transfer, ok = http.Post(rawURL, 0, 0, addHeader, contentType, uint32(len(request.bodyData)))
		if ok && len(request.bodyData) > 0 {
			sent := http.Send(transfer, request.bodyData)
			if sent != len(request.bodyData) {
				http.Free(transfer)
				return nil, errors.New("request body send failed")
			}
		}
	}
	if !ok {
		return nil, errors.New("request start failed")
	}

	for http.Receive(transfer) != 0 {
	}

	flags := transfer.Flags()
	if err := transferError(flags); err != nil {
		http.Free(transfer)
		return nil, err
	}

	response := responseFromTransfer(transfer, method, request)
	http.Free(transfer)
	return response, nil
}

func resolveHTTPSRequestTarget(target *urlpkg.URL) (serverName string, dialAddress string, requestHost string, requestPath string, err error) {
	if target == nil {
		return "", "", "", "", errors.New("missing URL")
	}
	requestHost = target.Host
	if requestHost == "" {
		return "", "", "", "", errors.New("missing host")
	}

	requestPath = target.EscapedPath()
	if requestPath == "" {
		requestPath = "/"
	}
	if target.RawQuery != "" {
		requestPath += "?" + target.RawQuery
	}

	if host, port, splitErr := net.SplitHostPort(requestHost); splitErr == nil {
		return host, net.JoinHostPort(host, port), requestHost, requestPath, nil
	}

	serverName = stripBracketHost(requestHost)
	return serverName, net.JoinHostPort(serverName, "443"), requestHost, requestPath, nil
}

func writeHTTPSRequest(writer io.Writer, request *Request, method string, requestHost string, requestPath string) error {
	var builder strings.Builder
	builder.WriteString(method)
	builder.WriteString(" ")
	builder.WriteString(requestPath)
	builder.WriteString(" HTTP/1.1\r\n")
	builder.WriteString("Host: ")
	builder.WriteString(requestHost)
	builder.WriteString("\r\n")

	userAgent := request.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "Go-http-client/1.1"
	}
	builder.WriteString("User-Agent: ")
	builder.WriteString(userAgent)
	builder.WriteString("\r\n")
	builder.WriteString("Connection: close\r\n")

	if method == MethodPost {
		contentType := request.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		builder.WriteString("Content-Type: ")
		builder.WriteString(contentType)
		builder.WriteString("\r\n")
		builder.WriteString("Content-Length: ")
		builder.WriteString(strconv.Itoa(len(request.bodyData)))
		builder.WriteString("\r\n")
	}

	addHeader := headerLines(request.Header, method == MethodPost)
	if addHeader != "" {
		builder.WriteString(addHeader)
	}
	builder.WriteString("\r\n")

	if _, err := writer.Write([]byte(builder.String())); err != nil {
		return err
	}
	if len(request.bodyData) > 0 {
		if _, err := writer.Write(request.bodyData); err != nil {
			return err
		}
	}

	return nil
}

func readHTTPSResponse(reader *bufio.Reader, method string, request *Request) (*Response, error) {
	statusLine, err := readHTTPLine(reader)
	if err != nil {
		return nil, err
	}
	header, err := readHTTPHeaders(reader)
	if err != nil {
		return nil, err
	}

	statusCode, status, proto, protoMajor, protoMinor := parseStatusLine(statusLine, 0)
	contentLength := parsedContentLength(header)
	bodyData := []byte{}

	if !responseHasNoBody(method, statusCode) {
		switch {
		case hasChunkedTransferEncoding(header):
			bodyData, err = readChunkedBody(reader)
		case contentLength >= 0:
			bodyData, err = io.ReadAll(io.LimitReader(reader, contentLength))
		default:
			bodyData, err = io.ReadAll(reader)
		}
		if err != nil {
			return nil, err
		}
		if contentLength < 0 {
			contentLength = int64(len(bodyData))
		}
	}

	return &Response{
		Status:        status,
		StatusCode:    statusCode,
		Proto:         proto,
		ProtoMajor:    protoMajor,
		ProtoMinor:    protoMinor,
		Header:        header,
		Body:          newMemoryBody(bodyData),
		ContentLength: contentLength,
		Request:       request,
	}, nil
}

func readHTTPHeaders(reader *bufio.Reader) (Header, error) {
	header := make(Header)
	for {
		line, err := readHTTPLine(reader)
		if err != nil {
			return nil, err
		}
		if line == "" {
			return header, nil
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
}

func readHTTPLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSuffix(line, "\n")
	return trimCR(line), nil
}

func responseHasNoBody(method string, statusCode int) bool {
	if method == MethodHead {
		return true
	}
	if statusCode >= 100 && statusCode < 200 {
		return true
	}
	return statusCode == 204 || statusCode == 304
}

func parsedContentLength(header Header) int64 {
	value := strings.TrimSpace(header.Get("Content-Length"))
	if value == "" {
		return -1
	}

	length, err := strconv.ParseInt(value, 10, 64)
	if err != nil || length < 0 {
		return -1
	}
	return length
}

func hasChunkedTransferEncoding(header Header) bool {
	values := header.Values("Transfer-Encoding")
	for valueIndex := 0; valueIndex < len(values); valueIndex++ {
		parts := strings.Split(values[valueIndex], ",")
		for partIndex := 0; partIndex < len(parts); partIndex++ {
			if asciiEqualFold(strings.TrimSpace(parts[partIndex]), "chunked") {
				return true
			}
		}
	}
	return false
}

func readChunkedBody(reader *bufio.Reader) ([]byte, error) {
	var body bytes.Buffer

	for {
		line, err := readHTTPLine(reader)
		if err != nil {
			return nil, err
		}

		sizeText := line
		if semicolon := strings.Index(sizeText, ";"); semicolon >= 0 {
			sizeText = sizeText[:semicolon]
		}
		sizeText = strings.TrimSpace(sizeText)
		if sizeText == "" {
			return nil, errors.New("malformed chunk size")
		}

		size, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil || size < 0 {
			return nil, errors.New("malformed chunk size")
		}
		if size == 0 {
			for {
				line, err = readHTTPLine(reader)
				if err != nil {
					return nil, err
				}
				if line == "" {
					return body.Bytes(), nil
				}
			}
		}

		chunk := make([]byte, int(size))
		if _, err := io.ReadFull(reader, chunk); err != nil {
			return nil, err
		}
		if _, err := body.Write(chunk); err != nil {
			return nil, err
		}
		if err := consumeHTTPLineEnd(reader); err != nil {
			return nil, err
		}
	}
}

func consumeHTTPLineEnd(reader *bufio.Reader) error {
	first, err := reader.ReadByte()
	if err != nil {
		return err
	}

	switch first {
	case '\n':
		return nil
	case '\r':
		second, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if second == '\n' {
			return nil
		}
		return errors.New("malformed chunk terminator")
	default:
		return errors.New("malformed chunk terminator")
	}
}

func stripBracketHost(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}
	return host
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return &tls.Config{}
	}
	return config.Clone()
}

func parseHeaderBlock(block string) (string, Header) {
	header := make(Header)
	lines := splitLines(block)
	if len(lines) == 0 {
		return "", header
	}

	for index := 1; index < len(lines); index++ {
		line := trimCR(lines[index])
		if line == "" {
			continue
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}

	return trimCR(lines[0]), header
}

func parseStatusLine(statusLine string, fallback int) (statusCode int, status string, proto string, protoMajor int, protoMinor int) {
	statusCode = fallback
	if statusLine == "" {
		if statusCode > 0 {
			status = strconv.Itoa(statusCode)
		}
		return
	}

	parts := strings.Fields(statusLine)
	if len(parts) >= 1 {
		proto = parts[0]
		protoMajor, protoMinor = parseProto(proto)
	}
	if len(parts) >= 2 {
		parsedCode, err := strconv.Atoi(parts[1])
		if err == nil {
			statusCode = parsedCode
		}
	}

	status = strings.TrimSpace(statusLine)
	if proto != "" && strings.HasPrefix(status, proto+" ") {
		status = strings.TrimSpace(status[len(proto):])
	}
	if status == "" && statusCode > 0 {
		text := StatusText(statusCode)
		if text != "" {
			status = strconv.Itoa(statusCode) + " " + text
		} else {
			status = strconv.Itoa(statusCode)
		}
	}

	return
}

func parseProto(proto string) (major int, minor int) {
	if !strings.HasPrefix(proto, "HTTP/") {
		return 0, 0
	}

	version := proto[5:]
	dot := strings.Index(version, ".")
	if dot < 0 {
		return 0, 0
	}

	majorValue, majorErr := strconv.Atoi(version[:dot])
	minorValue, minorErr := strconv.Atoi(version[dot+1:])
	if majorErr != nil || minorErr != nil {
		return 0, 0
	}

	return majorValue, minorValue
}

func transferError(flags kos.HTTPFlags) error {
	switch {
	case flags.Has(kos.HTTPFlagInvalidHeader):
		return errors.New("invalid header")
	case flags.Has(kos.HTTPFlagNoRAM):
		return errors.New("out of memory")
	case flags.Has(kos.HTTPFlagSocketError):
		return errors.New("socket error")
	case flags.Has(kos.HTTPFlagTimeoutError):
		return errors.New("timeout")
	case flags.Has(kos.HTTPFlagTransferFailed):
		return errors.New("transfer failed")
	case flags.Has(kos.HTTPFlagNeedMoreSpace):
		return errors.New("need more space")
	}

	return nil
}

func headerLines(header Header, excludePostManaged bool) string {
	if len(header) == 0 {
		return ""
	}

	keys := make([]string, 0, len(header))
	for key := range header {
		if skipHeader(key, excludePostManaged) {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return ""
	}

	sortStrings(keys)

	var builder strings.Builder
	for index := 0; index < len(keys); index++ {
		values := header[keys[index]]
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			builder.WriteString(keys[index])
			builder.WriteString(": ")
			builder.WriteString(values[valueIndex])
			builder.WriteString("\r\n")
		}
	}

	return builder.String()
}

func skipHeader(key string, excludePostManaged bool) bool {
	if asciiEqualFold(key, "Host") || asciiEqualFold(key, "Connection") || asciiEqualFold(key, "User-Agent") {
		return true
	}
	if excludePostManaged && (asciiEqualFold(key, "Content-Type") || asciiEqualFold(key, "Content-Length")) {
		return true
	}

	return false
}

func headerStoredKey(header Header, key string) string {
	if header == nil {
		return key
	}
	if _, ok := header[key]; ok {
		return key
	}

	for existingKey := range header {
		if asciiEqualFold(existingKey, key) {
			return existingKey
		}
	}

	return key
}

func normalizeMethod(method string) string {
	if method == "" {
		return MethodGet
	}

	buffer := []byte(method)
	for index := 0; index < len(buffer); index++ {
		if buffer[index] >= 'a' && buffer[index] <= 'z' {
			buffer[index] -= 'a' - 'A'
		}
	}

	return string(buffer)
}

func httpOperationName(method string) string {
	switch normalizeMethod(method) {
	case MethodGet:
		return "Get"
	case MethodHead:
		return "Head"
	case MethodPost:
		return "Post"
	default:
		return titleHTTPMethod(method)
	}
}

func supportsTransferMethod(method string) bool {
	switch normalizeMethod(method) {
	case MethodGet, MethodHead, MethodPost:
		return true
	default:
		return false
	}
}

func splitLines(value string) []string {
	if value == "" {
		return []string{}
	}

	return strings.Split(value, "\n")
}

func trimCR(value string) string {
	if strings.HasSuffix(value, "\r") {
		return value[:len(value)-1]
	}

	return value
}

func quote(value string) string {
	return `"` + value + `"`
}

func titleHTTPMethod(method string) string {
	value := normalizeMethod(method)
	if value == "" {
		return "Get"
	}

	buffer := []byte(value)
	for index := 1; index < len(buffer); index++ {
		if buffer[index] >= 'A' && buffer[index] <= 'Z' {
			buffer[index] += 'a' - 'A'
		}
	}

	return string(buffer)
}

func asciiEqualFold(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := 0; index < len(left); index++ {
		if asciiLower(left[index]) != asciiLower(right[index]) {
			return false
		}
	}

	return true
}

func asciiLower(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}

	return value
}

func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		current := values[index]
		position := index - 1
		for position >= 0 && stringLess(current, values[position]) {
			values[position+1] = values[position]
			position--
		}
		values[position+1] = current
	}
}

func stringLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}

	for index := 0; index < limit; index++ {
		if left[index] == right[index] {
			continue
		}
		return left[index] < right[index]
	}

	return len(left) < len(right)
}
