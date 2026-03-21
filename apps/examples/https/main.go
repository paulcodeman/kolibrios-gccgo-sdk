package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	urlpkg "net/url"
	"os"
	"strings"

	"kos"
)

const (
	httpsDemoTitle    = "KolibriOS HTTPS Demo"
	defaultHTTPSURL   = "https://example.com/"
	bodyPreviewLength = 512
	localCABundlePath = "./ca-bundle.pem"
)

var httpsIPCBuffer [4096]byte

func main() {
	console, ok := kos.OpenConsole(httpsDemoTitle)
	if !ok {
		kos.DebugString("https demo: failed to open /sys/lib/console.obj\n")
		os.Exit(1)
		return
	}

	kos.RegisterIPCBuffer(httpsIPCBuffer[:])
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC | kos.EventMaskNetwork)

	rootPath, ok := configureLocalCABundle()
	if ok {
		_, _ = fmt.Printf("TLS roots: %s\n", rootPath)
	}
	rootCAs, rootErr := loadRootPool(rootPath)
	if rootErr != nil {
		_, _ = fmt.Printf("TLS root load failed: %v\n", rootErr)
	}

	target := defaultHTTPSURL
	insecure := false
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		for index := 1; index < len(os.Args); index++ {
			argument := strings.TrimSpace(os.Args[index])
			if argument == "" {
				continue
			}
			if argument == "--insecure" {
				insecure = true
				continue
			}
			target = argument
		}
	}

	_, _ = fmt.Printf("HTTPS target: %s\n", target)
	if insecure {
		_, _ = fmt.Println("TLS verification: disabled by --insecure")
	}
	result, err := fetchHTTPS(target, insecure, rootCAs)
	if err != nil {
		_, _ = fmt.Printf("HTTPS request failed: %v\n", err)
		printTLSDiagnostics(target, rootPath, rootCAs)
		_, _ = fmt.Println("Hint: set SSL_CERT_FILE to a PEM bundle if certificate verification cannot find roots.")
		_, _ = fmt.Printf("Hint: this example also auto-loads %s when present.\n", localCABundlePath)
		_, _ = fmt.Println("Hint: use --insecure only for testing when you intentionally want to skip verification.")
		waitForExit(console)
		os.Exit(1)
		return
	}

	_, _ = fmt.Printf("Status: %s\n", result.StatusLine)
	for index := 0; index < len(result.Headers); index++ {
		_, _ = fmt.Println(result.Headers[index])
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
	StatusLine  string
	Headers     []string
	BodyPreview string
}

func fetchHTTPS(rawTarget string, insecure bool, rootCAs *x509.CertPool) (*fetchResult, error) {
	request, err := http.NewRequest(http.MethodGet, normalizeHTTPSURL(rawTarget), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "*/*")

	client := http.DefaultClient
	if insecure || rootCAs != nil {
		config := &tls.Config{
			InsecureSkipVerify: insecure,
			RootCAs:            rootCAs,
		}
		client = &http.Client{
			Transport: &http.Transport{TLSClientConfig: config},
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	bodyPreview, err := readBodyPreview(response.Body, bodyPreviewLength)
	if err != nil {
		return nil, err
	}

	return &fetchResult{
		StatusLine:  response.Status,
		Headers:     flattenHeaders(response.Header),
		BodyPreview: bodyPreview,
	}, nil
}

func normalizeHTTPSURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultHTTPSURL
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	return value
}

func configureLocalCABundle() (string, bool) {
	if os.Getenv("SSL_CERT_FILE") != "" {
		return os.Getenv("SSL_CERT_FILE"), true
	}

	if _, err := os.Stat(localCABundlePath); err != nil {
		return "", false
	}
	if err := os.Setenv("SSL_CERT_FILE", localCABundlePath); err != nil {
		return "", false
	}
	return localCABundlePath, true
}

func loadRootPool(path string) (*x509.CertPool, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no certificates parsed from %s", path)
	}
	return pool, nil
}

type tlsDiagnostics struct {
	ServerName       string
	DialAddress      string
	PeerCertificates []*x509.Certificate
	VerifyErr        error
}

