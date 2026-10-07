// Target execution checks for the upstream packages imported for OpenCode.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptrace"
	"net/netip"
	"net/textproto"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"text/template"

	"kos"
)

type payload struct {
	Name string
	IDs  []int
	Tags map[string]string
}

func check(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode upstream stdlib test")
	if !ok {
		return
	}
	defer console.Close()
	defer func() {
		if failure := recover(); failure != nil {
			fmt.Printf("OPENCODE_STDLIB_FAIL: %v\n", failure)
			kos.SleepSeconds(20)
		}
	}()
	check(sameInterface(nil, nil), "nil interface equality")
	check(sameInterface(os.ErrInvalid, os.ErrInvalid), "mixed interface equality")
	check(!sameInterface(os.ErrInvalid, nil) && !sameInterface(nil, os.ErrInvalid), "mixed nil interface inequality")
	check(!sameInterface(os.ErrInvalid, "invalid argument"), "mixed interface type mismatch")
	check(namedMapOperations(), "named map allocation and conversion")
	var buf bytes.Buffer
	flags := flag.NewFlagSet("smoke", flag.ContinueOnError)
	flags.SetOutput(&buf)
	count := flags.Int("count", 0, "count")
	check(flags.Parse([]string{"-count=7"}) == nil && *count == 7, "flag parse")
	check(flags.Parse([]string{"-count=bad"}) != nil, "flag invalid integer")
	check(strings.Contains(buf.String(), "invalid value"), "flag error output")
	fmt.Println("PASS flag")

	buf.Reset()
	want := payload{Name: "KolibriOS", IDs: []int{3, 8}, Tags: map[string]string{"target": "386"}}
	check(gob.NewEncoder(&buf).Encode(want) == nil, "gob encode")
	var got payload
	check(gob.NewDecoder(&buf).Decode(&got) == nil, "gob decode")
	check(got.Name == want.Name && len(got.IDs) == 2 && got.IDs[1] == 8 && got.Tags["target"] == "386", "gob round trip")
	fmt.Println("PASS encoding/gob")

	tmpl, err := template.New("list").Parse("{{range .}}{{.}};{{end}}")
	check(err == nil, "template parse")
	buf.Reset()
	check(tmpl.Execute(&buf, []string{"alpha", "beta"}) == nil && buf.String() == "alpha;beta;", "template execute")
	check(reflectedReceive(), "reflect channel receive")
	match, err := filepath.Match("*.go", "main.go")
	check(err == nil && match, "filepath match")
	fmt.Println("PASS text/template and reflect.Recv")

	ip := netip.MustParseAddr("fe80::1%kolibri")
	check(ip.String() == "fe80::1%kolibri" && ip == netip.MustParseAddr("fe80::1%kolibri"), "netip zone interning")
	check(netip.MustParsePrefix("10.2.3.4/16").Masked().String() == "10.2.0.0/16", "netip prefix")
	check(netip.MustParseAddr("::ffff:192.0.2.1").Unmap().String() == "192.0.2.1", "netip unmap")
	fmt.Println("PASS net/netip")

	reader := textproto.NewReader(bufio.NewReader(strings.NewReader("x-name: one\r\nX-Name: two\r\n\r\n")))
	header, err := reader.ReadMIMEHeader()
	check(err == nil && len(header["X-Name"]) == 2, "MIME header")
	trace := &httptrace.ClientTrace{}
	ctx := httptrace.WithClientTrace(context.Background(), trace)
	check(httptrace.ContextClientTrace(ctx) == trace, "httptrace context")
	fmt.Println("PASS net/textproto and httptrace context")
	buf.Reset()
	n, err := io.CopyN(&buf, strings.NewReader("short"), 9)
	check(n == 5 && err == io.EOF && buf.String() == "short", "CopyN short source returns EOF")
	buf.Reset()
	n, err = io.CopyN(&buf, strings.NewReader("longer"), 3)
	check(n == 3 && err == nil && buf.String() == "lon", "CopyN exact limit")

	buf.Reset()
	writer := multipart.NewWriter(&buf)
	check(writer.SetBoundary("kolibri-boundary") == nil, "multipart boundary")
	check(writer.WriteField("message", "hello") == nil, "multipart field")
	part, err := writer.CreateFormFile("upload", "sample.txt")
	check(err == nil, "multipart file part")
	_, err = io.WriteString(part, "file-data")
	check(err == nil && writer.Close() == nil, "multipart close")
	form, err := multipart.NewReader(bytes.NewReader(buf.Bytes()), writer.Boundary()).ReadForm(1024)
	check(err == nil, "multipart read form")
	check(form.Value["message"][0] == "hello", "multipart value")
	file, err := form.File["upload"][0].Open()
	check(err == nil, "multipart in-memory file")
	data, err := io.ReadAll(file)
	check(err == nil && string(data) == "file-data", "multipart file contents")
	file.Close()
	check(form.RemoveAll() == nil, "multipart cleanup")
	_, err = multipart.NewReader(bytes.NewReader(buf.Bytes()), writer.Boundary()).ReadForm(1)
	check(err != nil && strings.Contains(err.Error(), "exclusive file creation"), "multipart disk spill must fail explicitly")
	fmt.Println("PASS mime/multipart in memory")

	section := io.NewSectionReader(strings.NewReader("0123456789"), 2, 4)
	data, err = io.ReadAll(section)
	check(err == nil && string(data) == "2345", "section bounds")
	check(exclusiveCreateRejected(), "exclusive creation must fail explicitly")
	check(signalRegistrationRejected(), "signal registration must fail explicitly")
	fmt.Println("PASS explicit platform limits and io.SectionReader")

	info, err := debug.ParseBuildInfo("path\texample/app\nmod\texample/app\tv1.2.3\th1:test\n")
	check(err == nil && info.Path == "example/app" && info.Main.Version == "v1.2.3", "build info parsing")
	_, available := debug.ReadBuildInfo()
	check(!available, "SDK does not embed cmd/go module metadata")
	stack := string(debug.Stack())
	check(strings.Count(stack, "0x") >= 2, "real stack program counters")
	fmt.Println("PASS runtime/debug parsing and stack unwinding")
	fmt.Println("OPENCODE_STDLIB_PASS")
	kos.DebugString("OPENCODE_STDLIB_PASS")
	kos.SleepSeconds(20)
}

func sameInterface(err error, value any) bool { return err == value }

type stringLists map[string][]string

func namedMapOperations() bool {
	m := make(stringLists, 2)
	m["key"] = []string{"first"}
	n := map[string][]string(m)
	n["key"] = append(n["key"], "second")
	n["other"] = []string{"third"}
	if len(m) != 2 || len(m["key"]) != 2 || m["key"][1] != "second" {
		return false
	}
	delete(m, "other")
	return len(n) == 1
}

func reflectedReceive() bool {
	c := make(chan int, 1)
	c <- 42
	close(c)
	v, ok := reflect.ValueOf(c).Recv()
	if !ok || v.Int() != 42 {
		return false
	}
	v, ok = reflect.ValueOf(c).Recv()
	return !ok && v.Int() == 0
}

func exclusiveCreateRejected() bool {
	f, err := os.CreateTemp("/rd/1", "smoke-*")
	return f == nil && err != nil && strings.Contains(err.Error(), "exclusive file creation")
}

func signalRegistrationRejected() (rejected bool) {
	defer func() {
		if value := recover(); value != nil {
			message, ok := value.(string)
			rejected = ok && strings.Contains(message, "signal delivery is unsupported")
		}
	}()
	signal.Notify(make(chan os.Signal, 1), os.Interrupt)
	return false
}
