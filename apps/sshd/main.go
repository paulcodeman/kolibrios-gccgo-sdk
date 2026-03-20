package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"kos"
)

const (
	appTitle                 = "KolibriOS SSH Server"
	defaultPort              = 22
	defaultUser              = "kolibri"
	defaultHostKeyPath       = "/sys/sshd_host_key"
	defaultAuthorizedKeys    = "/sys/authorized_keys"
	serverVersion            = "SSH-2.0-KolibriOS-sshd"
	folderReadBatch          = 64
	defaultShellRoot         = "/"
	defaultShellHome         = "/sys"
)

var errUsage = errors.New("usage")
var sshdIPCBuffer [4096]byte
var stdinReader = bufio.NewReader(os.Stdin)

type options struct {
	port               int
	user               string
	password           string
	hostKeyPath        string
	authorizedKeysPath string
	once               bool
}

type authConfig struct {
	user       string
	password   string
	publicKeys map[string]ssh.PublicKey
}

type shellState struct {
	user string
	cwd  string
	out  io.Writer
	err  io.Writer
	pty  bool
}

type crlfWriter struct {
	dst io.Writer
}

func (writer crlfWriter) Write(data []byte) (int, error) {
	if writer.dst == nil || len(data) == 0 {
		return len(data), nil
	}

	buffer := make([]byte, 0, len(data)+16)
	for index := 0; index < len(data); index++ {
		value := data[index]
		if value == '\n' && (index == 0 || data[index-1] != '\r') {
			buffer = append(buffer, '\r')
		}
		buffer = append(buffer, value)
	}

	_, err := writer.dst.Write(buffer)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

type execRequest struct {
	Command string
}

type exitStatusMsg struct {
	Status uint32
}

func main() {
	runtime.GOMAXPROCS(4)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	console, ok := kos.OpenConsole(appTitle)
	if !ok {
		kos.DebugString("sshd: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	kos.RegisterIPCBuffer(sshdIPCBuffer[:])
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC | kos.EventMaskNetwork)
	if console.SupportsTitle() {
		console.SetTitle(appTitle)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "sshd panic: %T %v\n", recovered, recovered)
			waitForExit(console)
			os.Exit(2)
		}
	}()

	opts, err := parseArgs()
	if err != nil {
		if errors.Is(err, errUsage) {
			printUsage()
			waitForExit(console)
			os.Exit(2)
			return
		}
		fmt.Fprintf(os.Stderr, "sshd: %v\n", err)
		waitForExit(console)
		os.Exit(2)
		return
	}
	if !hasMeaningfulCLIArgs() {
		opts, err = promptOptions(console, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sshd: %v\n", err)
			waitForExit(console)
			os.Exit(1)
			return
		}
	}

	auth, err := loadAuthConfig(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sshd: %v\n", err)
		waitForExit(console)
		os.Exit(1)
		return
	}

	hostSigner, err := loadOrCreateHostSigner(opts.hostKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sshd: %v\n", err)
		waitForExit(console)
		os.Exit(1)
		return
	}

	var passwordCallback func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)
	if auth.password != "" {
		passwordCallback = auth.passwordCallback
	}
	var publicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)
	if len(auth.publicKeys) > 0 {
		publicKeyCallback = auth.publicKeyCallback
	}

	serverConfig := &ssh.ServerConfig{
		ServerVersion:     serverVersion,
		MaxAuthTries:      6,
		PasswordCallback:  passwordCallback,
		PublicKeyCallback: publicKeyCallback,
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			if err != nil {
				if errors.Is(err, ssh.ErrNoAuth) {
					return
				}
				fmt.Fprintf(os.Stderr, "Auth %s for %s from %s failed: %v\n", method, conn.User(), conn.RemoteAddr(), err)
				return
			}
			fmt.Printf("Auth %s for %s from %s succeeded\n", method, conn.User(), conn.RemoteAddr())
		},
	}
	serverConfig.AddHostKey(hostSigner)

	listenAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(opts.port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sshd: listen %s failed: %v\n", listenAddr, err)
		waitForExit(console)
		os.Exit(1)
		return
	}
	defer listener.Close()

	fmt.Printf("Listening on %s\n", listenAddr)
	fmt.Printf("User: %s\n", auth.user)
	fmt.Printf("Host key: %s\n", opts.hostKeyPath)
	fmt.Printf("Host fingerprint: %s\n", ssh.FingerprintSHA256(hostSigner.PublicKey()))
	printAuthSummary(opts, auth)
	if opts.once {
		fmt.Println("Mode: one connection, then exit")
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "sshd: accept failed: %v\n", err)
			os.Exit(1)
			return
		}

		if opts.once {
			serveConn(conn, serverConfig)
			return
		}

		go serveConn(conn, serverConfig)
	}
}

