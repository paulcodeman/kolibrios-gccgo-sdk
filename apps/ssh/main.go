package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"kos"
)

const (
	appTitle              = "KolibriOS SSH Client"
	defaultPort           = 22
	defaultHome           = "/sys"
	defaultKnownHostsFile = "known_hosts"
	defaultShellRows      = 25
	defaultShellCols      = 80
	defaultShellTerm      = "xterm"
)

var errUsage = errors.New("usage")

var stdinReader = bufio.NewReader(os.Stdin)
var sshIPCBuffer [4096]byte
var sshStage = "startup"

type options struct {
	user       string
	port       int
	portSet    bool
	keyPath    string
	knownHosts string
	acceptNew  bool
	insecure   bool
	cmd        string
	askPass    bool
	timeout    time.Duration
}

type hostKeyChecker struct {
	callback       ssh.HostKeyCallback
	knownHostsPath string
	acceptNew      bool
	console        kos.Console
	promptUnknown  bool
	lastPresented  ssh.PublicKey
}

func main() {
	setSSHStage("startup")
	runtime.GOMAXPROCS(2)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	console, ok := kos.OpenConsole(appTitle)
	if !ok {
		kos.DebugString("ssh: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = fmt.Printf("PANIC at stage %q: %T %v\n", sshStage, recovered, recovered)
			waitForExit(console)
			os.Exit(2)
		}
	}()

	setSSHStage("register IPC buffer")
	kos.RegisterIPCBuffer(sshIPCBuffer[:])
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC | kos.EventMaskNetwork)

	if console.SupportsTitle() {
		console.SetTitle(appTitle)
	}

	setSSHStage("parse arguments")
	opts, target, err := parseArgs()
	if err != nil {
		if errors.Is(err, errUsage) {
			printUsage()
			os.Exit(2)
			return
		}
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(2)
		return
	}

	if target == "" {
		setSSHStage("prompt target")
		target, err = promptTarget()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			waitForExit(console)
			os.Exit(1)
			return
		}
	}

	setSSHStage("resolve target")
	user, host, port, err := resolveTarget(target, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		os.Exit(2)
		return
	}

	if user == "" {
		setSSHStage("prompt username")
		user, err = readLine("Username: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
			waitForExit(console)
			os.Exit(1)
			return
		}
	}

	setSSHStage("build auth")
	authMethods, err := buildAuth(console, opts, user, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		waitForExit(console)
		os.Exit(1)
		return
	}
	if len(authMethods) == 0 {
		fmt.Fprintln(os.Stderr, "ssh: no authentication methods available")
		waitForExit(console)
		os.Exit(2)
		return
	}

	setSSHStage("prepare host key checker")
	checker, err := newHostKeyChecker(console, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		waitForExit(console)
		os.Exit(2)
		return
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: checker.Callback,
		Timeout:         opts.timeout,
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	if console.SupportsTitle() {
		console.SetTitle(user + "@" + host + " - SSH")
	}
	fmt.Printf("Connecting to %s as %s...\n", addr, user)

	setSSHStage("ssh dial")
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		if !reportHostKeyError(err, checker) {
			fmt.Fprintf(os.Stderr, "ssh: %v\n", err)
		}
		waitForExit(console)
		os.Exit(1)
		return
	}
	defer client.Close()

	var runErr error
	if opts.cmd != "" {
		setSSHStage("run remote command")
		runErr = runCommand(client, opts.cmd)
	} else {
		setSSHStage("run shell")
		runErr = runShell(client, console)
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "ssh: %v\n", runErr)
		client.Close()
		waitForExit(console)
		os.Exit(1)
		return
	}

	setSSHStage("shutdown")
	client.Close()
	os.Exit(0)
}

