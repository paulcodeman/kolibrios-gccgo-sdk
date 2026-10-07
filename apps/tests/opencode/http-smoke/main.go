package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"internal/cpu"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/http/internal/testcert"
	"net/rpc"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"kos"
)

func check(ok bool, what string) {
	if !ok {
		panic(what)
	}
}

func passed(what string) {
	fmt.Println(what)
	kos.DebugString(what + "\n")
}

//go:noinline
func callerMetadata() bool {
	pc, file, line, ok := runtime.Caller(0)
	f := runtime.FuncForPC(pc)
	return ok && f != nil && strings.HasSuffix(f.Name(), ".callerMetadata") &&
		strings.HasSuffix(file, "main.go") && line > 0
}

type Calculator int
type Operands struct{ A, B int }

func (Calculator) Add(args Operands, result *int) error {
	*result = args.A + args.B
	return nil
}

func main() {
	console, ok := kos.LoadConsole()
	if !ok {
		return
	}
	if !console.Init(48, 25, 48, 1000, "OpenCode upstream HTTP test") {
		return
	}
	defer console.Close()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("OPENCODE_HTTP_FAIL: %v\n", err)
			kos.DebugString(fmt.Sprintf("OPENCODE_HTTP_FAIL: %v", err))
			kos.SleepSeconds(30)
		}
	}()
	fmt.Printf("CPU: %s\n", cpu.Name())
	check(cpu.Name() != "", "CPU vendor name")
	check(callerMetadata(), "real caller function and source location")
	println("PASS runtime builtin print", true, false)
	passed("PASS CPU and source metadata")
	sleepStarted := make(chan struct{})
	sleepDone := make(chan struct{})
	go func() {
		close(sleepStarted)
		time.Sleep(time.Second)
		close(sleepDone)
	}()
	<-sleepStarted
	start := time.Now()
	<-time.After(20 * time.Millisecond)
	check(time.Since(start) < 500*time.Millisecond, "sleep parks only its calling goroutine")
	<-sleepDone
	passed("PASS cooperative sleep and independent timers")
	signature := reflect.TypeOf(func(int, ...string) (string, error) { return "", nil })
	check(signature.NumIn() == 2 && signature.NumOut() == 2 && signature.IsVariadic() && signature.In(1).Elem().Kind() == reflect.String, "real function signature")
	method, found := reflect.TypeOf(Calculator(0)).MethodByName("Add")
	check(found && method.Type.NumIn() == 3 && method.Type.NumOut() == 1 && method.Func.IsValid(), "exported method metadata")
	prefix := "closure: "
	values := reflect.ValueOf(func(input string) string { return prefix + input }).Call([]reflect.Value{reflect.ValueOf("works")})
	check(len(values) == 1 && values[0].String() == "closure: works", "libffi call with Go closure")
	generated := reflect.MakeFunc(reflect.TypeOf(func(int) int { return 0 }), func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(int(args[0].Int()) + 2)}
	}).Interface().(func(int) int)
	check(generated(40) == 42, "MakeFunc with original libffi Go closure")
	var methodSum int
	bound := reflect.ValueOf(Calculator(0)).MethodByName("Add")
	check(bound.Type().NumIn() == 2 && bound.NumMethod() == 0, "bound method signature")
	methodResult := bound.Call([]reflect.Value{reflect.ValueOf(Operands{20, 22}), reflect.ValueOf(&methodSum)})
	check(methodSum == 42 && methodResult[0].Interface() == nil, "bound method call")
	methodSum = 0
	boundFunc := bound.Interface().(func(Operands, *int) error)
	check(boundFunc(Operands{19, 23}, &methodSum) == nil && methodSum == 42, "bound method as Go function")
	snapshotValue := 7
	snapshot := reflect.ValueOf(&snapshotValue).Elem().Interface().(int)
	snapshotValue = 9
	check(snapshot == 7, "addressable value interface snapshot")
	mapFunc := reflect.MakeFunc(reflect.TypeOf(func() map[string]int { return nil }), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(map[string]int{"answer": 42})}
	}).Interface().(func() map[string]int)
	check(mapFunc()["answer"] == 42, "MakeFunc direct map return")
	errorFunc := reflect.MakeFunc(reflect.TypeOf(func() error { return nil }), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(errors.New("expected error"))}
	}).Interface().(func() error)
	check(errorFunc().Error() == "expected error", "MakeFunc concrete return assigned to error interface")
	var html bytes.Buffer
	tmpl, err := template.New("escaped").Parse("<p>{{.}}</p>")
	check(err == nil, "HTML template parse")
	err = tmpl.Execute(&html, "<script>alert(1)</script>")
	check(err == nil && html.String() == "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>", "original HTML autoescaping")
	passed("PASS reflect metadata, closure calls and HTML escaping")
	rpcServer := rpc.NewServer()
	check(rpcServer.RegisterName("Calculator", Calculator(0)) == nil, "RPC register original method")
	rpcClientConn, rpcPeer := net.Pipe()
	go rpcServer.ServeConn(rpcPeer)
	rpcClient := rpc.NewClient(rpcClientConn)
	var sum int
	err = rpcClient.Call("Calculator.Add", Operands{20, 22}, &sum)
	check(err == nil && sum == 42, "original RPC reflective invocation")
	rpcClient.Close()
	passed("PASS original RPC over net.Pipe")

	check((time.Duration(1500)*time.Millisecond).Round(time.Second) == 2*time.Second, "duration round")
	check(time.Unix(1700000000, 500000000).Round(time.Second).Unix() == 1700000001, "time round")
	check(time.Unix(-1, 500000000).Truncate(time.Second).Unix() == -1, "negative time truncate")
	passed("PASS upstream time rounding")

	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/plain")
	recorder.Header().Set("Trailer", "X-Done")
	recorder.WriteHeader(http.StatusCreated)
	recorder.WriteString("recorded")
	recorder.Header().Set("X-Done", "yes")
	response := recorder.Result()
	body, err := io.ReadAll(response.Body)
	check(err == nil && response.StatusCode == 201 && string(body) == "recorded" && response.Trailer.Get("X-Done") == "yes", "response recorder")
	req := httptest.NewRequest("POST", "http://example.com/path?q=yes", strings.NewReader("hello"))
	check(req.URL.Path == "/path" && req.ContentLength == 5, "test request")
	check(http.DetectContentType([]byte("<!DOCTYPE html><html>")) == "text/html; charset=utf-8", "content sniffing")
	passed("PASS httptest request/response and MIME sniffing")

	clientConn, peer := net.Pipe()
	release := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		defer peer.Close()
		_, err := http.ReadRequest(bufio.NewReader(peer))
		if err != nil {
			return
		}
		io.WriteString(peer, "HTTP/1.1 200 OK\r\nContent-Length: 14\r\nContent-Type: text/event-stream\r\n\r\n")
		io.WriteString(peer, "data: first\n\n")
		<-release
		io.WriteString(peer, "!")
	}()
	transport := &http.Transport{DisableKeepAlives: true,
		DialContext: func(context.Context, string, string) (net.Conn, error) { return clientConn, nil }}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, err = client.Get("http://stream.example/")
	check(err == nil, "HTTP returns before EOF")
	first := make([]byte, 13)
	_, err = io.ReadFull(response.Body, first)
	check(err == nil && string(first) == "data: first\n\n", "first SSE event before peer completes")
	close(release)
	body, err = io.ReadAll(response.Body)
	check(err == nil && string(body) == "!", "stream remainder")
	response.Body.Close()
	<-finished
	transport.CloseIdleConnections()
	passed("PASS incremental HTTP response body")

	left, right := net.Pipe()
	left.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	_, err = left.Read(make([]byte, 1))
	check(err != nil && strings.Contains(err.Error(), os.ErrDeadlineExceeded.Error()), "pipe deadline")
	left.Close()
	right.Close()
	passed("PASS net.Pipe deadline")
	regs := kos.SyscallRegs{EAX: 75, EBX: 0, ECX: 0xffff, EDX: 1}
	kos.SyscallRaw(&regs)
	fmt.Printf("invalid socket probe: eax=%d ebx=%d\n", int32(regs.EAX), regs.EBX)
	check(int32(regs.EAX) == -1, "native socket syscall reports invalid domain")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	check(err == nil, "TCP listen")
	tcpDone := make(chan struct{})
	go func() {
		defer close(tcpDone)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		io.WriteString(conn, "hello")
		conn.Close()
	}()
	conn, err := net.Dial("tcp4", listener.Addr().String())
	check(err == nil, "TCP dial")
	passed("PASS native TCP connection")
	conn.SetReadDeadline(time.Now().Add(time.Second))
	data := make([]byte, 5)
	_, err = io.ReadFull(conn, data)
	check(err == nil && string(data) == "hello", "native TCP read/write")
	conn.Close()
	listener.Close()
	<-tcpDone
	passed("PASS native TCP read and write")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Target", "KolibriOS")
		io.WriteString(w, "loopback works")
	}))
	fmt.Printf("local server: %s\n", server.URL)
	client = &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}}
	traceCalls := 0
	traceContext := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { traceCalls++ },
	})
	traceContext = httptrace.WithClientTrace(traceContext, &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { traceCalls++ },
	})
	localRequest, err := http.NewRequestWithContext(traceContext, "GET", server.URL, nil)
	check(err == nil, "traced local request")
	response, err = client.Do(localRequest)
	check(err == nil, "local HTTP server request")
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	check(err == nil && string(body) == "loopback works" && response.Header.Get("X-Target") == "KolibriOS", "local HTTP response")
	check(traceCalls == 2, "composed upstream HTTP trace hooks")
	passed("PASS loopback HTTP request and response")
	server.Close()
	passed("PASS upstream HTTP server on KolibriOS loopback")
	_, err = tls.X509KeyPair(testcert.LocalhostCert, testcert.LocalhostKey)
	check(err == nil, "TLS test key pair")
	passed("PASS TLS certificate and key parsing")
	tlsServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "verified TLS")
	}))
	tlsServer.StartTLS()
	tlsClient := tlsServer.Client()
	// Software RSA and TLS under emulation need a wider end-to-end budget.
	tlsClient.Timeout = 30 * time.Second
	tlsContext := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		TLSHandshakeStart: func() { passed("TLS handshake started") },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err != nil {
				passed("TLS handshake error: " + err.Error())
			} else {
				passed("PASS TLS handshake completed")
			}
		},
	})
	tlsRequest, err := http.NewRequestWithContext(tlsContext, "GET", tlsServer.URL, nil)
	check(err == nil, "TLS request creation")
	response, err = tlsClient.Do(tlsRequest)
	if err != nil {
		panic(err)
	}
	passed("PASS HTTPS response headers")
	check(err == nil, "HTTPS with trusted test certificate")
	firstTLS := make([]byte, 1)
	_, err = io.ReadFull(response.Body, firstTLS)
	check(err == nil && string(firstTLS) == "v", "first HTTPS body byte")
	passed("PASS first HTTPS body byte")
	body, err = io.ReadAll(response.Body)
	body = append(firstTLS, body...)
	response.Body.Close()
	check(err == nil && string(body) == "verified TLS" && response.TLS != nil && len(response.TLS.VerifiedChains) != 0, "TLS certificate verification")
	passed("PASS HTTPS response and certificate chain")
	tlsServer.Close()
	passed("PASS verified HTTPS and TLS server shutdown")
	fmt.Println("OPENCODE_HTTP_PASS")
	kos.DebugString("OPENCODE_HTTP_PASS")
	kos.SleepSeconds(30)
}