func parseArgs() (options, error) {
	opts := options{
		port:               defaultPort,
		user:               defaultUser,
		hostKeyPath:        defaultHostKeyPath,
		authorizedKeysPath: defaultAuthorizedKeys,
	}

	args := meaningfulCLIArgs()
	for len(args) > 0 {
		arg := args[0]
		if arg == "--" {
			args = args[1:]
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}

		switch {
		case arg == "-h" || arg == "--help":
			return opts, errUsage
		case arg == "-p" || strings.HasPrefix(arg, "-p="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			port, err := strconv.Atoi(value)
			if err != nil || port <= 0 || port > 65535 {
				return opts, fmt.Errorf("invalid port: %q", value)
			}
			opts.port = port
			args = rest
		case arg == "-user" || strings.HasPrefix(arg, "-user="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			value = strings.TrimSpace(value)
			if value == "" {
				return opts, fmt.Errorf("invalid user: %q", value)
			}
			opts.user = value
			args = rest
		case arg == "-password" || strings.HasPrefix(arg, "-password="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			opts.password = value
			args = rest
		case arg == "-host-key" || strings.HasPrefix(arg, "-host-key="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			if strings.TrimSpace(value) == "" {
				return opts, fmt.Errorf("invalid host key path")
			}
			opts.hostKeyPath = value
			args = rest
		case arg == "-authorized-keys" || strings.HasPrefix(arg, "-authorized-keys="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			opts.authorizedKeysPath = value
			args = rest
		case arg == "-once":
			opts.once = true
			args = args[1:]
		default:
			return opts, fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(args) != 0 {
		return opts, errUsage
	}
	return opts, nil
}

func meaningfulCLIArgs() []string {
	values := make([]string, 0, len(os.Args))
	for _, arg := range os.Args[1:] {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		values = append(values, arg)
	}
	return values
}

func hasMeaningfulCLIArgs() bool {
	return len(meaningfulCLIArgs()) > 0
}

func optionValue(arg string, args []string) (string, []string, error) {
	if eq := strings.Index(arg, "="); eq != -1 {
		if eq == len(arg)-1 {
			return "", args[1:], fmt.Errorf("missing value for %s", arg[:eq])
		}
		return arg[eq+1:], args[1:], nil
	}
	if len(args) < 2 {
		return "", args, fmt.Errorf("missing value for %s", arg)
	}
	return args[1], args[2:], nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: sshd [options]\n")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -p <port>                   Listen port (default 22)")
	fmt.Fprintf(os.Stderr, "  -user <name>                Accepted SSH username (default %s)\n", defaultUser)
	fmt.Fprintln(os.Stderr, "  -password <value>           Enable password authentication")
	fmt.Fprintf(os.Stderr, "  -host-key <path>            PEM private key path (default %s)\n", defaultHostKeyPath)
	fmt.Fprintf(os.Stderr, "  -authorized-keys <path>     authorized_keys path (default %s)\n", defaultAuthorizedKeys)
	fmt.Fprintln(os.Stderr, "  -once                       Accept one connection, then exit")
	fmt.Fprintln(os.Stderr, "  -h, --help                  Show this help text")
	fmt.Fprintln(os.Stderr, "\nNotes:")
	fmt.Fprintln(os.Stderr, "  If the host key file does not exist, sshd generates an Ed25519 key.")
	fmt.Fprintln(os.Stderr, "  At least one authentication method must be configured.")
	fmt.Fprintln(os.Stderr, "  Session channels support exec requests and a simple built-in shell.")
}

func promptOptions(console kos.Console, defaults options) (options, error) {
	opts := defaults
	fmt.Println("Interactive sshd setup")
	fmt.Println("Press Enter to accept the default shown in brackets.")
	fmt.Println("Enter - to clear a prefilled value.")

	for {
		portText, err := readLineWithDefault("Port", strconv.Itoa(opts.port))
		if err != nil {
			return opts, err
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port <= 0 || port > 65535 {
			fmt.Fprintf(os.Stderr, "sshd: invalid port: %q\n", portText)
			continue
		}
		opts.port = port

		user, err := readLineWithDefault("User", opts.user)
		if err != nil {
			return opts, err
		}
		user = strings.TrimSpace(user)
		if user == "" {
			fmt.Fprintln(os.Stderr, "sshd: user must not be empty")
			continue
		}
		opts.user = user

		password, err := readPasswordWithDefault(console, "Password (empty disables password auth)", opts.password)
		if err != nil {
			return opts, err
		}
		opts.password = password

		hostKeyPath, err := readLineWithDefault("Host key path", opts.hostKeyPath)
		if err != nil {
			return opts, err
		}
		hostKeyPath = strings.TrimSpace(hostKeyPath)
		if hostKeyPath == "" {
			fmt.Fprintln(os.Stderr, "sshd: host key path must not be empty")
			continue
		}
		opts.hostKeyPath = hostKeyPath

		keysPath, err := readLineWithDefault("authorized_keys path", opts.authorizedKeysPath)
		if err != nil {
			return opts, err
		}
		opts.authorizedKeysPath = strings.TrimSpace(keysPath)

		once, err := promptYesNoDefault(console, "One connection only?", opts.once)
		if err != nil {
			return opts, err
		}
		opts.once = once

		if _, err := loadAuthConfig(opts); err != nil {
			fmt.Fprintf(os.Stderr, "sshd: %v\n", err)
			fmt.Fprintln(os.Stderr, "sshd: provide a password or a valid authorized_keys path")
			continue
		}
		return opts, nil
	}
}

func printAuthSummary(opts options, auth *authConfig) {
	modes := []string{}
	if auth.password != "" {
		modes = append(modes, "password")
	}
	if len(auth.publicKeys) > 0 {
		modes = append(modes, fmt.Sprintf("publickey (%d keys from %s)", len(auth.publicKeys), opts.authorizedKeysPath))
	}
	fmt.Printf("Auth: %s\n", strings.Join(modes, ", "))
}

func loadAuthConfig(opts options) (*authConfig, error) {
	auth := &authConfig{
		user:       opts.user,
		password:   opts.password,
		publicKeys: map[string]ssh.PublicKey{},
	}

	if strings.TrimSpace(opts.authorizedKeysPath) != "" {
		keys, err := loadAuthorizedKeys(opts.authorizedKeysPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			auth.publicKeys = keys
		}
	}

	if auth.password == "" && len(auth.publicKeys) == 0 {
		return nil, fmt.Errorf("no authentication methods configured; set -password or provide %s", opts.authorizedKeysPath)
	}
	return auth, nil
}

func loadAuthorizedKeys(path string) (map[string]ssh.PublicKey, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	keys := map[string]ssh.PublicKey{}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
		}
		keys[string(key.Marshal())] = key
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return keys, nil
}

func (auth *authConfig) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	if auth == nil || auth.password == "" {
		return nil, fmt.Errorf("password authentication disabled")
	}
	if conn.User() != auth.user {
		return nil, fmt.Errorf("unknown user %q", conn.User())
	}
	if string(password) != auth.password {
		return nil, fmt.Errorf("password rejected for %q", conn.User())
	}
	return nil, nil
}

func (auth *authConfig) publicKeyCallback(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	if auth == nil || len(auth.publicKeys) == 0 {
		return nil, fmt.Errorf("public key authentication disabled")
	}
	if conn.User() != auth.user {
		return nil, fmt.Errorf("unknown user %q", conn.User())
	}
	if _, ok := auth.publicKeys[string(key.Marshal())]; !ok {
		return nil, fmt.Errorf("unknown public key for %q", conn.User())
	}
	return &ssh.Permissions{
		Extensions: map[string]string{
			"pubkey-fp": ssh.FingerprintSHA256(key),
		},
	}, nil
}

func loadOrCreateHostSigner(filePath string) (ssh.Signer, error) {
	data, err := os.ReadFile(filePath)
	if err == nil {
		signer, parseErr := ssh.ParsePrivateKey(data)
		if parseErr != nil {
			return nil, fmt.Errorf("parse host key %s: %w", filePath, parseErr)
		}
		return signer, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read host key %s: %w", filePath, err)
	}

	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("prepare host key folder: %w", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("encode host key: %w", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
	if err := os.WriteFile(filePath, pemData, 0600); err != nil {
		return nil, fmt.Errorf("write host key %s: %w", filePath, err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("build signer from generated host key: %w", err)
	}
	return signer, nil
}

func serveConn(raw net.Conn, config *ssh.ServerConfig) {
	remote := remoteAddress(raw)
	defer raw.Close()

	conn, chans, reqs, err := ssh.NewServerConn(raw, config)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return
		}
		fmt.Fprintf(os.Stderr, "sshd: handshake from %s failed: %v\n", remote, err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected: %s as %s\n", conn.RemoteAddr(), conn.User())
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "sshd: accept channel from %s failed: %v\n", conn.RemoteAddr(), err)
			continue
		}

		go handleSession(conn, channel, requests)
	}

	fmt.Printf("Disconnected: %s\n", conn.RemoteAddr())
}

func handleSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	stderr := channel.Stderr()
	state := shellState{
		user: conn.User(),
		cwd:  initialShellDir(),
		out:  channel,
		err:  stderr,
	}
	sessionStarted := false
	sessionPTY := false

	for req := range requests {
		switch req.Type {
		case "pty-req":
			sessionPTY = true
			_ = req.Reply(true, nil)
		case "env", "window-change":
			_ = req.Reply(true, nil)
		case "signal":
			_ = req.Reply(false, nil)
		case "shell":
			if sessionStarted {
				_ = req.Reply(false, nil)
				continue
			}
			sessionStarted = true
			state.pty = sessionPTY
			_ = req.Reply(true, nil)
			go runShellSession(channel, &state)
		case "exec":
			if sessionStarted {
				_ = req.Reply(false, nil)
				continue
			}
			var payload execRequest
			if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
				_ = req.Reply(false, nil)
				fmt.Fprintf(stderr, "invalid exec payload: %v\n", err)
				_ = sendExitStatus(channel, 1)
				_ = channel.Close()
				return
			}
			sessionStarted = true
			_ = req.Reply(true, nil)
			status := runExecCommand(&state, payload.Command)
			_ = sendExitStatus(channel, status)
			_ = channel.Close()
			return
		default:
			_ = req.Reply(false, nil)
		}
	}

	if !sessionStarted {
		_ = channel.Close()
	}
}

func runShellSession(channel ssh.Channel, state *shellState) {
	reader := bufio.NewReader(channel)
	if state.pty {
		state.out = crlfWriter{dst: state.out}
		state.err = crlfWriter{dst: state.err}
	}
	fmt.Fprintf(state.out, "KolibriOS SSH shell\n")
	fmt.Fprintf(state.out, "Type 'help' for available commands.\n")

	for {
		if _, err := fmt.Fprintf(state.out, "%s@kolibri:%s$ ", state.user, state.cwd); err != nil {
			break
		}

		line, err := readShellInputLine(reader, state.out, state.pty)
		if err != nil && err != io.EOF {
			fmt.Fprintf(state.err, "read failed: %v\n", err)
			_ = sendExitStatus(channel, 1)
			_ = channel.Close()
			return
		}

		line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if line != "" {
			status, quit := runCommandLine(state, line)
			if quit {
				_ = sendExitStatus(channel, status)
				_ = channel.Close()
				return
			}
		}

		if err == io.EOF {
			break
		}
	}

	_ = sendExitStatus(channel, 0)
	_ = channel.Close()
}

func readShellInputLine(reader *bufio.Reader, echo io.Writer, pty bool) (string, error) {
	if !pty {
		return reader.ReadString('\n')
	}

	buf := make([]byte, 0, 64)
	inEscape := false

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return string(buf), io.EOF
			}
			return "", err
		}

		if inEscape {
			if b >= '@' && b <= '~' {
				inEscape = false
			}
			continue
		}

		switch b {
		case 0:
			continue
		case 27:
			inEscape = true
			continue
		case '\r':
			if next, peekErr := reader.Peek(1); peekErr == nil && len(next) == 1 && next[0] == '\n' {
				_, _ = reader.ReadByte()
			}
			_, _ = io.WriteString(echo, "\r\n")
			return string(buf), nil
		case '\n':
			_, _ = io.WriteString(echo, "\r\n")
			return string(buf), nil
		case 8, 127:
			if len(buf) == 0 {
				continue
			}
			buf = buf[:len(buf)-1]
			_, _ = io.WriteString(echo, "\b \b")
		case 3:
			buf = buf[:0]
			_, _ = io.WriteString(echo, "^C\r\n")
			return "", nil
		default:
			if b < 32 {
				continue
			}
			buf = append(buf, b)
			_, _ = echo.Write([]byte{b})
		}
	}
}