func parseArgs() (options, string, error) {
	var opts options
	opts.port = defaultPort
	opts.timeout = 10 * time.Second

	args := os.Args[1:]
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
			return opts, "", errUsage
		case arg == "-p" || strings.HasPrefix(arg, "-p="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			port, err := strconv.Atoi(value)
			if err != nil || port <= 0 || port > 65535 {
				return opts, "", fmt.Errorf("invalid port: %q", value)
			}
			opts.port = port
			opts.portSet = true
			args = rest
		case arg == "-i" || strings.HasPrefix(arg, "-i="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			opts.keyPath = value
			args = rest
		case arg == "-user" || strings.HasPrefix(arg, "-user="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			opts.user = value
			args = rest
		case arg == "-known-hosts" || strings.HasPrefix(arg, "-known-hosts="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			opts.knownHosts = value
			args = rest
		case arg == "-accept-new":
			opts.acceptNew = true
			args = args[1:]
		case arg == "-insecure":
			opts.insecure = true
			args = args[1:]
		case arg == "-cmd" || strings.HasPrefix(arg, "-cmd="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			opts.cmd = value
			args = rest
		case arg == "-ask-pass":
			opts.askPass = true
			args = args[1:]
		case arg == "-timeout" || strings.HasPrefix(arg, "-timeout="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, "", err
			}
			timeout, err := time.ParseDuration(value)
			if err != nil {
				return opts, "", fmt.Errorf("invalid timeout: %q", value)
			}
			opts.timeout = timeout
			args = rest
		default:
			return opts, "", fmt.Errorf("unknown option: %s", arg)
		}
	}
	if len(args) == 0 {
		return opts, "", nil
	}
	if len(args) != 1 {
		return opts, "", errUsage
	}
	return opts, args[0], nil
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
	fmt.Fprintf(os.Stderr, "Usage: ssh [options] [user@]host[:port]\n\n")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -p <port>             SSH port (default 22)")
	fmt.Fprintln(os.Stderr, "  -user <name>          SSH username")
	fmt.Fprintln(os.Stderr, "  -i <path>             Path to private key")
	fmt.Fprintf(os.Stderr, "  -known-hosts <path>   Path to known_hosts file (default: %s/%s)\n", defaultHome, defaultKnownHostsFile)
	fmt.Fprintln(os.Stderr, "  -accept-new           Append new host keys to known_hosts")
	fmt.Fprintln(os.Stderr, "  -insecure             Accept any host key (insecure)")
	fmt.Fprintln(os.Stderr, "  -cmd <command>        Run a single command instead of a shell")
	fmt.Fprintln(os.Stderr, "  -ask-pass             Prompt for a password even when a key is provided")
	fmt.Fprintln(os.Stderr, "  -timeout <duration>   Connection timeout (default 10s, use 0 to disable)")
	fmt.Fprintln(os.Stderr, "  -h, --help            Show this help text")
	fmt.Fprintln(os.Stderr, "\nNotes:")
	fmt.Fprintln(os.Stderr, "  Run without [user@]host to enter the connection target interactively.")
	fmt.Fprintln(os.Stderr, "  Provide -known-hosts or use -insecure to skip host key checks.")
	fmt.Fprintln(os.Stderr, "  Use -accept-new to append new host keys to the known_hosts file.")
}

func promptTarget() (string, error) {
	for {
		target, err := readLine("Connect to ([user@]host[:port]): ")
		if err != nil {
			return "", err
		}
		target = strings.TrimSpace(target)
		if target != "" {
			return target, nil
		}
	}
}

func resolveTarget(target string, opts options) (string, string, int, error) {
	user, hostport := splitUserHost(target)
	if opts.user != "" {
		user = opts.user
	}

	host, portFromTarget, hasPort, err := splitHostPort(hostport)
	if err != nil {
		return "", "", 0, err
	}

	port := opts.port
	if !opts.portSet && hasPort {
		port = portFromTarget
	}

	if user == "" {
		user = defaultUser()
	}

	if host == "" {
		return "", "", 0, errors.New("missing host")
	}
	return user, host, port, nil
}

func splitUserHost(target string) (string, string) {
	at := strings.LastIndex(target, "@")
	if at == -1 {
		return "", target
	}
	return target[:at], target[at+1:]
}

func splitHostPort(input string) (string, int, bool, error) {
	if strings.HasPrefix(input, "[") {
		end := strings.Index(input, "]")
		if end == -1 {
			return "", 0, false, fmt.Errorf("invalid address: %q", input)
		}
		if end+1 < len(input) && input[end+1] == ':' {
			host, portStr, err := net.SplitHostPort(input)
			if err != nil {
				return "", 0, false, err
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return "", 0, false, fmt.Errorf("invalid port: %q", portStr)
			}
			return host, port, true, nil
		}
		return strings.TrimPrefix(strings.TrimSuffix(input, "]"), "["), 0, false, nil
	}

	if strings.Count(input, ":") == 1 {
		parts := strings.SplitN(input, ":", 2)
		if parts[1] == "" {
			return "", 0, false, fmt.Errorf("missing port in address: %q", input)
		}
		if !isDigits(parts[1]) {
			return "", 0, false, fmt.Errorf("invalid port: %q", parts[1])
		}
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, false, fmt.Errorf("invalid port: %q", parts[1])
		}
		return parts[0], port, true, nil
	}

	return input, 0, false, nil
}

func isDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return len(value) > 0
}

func defaultUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return ""
}

