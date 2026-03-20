package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"kos"
	"net"
	"net/url"
)

const (
	httpsDemoTitle    = "KolibriOS HTTPS Demo"
	defaultHTTPSURL   = "https://example.com/"
	bodyPreviewLength = 512
)

var httpsIPCBuffer [4096]byte
var httpsStage = "startup"

func main() {
	setHTTPSStage("startup")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	console, ok := kos.OpenConsole(httpsDemoTitle)
	if !ok {
		kos.DebugString("https demo: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = fmt.Printf("PANIC at stage %q: %T %v\n", httpsStage, recovered, recovered)
			waitForExit(console)
			os.Exit(2)
		}
	}()

	setHTTPSStage("register IPC buffer")
	kos.RegisterIPCBuffer(httpsIPCBuffer[:])
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC | kos.EventMaskNetwork)

	setHTTPSStage("resolve arguments")
	target := defaultHTTPSURL
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		target = os.Args[1]
	}

	_, _ = fmt.Printf("HTTPS target: %s\n", target)
	if tid, ok := kos.CurrentThreadID(); ok {
		_, _ = fmt.Printf("Current thread id: 0x%X\n", tid)
	}
	if slot, ok := kos.CurrentThreadSlotIndex(); ok {
		_, _ = fmt.Printf("Current thread slot: %d\n", slot)
	}
	setHTTPSStage("fetch HTTPS")
	result, err := fetchHTTPS(target)
	if err != nil {
		_, _ = fmt.Printf("HTTPS request failed: %v\n", err)
		waitForExit(console)
		os.Exit(1)
		return
	}

	_, _ = fmt.Printf("Connected to %s using %s / %s\n", result.ServerName, tlsVersion(result.Version), tls.CipherSuiteName(result.CipherSuite))
	_, _ = fmt.Printf("Status: %s\n", result.StatusLine)
	for _, header := range result.Headers {
		_, _ = fmt.Println(header)
	}
	if result.BodyPreview != "" {
		_, _ = fmt.Println("")
		_, _ = fmt.Printf("Body preview (%d bytes max):\n", bodyPreviewLength)
		_, _ = fmt.Println(result.BodyPreview)
	}

	waitForExit(console)
	os.Exit(0)
}

type fetchResult struct {
	ServerName  string
	Version     uint16
	CipherSuite uint16
	StatusLine  string
	Headers     []string
	BodyPreview string
}

func fetchHTTPS(rawTarget string) (*fetchResult, error) {
	setHTTPSStage("normalize URL")
	target, err := normalizeHTTPSURL(rawTarget)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("resolve target")
	serverName, dialAddress, requestHost, requestPath, err := resolveTarget(target)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("TCP dial")
	rawConn, err := net.Dial("tcp", dialAddress)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("TLS client")
	conn := tls.Client(rawConn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	})
	defer conn.Close()

	setHTTPSStage("TLS handshake")
	if err := conn.Handshake(); err != nil {
		return nil, err
	}

	setHTTPSStage("write HTTP request")
	request := "GET " + requestPath + " HTTP/1.1\r\n" +
		"Host: " + requestHost + "\r\n" +
		"User-Agent: kolibrios-gccgo-sdk/https-example\r\n" +
		"Connection: close\r\n" +
		"Accept: */*\r\n\r\n"

	if _, err := conn.Write([]byte(request)); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	setHTTPSStage("read status line")
	statusLine, err := readHeaderLine(reader)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("read headers")
	headers, err := readHeaders(reader)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("read body preview")
	bodyPreview, err := readBodyPreview(reader, bodyPreviewLength)
	if err != nil {
		return nil, err
	}

	setHTTPSStage("collect TLS state")
	state := conn.ConnectionState()
	return &fetchResult{
		ServerName:  serverName,
		Version:     state.Version,
		CipherSuite: state.CipherSuite,
		StatusLine:  statusLine,
		Headers:     headers,
		BodyPreview: bodyPreview,
	}, nil
}

func normalizeHTTPSURL(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultHTTPSURL
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	target, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "https" {
		return nil, fmt.Errorf("https demo only supports https:// URLs")
	}
	if target.Host == "" {
		return nil, fmt.Errorf("missing host in URL")
	}

	return target, nil
}

func resolveTarget(target *url.URL) (serverName string, dialAddress string, requestHost string, requestPath string, err error) {
	requestHost = target.Host
	requestPath = target.EscapedPath()
	if requestPath == "" {
		requestPath = "/"
	}
	if target.RawQuery != "" {
		requestPath += "?" + target.RawQuery
	}

	serverName = requestHost
	dialAddress = requestHost
	if host, port, splitErr := net.SplitHostPort(requestHost); splitErr == nil {
		serverName = host
		dialAddress = net.JoinHostPort(host, port)
		return serverName, dialAddress, requestHost, requestPath, nil
	}

	serverName = requestHost
	dialAddress = net.JoinHostPort(requestHost, "443")
	return serverName, dialAddress, requestHost, requestPath, nil
}

func readHeaderLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return trimLine(line), nil
}

func readHeaders(reader *bufio.Reader) ([]string, error) {
	var headers []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = trimLine(line)
		if line == "" {
			return headers, nil
		}
		headers = append(headers, line)
	}
}

func readBodyPreview(reader *bufio.Reader, limit int) (string, error) {
	buffer := make([]byte, limit)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(string(buffer[:n])), nil
}

func setHTTPSStage(stage string) {
	httpsStage = stage
	kos.DebugString("https demo stage: " + stage + "\n")
}

func trimLine(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}

func tlsVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return fmt.Sprintf("0x%04x", version)
	}
}

func waitForExit(console kos.Console) {
	if console.SupportsInput() {
		_, _ = fmt.Println("")
		_, _ = fmt.Println("Press any key to close.")
		console.Getch()
		return
	}

	_, _ = fmt.Println("")
	_, _ = fmt.Println("Input export missing, closing in five seconds.")
	kos.SleepSeconds(5)
}