func runExecCommand(state *shellState, command string) int {
	command = strings.TrimSpace(command)
	if command == "" {
		return 0
	}
	status, _ := runCommandLine(state, command)
	return status
}

func runCommandLine(state *shellState, line string) (int, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0, false
	}

	switch fields[0] {
	case "help":
		fmt.Fprintln(state.out, "Commands: help, echo, pwd, cd, ls, cat, whoami, uname, date, exit, quit")
		return 0, false
	case "echo":
		fmt.Fprintln(state.out, strings.Join(fields[1:], " "))
		return 0, false
	case "pwd":
		fmt.Fprintln(state.out, state.cwd)
		return 0, false
	case "cd":
		target := state.cwd
		if len(fields) > 1 {
			target = resolveShellPath(state.cwd, fields[1])
		}
		resolved, _, err := readShellDirectory(target)
		if err == nil {
			state.cwd = resolved
			return 0, false
		}
		info, statErr := os.Stat(target)
		if statErr != nil {
			fmt.Fprintf(state.err, "cd: %v\n", err)
			return 1, false
		}
		if !info.IsDir() {
			fmt.Fprintf(state.err, "cd: not a directory: %s\n", target)
			return 1, false
		}
		state.cwd = path.Clean(target)
		return 0, false
	case "ls":
		target := state.cwd
		if len(fields) > 1 {
			target = resolveShellPath(state.cwd, fields[1])
		}
		if err := listDirectory(state.out, target); err != nil {
			fmt.Fprintf(state.err, "ls: %v\n", err)
			return 1, false
		}
		return 0, false
	case "cat":
		if len(fields) < 2 {
			fmt.Fprintln(state.err, "cat: missing file operand")
			return 1, false
		}
		target := resolveShellPath(state.cwd, fields[1])
		data, err := os.ReadFile(target)
		if err != nil {
			fmt.Fprintf(state.err, "cat: %v\n", err)
			return 1, false
		}
		if len(data) > 0 {
			_, _ = state.out.Write(data)
			if data[len(data)-1] != '\n' {
				fmt.Fprintln(state.out)
			}
		}
		return 0, false
	case "whoami":
		fmt.Fprintln(state.out, state.user)
		return 0, false
	case "uname":
		fmt.Fprintln(state.out, "KolibriOS")
		return 0, false
	case "date":
		fmt.Fprintln(state.out, time.Now().Format(time.RFC3339))
		return 0, false
	case "exit", "quit":
		status := 0
		if len(fields) > 1 {
			value, err := strconv.Atoi(fields[1])
			if err != nil || value < 0 || value > 255 {
				fmt.Fprintf(state.err, "%s: invalid status: %s\n", fields[0], fields[1])
				return 1, false
			}
			status = value
		}
		return status, true
	default:
		fmt.Fprintf(state.err, "unknown command: %s\n", fields[0])
		return 127, false
	}
}