func buildAuth(console kos.Console, opts options, user string, host string) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if opts.keyPath != "" {
		signer, err := loadPrivateKey(console, opts.keyPath)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	password := ""
	if opts.askPass || opts.keyPath == "" {
		var err error
		password, err = readPassword(console, passwordPrompt(user, host))
		if err != nil {
			return nil, err
		}
	}
	if password != "" {
		methods = append(methods, ssh.Password(password))
	}
	methods = append(methods, ssh.KeyboardInteractive(promptKeyboardInteractive(console)))

	return methods, nil
}

func loadPrivateKey(console kos.Console, path string) (ssh.Signer, error) {
	resolved := expandPath(path)
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(data)
	if err == nil {
		return signer, nil
	}

	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		passphrase, perr := readPassword(console, "Key passphrase: ")
		if perr != nil {
			return nil, perr
		}
		return ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	}

	return nil, err
}

func promptKeyboardInteractive(console kos.Console) ssh.KeyboardInteractiveChallenge {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		if instruction != "" {
			fmt.Println(instruction)
		}
		answers := make([]string, len(questions))
		for i, question := range questions {
			var answer string
			var err error
			if i < len(echos) && echos[i] {
				answer, err = readLine(question)
			} else {
				answer, err = readPassword(console, question)
			}
			if err != nil {
				return nil, err
			}
			answers[i] = answer
		}
		return answers, nil
	}
}

func newHostKeyChecker(console kos.Console, opts options) (*hostKeyChecker, error) {
	checker := &hostKeyChecker{
		console:       console,
		promptUnknown: console.SupportsInput(),
	}
	if opts.insecure {
		checker.callback = ssh.InsecureIgnoreHostKey()
		return checker, nil
	}

	path := opts.knownHosts
	if path == "" {
		path = defaultKnownHostsPath()
	}
	if path == "" {
		if !checker.promptUnknown {
			return nil, errors.New("known_hosts path not set (use -known-hosts or -insecure)")
		}
		checker.acceptNew = opts.acceptNew
		return checker, nil
	}
	path = expandPath(path)

	if opts.acceptNew {
		if err := ensureKnownHostsFile(path); err != nil {
			return nil, err
		}
	} else {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				if !checker.promptUnknown {
					return nil, fmt.Errorf("known_hosts file not found: %s (use -accept-new or -insecure)", path)
				}
				checker.knownHostsPath = path
				checker.acceptNew = opts.acceptNew
				return checker, nil
			}
			return nil, err
		}
	}

	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}

	checker.callback = cb
	checker.knownHostsPath = path
	checker.acceptNew = opts.acceptNew
	return checker, nil
}

func (c *hostKeyChecker) Callback(hostname string, remote net.Addr, key ssh.PublicKey) error {
	c.lastPresented = key
	if c.callback == nil {
		return c.handleUnknownHost(hostname, key)
	}

	err := c.callback(hostname, remote, key)
	if err == nil {
		return nil
	}

	if keyErr, ok := err.(*knownhosts.KeyError); ok && len(keyErr.Want) == 0 {
		return c.handleUnknownHost(hostname, key)
	}

	return err
}

func (c *hostKeyChecker) handleUnknownHost(hostname string, key ssh.PublicKey) error {
	if c.acceptNew {
		if err := c.rememberHost(hostname, key); err != nil {
			return err
		}
		return nil
	}

	if c.promptUnknown {
		fmt.Fprintf(os.Stdout, "The authenticity of host %s can't be established.\n", hostname)
		fmt.Fprintf(os.Stdout, "Key fingerprint: %s\n", ssh.FingerprintSHA256(key))
		ok, err := promptYesNo(c.console, "Trust this host key?")
		if err != nil {
			return err
		}
		if ok {
			return c.rememberHost(hostname, key)
		}
	}

	return &knownhosts.KeyError{}
}