func printTLSDiagnostics(rawTarget string, rootPath string, rootCAs *x509.CertPool) {
	diagnostics, err := collectTLSDiagnostics(rawTarget, rootCAs)
	if err != nil {
		_, _ = fmt.Printf("TLS diagnostics failed: %v\n", err)
		return
	}

	_, _ = fmt.Printf("TLS diagnostics: server=%s addr=%s peer_certs=%d\n",
		diagnostics.ServerName, diagnostics.DialAddress, len(diagnostics.PeerCertificates))
	for index := 0; index < len(diagnostics.PeerCertificates); index++ {
		cert := diagnostics.PeerCertificates[index]
		_, _ = fmt.Printf("  cert[%d] subject=%s\n", index, cert.Subject.String())
		_, _ = fmt.Printf("  cert[%d] issuer=%s\n", index, cert.Issuer.String())
	}
	if diagnostics.VerifyErr != nil {
		_, _ = fmt.Printf("TLS manual verify failed: %v\n", diagnostics.VerifyErr)
	} else {
		_, _ = fmt.Println("TLS manual verify: ok")
	}
	printRootBundleDiagnostics(rootPath, diagnostics)
}

func collectTLSDiagnostics(rawTarget string, rootCAs *x509.CertPool) (*tlsDiagnostics, error) {
	parsedURL, err := urlpkg.Parse(normalizeHTTPSURL(rawTarget))
	if err != nil {
		return nil, err
	}

	serverName, dialAddress, err := resolveTLSDiagnosticTarget(parsedURL)
	if err != nil {
		return nil, err
	}

	conn, err := tls.Dial("tcp", dialAddress, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	diagnostics := &tlsDiagnostics{
		ServerName:       serverName,
		DialAddress:      dialAddress,
		PeerCertificates: state.PeerCertificates,
	}
	if len(state.PeerCertificates) == 0 {
		return diagnostics, nil
	}

	options := x509.VerifyOptions{
		Roots:         rootCAs,
		DNSName:       serverName,
		Intermediates: x509.NewCertPool(),
	}
	for index := 1; index < len(state.PeerCertificates); index++ {
		options.Intermediates.AddCert(state.PeerCertificates[index])
	}
	_, diagnostics.VerifyErr = state.PeerCertificates[0].Verify(options)
	return diagnostics, nil
}

func resolveTLSDiagnosticTarget(target *urlpkg.URL) (serverName string, dialAddress string, err error) {
	if target == nil {
		return "", "", fmt.Errorf("missing URL")
	}
	host := strings.TrimSpace(target.Host)
	if host == "" {
		return "", "", fmt.Errorf("missing host")
	}
	if resolvedHost, port, splitErr := net.SplitHostPort(host); splitErr == nil {
		return resolvedHost, net.JoinHostPort(resolvedHost, port), nil
	}
	serverName = host
	if len(serverName) >= 2 && serverName[0] == '[' && serverName[len(serverName)-1] == ']' {
		serverName = serverName[1 : len(serverName)-1]
	}
	return serverName, net.JoinHostPort(serverName, "443"), nil
}

func printRootBundleDiagnostics(rootPath string, diagnostics *tlsDiagnostics) {
	if rootPath == "" || diagnostics == nil || len(diagnostics.PeerCertificates) == 0 {
		return
	}

	data, err := os.ReadFile(rootPath)
	if err != nil {
		_, _ = fmt.Printf("TLS root bundle read failed: %v\n", err)
		return
	}

	totalBlocks := 0
	parsedCerts := 0
	matchCount := 0
	lastPeer := diagnostics.PeerCertificates[len(diagnostics.PeerCertificates)-1]
	issuerText := lastPeer.Issuer.String()
	raw := data

	for len(raw) > 0 {
		var block *pem.Block
		block, raw = pem.Decode(raw)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		totalBlocks++
		root, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			continue
		}
		parsedCerts++
		if root.Subject.String() != issuerText {
			continue
		}

		matchCount++
		_, _ = fmt.Printf("TLS root candidate[%d] subject=%s\n", matchCount-1, root.Subject.String())
		_, _ = fmt.Printf("TLS root candidate[%d] issuer=%s\n", matchCount-1, root.Issuer.String())
		if err := lastPeer.CheckSignatureFrom(root); err != nil {
			_, _ = fmt.Printf("TLS root candidate[%d] signature check failed: %v\n", matchCount-1, err)
		} else {
			_, _ = fmt.Printf("TLS root candidate[%d] signature check: ok\n", matchCount-1)
		}
	}

	_, _ = fmt.Printf("TLS root bundle diagnostics: pem_blocks=%d parsed=%d matches_for_last_issuer=%d\n",
		totalBlocks, parsedCerts, matchCount)
}

func flattenHeaders(header http.Header) []string {
	lines := make([]string, 0, len(header))
	for key, values := range header {
		for index := 0; index < len(values); index++ {
			lines = append(lines, key+": "+values[index])
		}
	}
	return lines
}

func readBodyPreview(reader io.Reader, limit int) (string, error) {
	buffer := make([]byte, limit)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(string(buffer[:n])), nil
}

func waitForExit(console kos.Console) {
	_, _ = fmt.Print("\nPress any key to exit...")
	console.Getch()
}