func initialShellDir() string {
	if dir, _, err := readShellDirectory(defaultShellHome); err == nil {
		return dir
	}
	if dir, _, err := readShellDirectory(defaultShellRoot); err == nil {
		return dir
	}
	dir, err := os.Getwd()
	if err != nil || strings.TrimSpace(dir) == "" {
		return defaultShellRoot
	}
	return path.Clean(dir)
}

func resolveShellPath(cwd string, value string) string {
	if strings.TrimSpace(value) == "" {
		return cwd
	}
	if strings.HasPrefix(value, "/") {
		return path.Clean(value)
	}
	if cwd == "" {
		cwd = defaultShellRoot
	}
	return path.Clean(path.Join(cwd, value))
}

func listDirectory(dst io.Writer, dir string) error {
	_, names, err := readShellDirectory(dir)
	if err == nil {
		for _, name := range names {
			fmt.Fprintln(dst, name)
		}
		return nil
	}

	info, statErr := os.Stat(dir)
	if statErr != nil {
		return err
	}
	if info.IsDir() {
		return err
	}
	fmt.Fprintln(dst, info.Name())
	return nil
}

func readShellDirectory(dir string) (string, []string, error) {
	var lastErr error
	for _, candidate := range shellPathCandidates(dir) {
		names, err := readShellDirectoryCandidate(candidate)
		if err == nil {
			return path.Clean(candidate), names, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("read %s: status %d", dir, kos.FileSystemNotFound)
	}
	return "", nil, lastErr
}

func readShellDirectoryCandidate(dir string) ([]string, error) {
	names := []string{}
	start := uint32(0)
	for {
		result, status := kos.ReadFolder(dir, start, folderReadBatch)
		if status != kos.FileSystemOK && status != kos.FileSystemEOF {
			return nil, fmt.Errorf("read %s: status %d", dir, status)
		}
		if len(result.Entries) == 0 {
			break
		}

		for _, entry := range result.Entries {
			if entry.Name == "." || entry.Name == ".." {
				continue
			}
			name := entry.Name
			if entry.Info.Attributes&kos.FileAttributeDirectory != 0 {
				name += "/"
			}
			names = append(names, name)
		}

		start += uint32(len(result.Entries))
		if status == kos.FileSystemEOF || start >= result.Total {
			break
		}
	}

	sort.Strings(names)
	return names, nil
}

func shellPathCandidates(dir string) []string {
	cleaned := path.Clean(dir)
	candidates := []string{cleaned}

	if cleaned != "/" {
		candidates = append(candidates, cleaned+"/")
	}

	if canonical := canonicalShellPath(cleaned); canonical != "" && canonical != cleaned {
		candidates = append(candidates, canonical)
		if canonical != "/" {
			candidates = append(candidates, canonical+"/")
		}
	}

	return uniqueStrings(candidates)
}

func canonicalShellPath(value string) string {
	if len(value) == 0 || value[0] != '/' || value == "/" {
		return value
	}

	parts := strings.Split(value, "/")
	if len(parts) < 2 || parts[1] == "" {
		return value
	}

	parts[1] = strings.ToUpper(parts[1])
	return strings.Join(parts, "/")
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sendExitStatus(channel ssh.Channel, status int) error {
	if status < 0 {
		status = 0
	}
	_, err := channel.SendRequest("exit-status", false, ssh.Marshal(&exitStatusMsg{
		Status: uint32(status),
	}))
	return err
}

func remoteAddress(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return "<unknown>"
	}
	return conn.RemoteAddr().String()
}

func readLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Print(prompt)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return trimConsoleLine(line), nil
}

