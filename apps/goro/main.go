package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MagicalTux/goro/core/phpctx"
	"github.com/MagicalTux/goro/core/phpv"
	_ "github.com/MagicalTux/goro/ext/standard"
	"gopkg.in/readline.v1"
	"kos"
)

const consoleTitle = "Goro PHP Console"

var serverIndexFiles = []string{"index.php", "index.html"}

type launchOptions struct {
	script string
	server serverOptions
}

type serverOptions struct {
	enabled bool
	addr    string
	docroot string
	router  string
}

type requestMode int

const (
	requestModePHP requestMode = iota
	requestModeStatic
)

type resolvedRequest struct {
	mode           requestMode
	execFile       string
	scriptFilename string
	scriptName     string
	pathInfo       string
	staticFile     string
}

type phpServer struct {
	process *phpctx.Process
	docroot string
	router  string
}

type phpExecution struct {
	result *phpv.ZVal
	output []byte
	header http.Header
	status int
}

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("goro: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	options, err := parseLaunchOptions(os.Args[1:])
	if err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}

	sapi := "cli"
	if options.server.enabled {
		sapi = "cli-server"
	}

	process := phpctx.NewProcess(sapi)
	if err := loadLocalPHPIni(process); err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}
	if err := process.CommandLine(os.Args); err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}

	if options.server.enabled {
		if err := runServer(process, options.server); err != nil {
			_, _ = fmt.Printf("goro: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ctx := phpctx.NewGlobal(context.Background(), process)
	if options.script == "" {
		if err := runREPL(ctx); err != nil {
			_, _ = fmt.Printf("goro: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := ctx.RunFile(options.script); err != nil {
		_, _ = fmt.Printf("goro: %v\n", err)
		os.Exit(1)
		return
	}
}

func parseLaunchOptions(args []string) (launchOptions, error) {
	options := launchOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "-S":
			if index+1 >= len(args) {
				return options, fmt.Errorf("missing value for -S")
			}
			options.server.enabled = true
			options.server.addr = args[index+1]
			index++
		case "-t":
			if index+1 >= len(args) {
				return options, fmt.Errorf("missing value for -t")
			}
			options.server.docroot = args[index+1]
			index++
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown option %s", arg)
			}
			if options.server.enabled {
				if options.server.router == "" {
					options.server.router = arg
					continue
				}
				return options, fmt.Errorf("unexpected argument %s", arg)
			}
			if options.script == "" {
				options.script = arg
				continue
			}
			return options, fmt.Errorf("unexpected argument %s", arg)
		}
	}

	if options.server.enabled {
		if options.server.addr == "" {
			return options, fmt.Errorf("missing listen address for -S")
		}
		if options.server.docroot == "" {
			workingDir, err := os.Getwd()
			if err != nil {
				return options, err
			}
			options.server.docroot = workingDir
		}
	}

	return options, nil
}

func loadLocalPHPIni(process *phpctx.Process) error {
	iniPath, err := localPHPIniPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(iniPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	file, err := os.Open(iniPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := parsePHPIni(process, file); err != nil {
		return fmt.Errorf("%s: %v", iniPath, err)
	}
	process.SetConfig("cfg_file_path", phpv.ZString(cleanFSPath(iniPath)).ZVal())
	return nil
}

func localPHPIniPath() (string, error) {
	loaderPath := kos.LoaderPath()
	if loaderPath == "" && len(os.Args) != 0 {
		loaderPath = os.Args[0]
	}
	if loaderPath == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(workingDir, "php.ini"), nil
	}
	if !strings.HasPrefix(loaderPath, "/") {
		absolutePath, err := filepath.Abs(loaderPath)
		if err != nil {
			return "", err
		}
		loaderPath = absolutePath
	}
	return filepath.Join(filepath.Dir(loaderPath), "php.ini"), nil
}

func parsePHPIni(process *phpctx.Process, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		separator := strings.IndexByte(line, '=')
		if separator < 0 {
			return fmt.Errorf("line %d: expected key = value", lineNumber)
		}

		key := strings.ToLower(strings.TrimSpace(line[:separator]))
		if key == "" {
			return fmt.Errorf("line %d: missing directive name", lineNumber)
		}

		value := parsePHPIniValue(line[separator+1:])
		process.SetConfig(phpv.ZString(key), value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func parsePHPIniValue(raw string) *phpv.ZVal {
	value := strings.TrimSpace(stripPHPIniComment(raw))
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	switch strings.ToLower(value) {
	case "on", "true", "yes":
		return phpv.ZInt(1).ZVal()
	case "off", "false", "no":
		return phpv.ZInt(0).ZVal()
	case "null":
		return phpv.ZNULL.ZVal()
	}

	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return phpv.ZInt(parsed).ZVal()
	}
	return phpv.ZString(value).ZVal()
}

func stripPHPIniComment(value string) string {
	quote := byte(0)
	for index := 0; index < len(value); index++ {
		ch := value[index]
		switch quote {
		case 0:
			switch ch {
			case '\'', '"':
				quote = ch
			case ';', '#':
				if index == 0 || value[index-1] == ' ' || value[index-1] == '\t' {
					return strings.TrimSpace(value[:index])
				}
			}
		case '\'', '"':
			if ch == quote {
				quote = 0
			}
		}
	}
	return strings.TrimSpace(value)
}

func runServer(process *phpctx.Process, options serverOptions) error {
	docroot, err := filepath.Abs(options.docroot)
	if err != nil {
		return err
	}
	docroot = cleanFSPath(docroot)

	router := ""
	if options.router != "" {
		router, err = filepath.Abs(options.router)
		if err != nil {
			return err
		}
		router = cleanFSPath(router)
		if _, err := os.Stat(router); err != nil {
			return err
		}
	}

	_, _ = fmt.Printf("goro: serving %s on http://%s\n", docroot, displayListenAddr(options.addr))
	return http.ListenAndServe(options.addr, &phpServer{
		process: process,
		docroot: docroot,
		router:  router,
	})
}

func displayListenAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0" + addr
	}
	return addr
}

func (server *phpServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost:
	default:
		writer.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(writer, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cleaned := cleanRequestPath(requestPathValue(request))
	resolved, ok := server.resolveRequest(cleaned)

	if server.router != "" {
		server.serveRouter(writer, request, resolved, ok)
		return
	}

	if !ok {
		http.Error(writer, "Not Found", http.StatusNotFound)
		return
	}

	server.serveResolved(writer, request, resolved)
}

func requestPathValue(request *http.Request) string {
	if request == nil {
		return "/"
	}
	if request.URL != nil && request.URL.Path != "" {
		return request.URL.Path
	}
	if request.RequestURI != "" {
		if separator := strings.IndexByte(request.RequestURI, '?'); separator >= 0 {
			return request.RequestURI[:separator]
		}
		return request.RequestURI
	}
	return "/"
}

func (server *phpServer) serveRouter(writer http.ResponseWriter, request *http.Request, fallback resolvedRequest, hasFallback bool) {
	execution, err := server.executePHP(request, server.routerTarget())
	if err != nil {
		http.Error(writer, "goro: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if routerReturnedFalse(execution.result) {
		if !hasFallback {
			http.Error(writer, "Not Found", http.StatusNotFound)
			return
		}
		server.serveResolved(writer, request, fallback)
		return
	}

	server.applyPHPResponse(writer, execution)
}

func routerReturnedFalse(result *phpv.ZVal) bool {
	if result == nil || result.GetType() != phpv.ZtBool {
		return false
	}
	return !bool(result.Value().(phpv.ZBool))
}

func (server *phpServer) serveResolved(writer http.ResponseWriter, request *http.Request, resolved resolvedRequest) {
	switch resolved.mode {
	case requestModeStatic:
		server.serveStatic(writer, resolved)
	case requestModePHP:
		server.servePHP(writer, request, resolved)
	default:
		http.Error(writer, "Not Found", http.StatusNotFound)
	}
}

func (server *phpServer) serveStatic(writer http.ResponseWriter, resolved resolvedRequest) {
	data, err := os.ReadFile(resolved.staticFile)
	if err != nil {
		http.Error(writer, "Not Found", http.StatusNotFound)
		return
	}

	contentType := mime.TypeByExtension(strings.ToLower(path.Ext(resolved.staticFile)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writer.Header().Set("Content-Type", contentType)
	_, _ = writer.Write(data)
}

func (server *phpServer) servePHP(writer http.ResponseWriter, request *http.Request, resolved resolvedRequest) {
	execution, err := server.executePHP(request, resolved)
	if err != nil {
		http.Error(writer, "goro: "+err.Error(), http.StatusInternalServerError)
		return
	}
	server.applyPHPResponse(writer, execution)
}

func (server *phpServer) executePHP(request *http.Request, resolved resolvedRequest) (*phpExecution, error) {
	ctx := phpctx.NewGlobalReq(request, server.process)
	setResolvedServerVars(ctx, server.docroot, resolved)

	var output bytes.Buffer
	ctx.SetOutput(&output)
	result, err := runPHPFile(ctx, resolved.execFile)
	if err != nil {
		return nil, err
	}

	return &phpExecution{
		result: result,
		output: output.Bytes(),
		header: ctx.ResponseHeaders(),
		status: ctx.ResponseStatusCode(),
	}, nil
}

func (server *phpServer) applyPHPResponse(writer http.ResponseWriter, execution *phpExecution) {
	if execution == nil {
		http.Error(writer, "goro: empty response", http.StatusInternalServerError)
		return
	}

	for key, values := range execution.header {
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			writer.Header().Add(key, values[valueIndex])
		}
	}
	if writer.Header().Get("Content-Type") == "" {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

	status := execution.status
	if status == 0 {
		status = http.StatusOK
	}
	writer.WriteHeader(status)

	if len(execution.output) > 0 {
		_, _ = writer.Write(execution.output)
	}
}

func runPHPFile(ctx *phpctx.Global, filename string) (*phpv.ZVal, error) {
	result, err := ctx.Require(ctx, phpv.ZString(filename))
	err = phpv.FilterError(err)
	if closeErr := ctx.Close(); err == nil {
		err = closeErr
	}
	return result, err
}

func setResolvedServerVars(ctx *phpctx.Global, docroot string, resolved resolvedRequest) {
	_ = ctx.SetServerValue("DOCUMENT_ROOT", docroot)
	_ = ctx.SetServerValue("SCRIPT_FILENAME", resolved.scriptFilename)
	if resolved.scriptName != "" {
		_ = ctx.SetServerValue("SCRIPT_NAME", resolved.scriptName)
		phpSelf := resolved.scriptName
		if resolved.pathInfo != "" {
			phpSelf += resolved.pathInfo
		}
		_ = ctx.SetServerValue("PHP_SELF", phpSelf)
	}
	if resolved.pathInfo != "" {
		_ = ctx.SetServerValue("PATH_INFO", resolved.pathInfo)
		_ = ctx.SetServerValue("PATH_TRANSLATED", cleanFSPath(filepath.Join(docroot, strings.TrimPrefix(resolved.pathInfo, "/"))))
	}
}

func (server *phpServer) resolveRequest(cleaned string) (resolvedRequest, bool) {
	if resolved, ok := server.resolveExistingPath(cleaned); ok {
		return resolved, true
	}
	if resolved, ok := server.resolvePathInfoPHP(cleaned); ok {
		return resolved, true
	}
	if resolved, ok := server.resolveDirectoryIndex(cleaned); ok {
		return resolved, true
	}

	return resolvedRequest{}, false
}

func (server *phpServer) resolveExistingPath(cleaned string) (resolvedRequest, bool) {
	fullPath := server.docrootPath(cleaned)
	info, err := os.Stat(fullPath)
	if err != nil {
		return resolvedRequest{}, false
	}

	if info.IsDir() {
		for index := 0; index < len(serverIndexFiles); index++ {
			scriptName := path.Join(cleaned, serverIndexFiles[index])
			if !strings.HasPrefix(scriptName, "/") {
				scriptName = "/" + scriptName
			}
			candidate := server.docrootPath(scriptName)
			candidateInfo, statErr := os.Stat(candidate)
			if statErr != nil || candidateInfo.IsDir() {
				continue
			}
			if strings.EqualFold(path.Ext(candidate), ".php") {
				return resolvedRequest{
					mode:           requestModePHP,
					execFile:       candidate,
					scriptFilename: candidate,
					scriptName:     scriptName,
				}, true
			}
			return resolvedRequest{
				mode:       requestModeStatic,
				staticFile: candidate,
			}, true
		}
		return resolvedRequest{}, false
	}

	if strings.EqualFold(path.Ext(fullPath), ".php") {
		return resolvedRequest{
			mode:           requestModePHP,
			execFile:       fullPath,
			scriptFilename: fullPath,
			scriptName:     cleaned,
		}, true
	}

	return resolvedRequest{
		mode:       requestModeStatic,
		staticFile: fullPath,
	}, true
}

func (server *phpServer) resolvePathInfoPHP(cleaned string) (resolvedRequest, bool) {
	candidate := cleaned
	for {
		fullPath := server.docrootPath(candidate)
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() && strings.EqualFold(path.Ext(fullPath), ".php") {
			return resolvedRequest{
				mode:           requestModePHP,
				execFile:       fullPath,
				scriptFilename: fullPath,
				scriptName:     candidate,
				pathInfo:       strings.TrimPrefix(cleaned, candidate),
			}, true
		}

		if candidate == "/" {
			break
		}
		candidate = path.Dir(candidate)
		if candidate == "." {
			candidate = "/"
		}
	}

	return resolvedRequest{}, false
}

func (server *phpServer) resolveDirectoryIndex(cleaned string) (resolvedRequest, bool) {
	if path.Ext(cleaned) != "" {
		return resolvedRequest{}, false
	}

	searchDir := cleaned
	for {
		for index := 0; index < len(serverIndexFiles); index++ {
			scriptName := path.Join(searchDir, serverIndexFiles[index])
			if !strings.HasPrefix(scriptName, "/") {
				scriptName = "/" + scriptName
			}

			candidate := server.docrootPath(scriptName)
			candidateInfo, err := os.Stat(candidate)
			if err != nil || candidateInfo.IsDir() {
				continue
			}

			pathInfo := strings.TrimPrefix(cleaned, searchDir)
			if pathInfo != "" && !strings.HasPrefix(pathInfo, "/") {
				pathInfo = "/" + pathInfo
			}
			if strings.EqualFold(path.Ext(candidate), ".php") {
				return resolvedRequest{
					mode:           requestModePHP,
					execFile:       candidate,
					scriptFilename: candidate,
					scriptName:     scriptName,
					pathInfo:       pathInfo,
				}, true
			}
			return resolvedRequest{
				mode:       requestModeStatic,
				staticFile: candidate,
			}, true
		}

		if searchDir == "/" {
			break
		}
		searchDir = path.Dir(searchDir)
		if searchDir == "." {
			searchDir = "/"
		}
	}

	return resolvedRequest{}, false
}

func (server *phpServer) routerTarget() resolvedRequest {
	scriptName := server.routerScriptName()
	return resolvedRequest{
		mode:           requestModePHP,
		execFile:       server.router,
		scriptFilename: server.router,
		scriptName:     scriptName,
	}
}

func (server *phpServer) routerScriptName() string {
	if relative, ok := trimDocrootPrefix(server.docroot, server.router); ok {
		return relative
	}
	return "/" + path.Base(server.router)
}

func (server *phpServer) docrootPath(requestPath string) string {
	relativePath := strings.TrimPrefix(requestPath, "/")
	if relativePath == "" {
		return server.docroot
	}
	return cleanFSPath(filepath.Join(server.docroot, relativePath))
}

func trimDocrootPrefix(docroot string, file string) (string, bool) {
	docroot = cleanFSPath(docroot)
	file = cleanFSPath(file)
	if file == docroot {
		return "/", true
	}

	prefix := docroot
	if prefix != "/" {
		prefix += "/"
	}
	if !strings.HasPrefix(file, prefix) {
		return "", false
	}
	return "/" + strings.TrimPrefix(file, prefix), true
}

func cleanRequestPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean("/" + value)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}

func cleanFSPath(value string) string {
	return path.Clean(filepath.ToSlash(value))
}

func runREPL(ctx *phpctx.Global) error {
	_, _ = fmt.Printf(
		"Goro PHP Console (KolibriOS)\n" +
			"Usage: goro [script.php] | goro -S :80 [-t docroot] [router.php]\n" +
			"Server sample: apps/goro/index.php\n" +
			"Enter PHP code and press Enter.\n" +
			"Press Ctrl+C, Esc, or Ctrl+D to exit.\n",
	)

	rl, err := readline.NewEx(&readline.Config{Prompt: "php> "})
	if err != nil {
		return err
	}
	defer rl.Close()
	output := &replOutputWriter{w: rl.Stdout()}
	ctx.SetOutput(output)

	evalFn, err := ctx.GetFunction(ctx, phpv.ZString("eval"))
	if err != nil {
		return err
	}

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			_, _ = fmt.Printf("\n")
			return nil
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" || line == ".exit" {
			return nil
		}

		ctx.ResetExecutionDeadline()
		output.reset()
		result, err := evalLine(ctx, evalFn, normalizeREPLLine(line))
		if err != nil {
			if _, ok := err.(*phpv.PhpExit); ok {
				return nil
			}
			output.finishLine()
			_, _ = fmt.Printf("Error: %v\n", err)
			if hint := replHint(line, err); hint != "" {
				_, _ = fmt.Printf("Hint: %s\n", hint)
			}
			rl.Refresh()
			continue
		}
		ctx.Flush()
		output.finishLine()
		if result != nil && result.GetType() != phpv.ZtNull {
			_, _ = fmt.Printf("%s\n", result.String())
		}
		rl.Refresh()
	}
}

func evalLine(ctx *phpctx.Global, evalFn phpv.Callable, line string) (result *phpv.ZVal, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return ctx.CallZVal(ctx, evalFn, []*phpv.ZVal{phpv.ZString(line).ZVal()}, nil)
}

func normalizeREPLLine(line string) string {
	switch {
	case strings.HasSuffix(line, ";"):
		return line
	case strings.HasSuffix(line, "{"):
		return line
	case strings.HasSuffix(line, "}"):
		return line
	case strings.HasSuffix(line, ":"):
		return line
	default:
		return line + ";"
	}
}

func replHint(line string, err error) string {
	if err == nil {
		return ""
	}
	if (strings.Contains(err.Error(), "write context") || strings.Contains(err.Error(), "not writable")) &&
		strings.Contains(line, "=") && !strings.Contains(line, "$") {
		return "PHP variables must start with $, for example $x = []"
	}
	return ""
}

type replOutputWriter struct {
	w                io.Writer
	wrote            bool
	endedWithNewline bool
}

func (w *replOutputWriter) Write(p []byte) (int, error) {
	if len(p) != 0 {
		w.wrote = true
		last := p[len(p)-1]
		w.endedWithNewline = last == '\n' || last == '\r'
	}
	return w.w.Write(p)
}

func (w *replOutputWriter) reset() {
	w.wrote = false
	w.endedWithNewline = false
}

func (w *replOutputWriter) finishLine() {
	if w.wrote && !w.endedWithNewline {
		_, _ = w.w.Write([]byte("\n"))
		w.endedWithNewline = true
	}
}