func (c *hostKeyChecker) rememberHost(hostname string, key ssh.PublicKey) error {
	if c.knownHostsPath == "" {
		fmt.Printf("Trusted host key for %s for this session\n", hostname)
		return nil
	}
	if err := ensureKnownHostsFile(c.knownHostsPath); err != nil {
		return err
	}
	if err := appendKnownHost(c.knownHostsPath, hostname, key); err != nil {
		return err
	}
	fmt.Printf("Added host key for %s to %s\n", hostname, c.knownHostsPath)
	return nil
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	line := knownhosts.Line([]string{hostname}, key)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}

func defaultKnownHostsPath() string {
	if home := defaultHomePath(); home != "" {
		return filepath.Join(home, defaultKnownHostsFile)
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return filepath.Join(wd, defaultKnownHostsFile)
	}
	return ""
}

func ensureKnownHostsFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != path {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return file.Close()
}

func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if path[0] != '~' {
		return path
	}
	if path == "~" {
		if home := defaultHomePath(); home != "" {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home := defaultHomePath(); home != "" {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func defaultHomePath() string {
	if defaultHome != "" {
		return defaultHome
	}
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return ""
}

func reportHostKeyError(err error, checker *hostKeyChecker) bool {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		if len(keyErr.Want) == 0 {
			fmt.Fprintln(os.Stderr, "ssh: unknown host key")
			if checker != nil && checker.lastPresented != nil {
				fmt.Fprintf(os.Stderr, "ssh: presented key fingerprint: %s\n", ssh.FingerprintSHA256(checker.lastPresented))
			}
			if checker != nil && checker.knownHostsPath != "" {
				fmt.Fprintf(os.Stderr, "ssh: add to %s or use -accept-new\n", checker.knownHostsPath)
			} else {
				fmt.Fprintln(os.Stderr, "ssh: use -known-hosts or -insecure to proceed")
			}
			return true
		}
		fmt.Fprintln(os.Stderr, "ssh: host key mismatch")
		if checker != nil && checker.lastPresented != nil {
			fmt.Fprintf(os.Stderr, "ssh: presented key fingerprint: %s\n", ssh.FingerprintSHA256(checker.lastPresented))
		}
		fmt.Fprintln(os.Stderr, "ssh: update your known_hosts file if this is expected")
		return true
	}

	var revokedErr *knownhosts.RevokedError
	if errors.As(err, &revokedErr) {
		fmt.Fprintf(os.Stderr, "ssh: host key is revoked: %s\n", revokedErr.Revoked.String())
		return true
	}

	return false
}

func runCommand(client *ssh.Client, command string) error {
	output, err := runRemoteOutput(client, command)
	if len(output) > 0 {
		if _, writeErr := os.Stdout.Write(output); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func runRemoteOutput(client *ssh.Client, command string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := session.Start(wrapShellCommand(command + " 2>&1")); err != nil {
		return nil, err
	}

	output, readErr := io.ReadAll(stdout)
	waitErr := session.Wait()
	if readErr != nil {
		return output, readErr
	}
	return output, waitErr
}

func runShell(client *ssh.Client, console kos.Console) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 38400,
		ssh.TTY_OP_OSPEED: 38400,
	}
	if err := session.RequestPty(defaultShellTerm, defaultShellRows, defaultShellCols, modes); err != nil {
		return err
	}
	if err := session.Shell(); err != nil {
		return err
	}

	outputErrs := make(chan error, 2)
	stopInput := make(chan struct{})
	go copyShellStream(os.Stdout, stdout, outputErrs)
	go copyShellStream(os.Stderr, stderr, outputErrs)
	go pumpConsoleToSSH(console, stdin, stopInput)

	waitErr := session.Wait()
	close(stopInput)
	_ = stdin.Close()
	_ = session.Close()

	for i := 0; i < 2; i++ {
		if copyErr := <-outputErrs; copyErr != nil && copyErr != io.EOF && waitErr == nil {
			waitErr = copyErr
		}
	}

	return waitErr
}

func copyShellStream(dst io.Writer, src io.Reader, errs chan<- error) {
	_, err := io.Copy(dst, src)
	errs <- err
}

func pumpConsoleToSSH(console kos.Console, dst io.WriteCloser, stop <-chan struct{}) {
	defer dst.Close()
	for {
		select {
		case <-stop:
			return
		default:
		}

		data := readConsoleKey(console)
		if len(data) == 0 {
			kos.Sleep(1)
			continue
		}
		if _, err := dst.Write(data); err != nil {
			return
		}
	}
}

func readConsoleKey(console kos.Console) []byte {
	if console.KeyHit() == false {
		return nil
	}
	if console.SupportsInputFull() {
		key := console.Getch2()
		if key == 0 {
			return nil
		}
		ascii := byte(key)
		scan := byte(key >> 8)
		if ascii != 0 {
			return translateConsoleASCII(ascii)
		}
		return translateConsoleScan(scan)
	}
	if console.SupportsInput() {
		key := console.Getch()
		if key == 0 {
			return nil
		}
		return translateConsoleASCII(byte(key))
	}
	return nil
}

func translateConsoleASCII(ch byte) []byte {
	switch ch {
	case 0:
		return nil
	case '\r':
		return []byte{'\r'}
	case '\n':
		return []byte{'\r'}
	case 8:
		return []byte{127}
	default:
		return []byte{ch}
	}
}

func translateConsoleScan(scan byte) []byte {
	switch scan {
	case 71:
		return []byte("\x1b[H")
	case 72:
		return []byte("\x1b[A")
	case 73:
		return []byte("\x1b[5~")
	case 75:
		return []byte("\x1b[D")
	case 77:
		return []byte("\x1b[C")
	case 79:
		return []byte("\x1b[F")
	case 80:
		return []byte("\x1b[B")
	case 81:
		return []byte("\x1b[6~")
	case 82:
		return []byte("\x1b[2~")
	case 83:
		return []byte("\x1b[3~")
	default:
		return nil
	}
}

func sshExitStatus(err error) (int, bool) {
	for err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = unwrapper.Unwrap()
	}
	return 0, false
}

func detectRemoteWorkingDir(client *ssh.Client) (string, error) {
	output, err := runRemoteOutput(client, "pwd")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func runShellBuiltin(client *ssh.Client, cwd string, line string) (string, bool, error) {
	if line == "pwd" {
		if cwd != "" {
			fmt.Println(cwd)
			return cwd, true, nil
		}
		next, err := detectRemoteWorkingDir(client)
		if err != nil {
			return cwd, true, err
		}
		if next != "" {
			fmt.Println(next)
		}
		return next, true, nil
	}

	if line == "cd" || strings.HasPrefix(line, "cd ") {
		target := strings.TrimSpace(line[2:])
		if target == "" {
			target = "~"
		}

		command := "cd " + shellPathArg(target) + " && pwd"
		if cwd != "" {
			command = "cd " + shellQuote(cwd) + " && " + command
		}

		output, err := runRemoteOutput(client, command)
		if len(output) > 0 && err != nil {
			if _, writeErr := os.Stdout.Write(output); writeErr != nil {
				return cwd, true, writeErr
			}
		}
		if err != nil {
			return cwd, true, err
		}

		next := strings.TrimSpace(string(output))
		if next == "" {
			return cwd, true, errors.New("remote shell returned empty directory")
		}
		return next, true, nil
	}

	return cwd, false, nil
}

func wrapShellCommand(command string) string {
	return "sh -lc " + shellQuote(command)
}

func passwordPrompt(user string, host string) string {
	if user == "" || host == "" {
		return "Password: "
	}
	return user + "@" + host + "'s password: "
}

func shellPrompt(user string, host string, cwd string) string {
	location := cwd
	if location == "" {
		location = "~"
	}
	suffix := "$ "
	if user == "root" {
		suffix = "# "
	}
	if user == "" || host == "" {
		return location + suffix
	}
	return user + "@" + host + ":" + location + suffix
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func shellPathArg(value string) string {
	if value == "~" || strings.HasPrefix(value, "~/") {
		return value
	}
	return shellQuote(value)
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
				return "", errors.New("password entry canceled")
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

func promptYesNo(console kos.Console, prompt string) (bool, error) {
	if console.SupportsInput() {
		fmt.Printf("%s [y/N]: ", prompt)
		for {
			ch := console.Getch()
			if ch == 0 {
				continue
			}
			switch ch {
			case 'y', 'Y':
				fmt.Print("y\n")
				return true, nil
			case 'n', 'N', 27:
				fmt.Print("n\n")
				return false, nil
			case '\r', '\n':
				fmt.Print("\n")
				return false, nil
			}
		}
	}

	answer, err := readLine(prompt + " [y/N]: ")
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
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

func setSSHStage(stage string) {
	sshStage = stage
	kos.DebugString("ssh stage: " + stage)
}

func waitForExit(console kos.Console) {
	if console.SupportsInput() {
		fmt.Println("Press any key to close.")
		console.Getch()
	}
}