func readLineWithDefault(label string, value string) (string, error) {
	prompt := label + ": "
	if value != "" {
		prompt = label + " [" + value + "]: "
	}
	line, err := readLine(prompt)
	if err != nil {
		return "", err
	}
	if line == "-" {
		return "", nil
	}
	if line == "" && value != "" {
		return value, nil
	}
	return line, nil
}

func readPassword(console kos.Console, prompt string) (string, error) {
	if console.SupportsInput() {
		fmt.Print(prompt)
		var buf []byte
		for {
			ch := console.Getch()
			if ch == 0 {
				continue
			}
			switch ch {
			case '\r', '\n':
				fmt.Print("\n")
				return string(buf), nil
			case 27:
				fmt.Print("\n")
				return "", errors.New("input canceled")
			case 8, 127:
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
			default:
				if ch >= 32 && ch <= 126 {
					buf = append(buf, byte(ch))
				}
			}
		}
	}
	return readLine(prompt)
}

func readPasswordWithDefault(console kos.Console, label string, value string) (string, error) {
	prompt := label + ": "
	if value != "" {
		prompt = label + " [configured, Enter keeps current, - clears]: "
	}
	result, err := readPassword(console, prompt)
	if err != nil {
		return "", err
	}
	if result == "-" {
		return "", nil
	}
	if result == "" && value != "" {
		return value, nil
	}
	return result, nil
}

func promptYesNoDefault(console kos.Console, prompt string, defaultValue bool) (bool, error) {
	defaultSuffix := "y/N"
	if defaultValue {
		defaultSuffix = "Y/n"
	}

	if console.SupportsInput() {
		fmt.Printf("%s [%s]: ", prompt, defaultSuffix)
		for {
			ch := console.Getch()
			if ch == 0 {
				continue
			}
			switch ch {
			case 'y', 'Y':
				fmt.Print("y\n")
				return true, nil
			case 'n', 'N':
				fmt.Print("n\n")
				return false, nil
			case '\r', '\n':
				fmt.Print("\n")
				return defaultValue, nil
			case 27:
				fmt.Print("\n")
				return false, errors.New("input canceled")
			}
		}
	}

	answer, err := readLine(prompt + " [" + defaultSuffix + "]: ")
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" {
		return defaultValue, nil
	}
	return answer == "y" || answer == "yes", nil
}

func trimConsoleLine(line string) string {
	for len(line) > 0 {
		last := line[len(line)-1]
		if last != '\r' && last != '\n' {
			break
		}
		line = line[:len(line)-1]
	}
	return line
}

func waitForExit(console kos.Console) {
	if console.SupportsInput() {
		fmt.Println("Press any key to close.")
		console.Getch()
	}
}
