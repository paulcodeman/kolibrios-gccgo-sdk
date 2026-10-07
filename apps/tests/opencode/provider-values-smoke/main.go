package main

import (
	"bytes"
	"errors"
	"github.com/openai/openai-go/packages/param"
	"github.com/atotto/clipboard"
	"io"
	"io/fs"
	"kos"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

func check(ok bool, why string) {
	if !ok {
		kos.DebugString(why)
		panic(why)
	}
}

func must(err error) { if err != nil { kos.DebugString(err.Error()); panic(err) } }

func panics(f func()) (yes bool) {
	defer func() { yes = recover() != nil }()
	f()
	return
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "exclusive-child" {
		_, err := os.OpenFile("/hd0/1/exclusive-race.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		check(errors.Is(err, os.ErrExist), "child exclusive creation rejects existing file")
		must(os.WriteFile("/hd0/1/exclusive-child-state.txt", []byte("rejected"), 0600))
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "cwd-child" {
		wd, err := os.Getwd(); must(err)
		data, err := os.ReadFile("marker.txt"); must(err)
		must(os.WriteFile("/hd0/1/cwd-child.txt", []byte(wd+":"+string(data)), 0600))
		return
	}
	console, ok := kos.OpenConsole("OpenCode provider API test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_PROVIDER_VALUES_START")
	negativeZero := math.Float64frombits(1 << 63)
	check(reflect.ValueOf(negativeZero).IsZero(), "signed zero")
	check(reflect.ValueOf([2]float64{0, negativeZero}).IsZero(), "array zero")
	check(!reflect.ValueOf(math.NaN()).IsZero(), "NaN nonzero")
	check(!reflect.ValueOf([]int{}).IsZero() && reflect.ValueOf([]int(nil)).IsZero(), "nil and empty slices")
	check(reflect.Value{}.Equal(reflect.Value{}), "invalid equality")
	check(reflect.ValueOf([2]int{3, 4}).Equal(reflect.ValueOf([2]int{3, 4})), "array equality")
	check(!reflect.ValueOf(math.NaN()).Equal(reflect.ValueOf(math.NaN())), "NaN equality")
	check(panics(func() { reflect.ValueOf([]int{}).Equal(reflect.ValueOf([]int{})) }), "uncomparable equality panic")
	check(panics(func() { reflect.ValueOf([0]func(){}).Equal(reflect.ValueOf([0]func(){})) }), "empty uncomparable array panic")
	type record struct { private int }
	r := record{private: 7}
	field := reflect.ValueOf(&r).Elem().Field(0)
	check(!field.CanInterface(), "private field")
	accessible := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	accessible.SetInt(42)
	check(r.private == 42 && accessible.Equal(reflect.ValueOf(42)), "provider private-field decoder access")
	values := []int{1, 2}
	v := reflect.ValueOf(&values).Elem()
	v.Grow(8)
	check(len(values) == 2 && cap(values) >= 10 && values[1] == 2, "decoder slice growth")
	v.SetLen(3)
	v.Index(2).SetInt(42)
	check(values[2] == 42, "decoder appended value")
	check(param.IsOmitted(struct{ Count int }{}) && !param.IsOmitted(struct{ Count int }{1}), "original provider omitted fields")
	var buffer bytes.Buffer
	buffer.Grow(32)
	available := buffer.AvailableBuffer()
	check(len(available) == 0 && cap(available) == buffer.Available(), "encoder available buffer")
	available = strconv.AppendInt(available, 123, 10)
	_, err := buffer.Write(available)
	must(err)
	check(buffer.String() == "123", "encoder buffer alias append")
	check((90*time.Second).Minutes() == 1.5 && (90*time.Minute).Hours() == 1.5, "duration fractions")
	dns := &net.DNSError{Name: "example", Server: "server", Err: "temporary", IsTemporary: true, IsNotFound: true}
	check(dns.Temporary() && !dns.Timeout() && dns.IsNotFound, "DNS error classification")
	const filename = "/hd0/1/provider-values.txt"
	must(os.WriteFile(filename, []byte("native DirFS"), 0600))
	filesystem := os.DirFS("/hd0/1")
	data, err := fs.ReadFile(filesystem, "provider-values.txt")
	must(err)
	check(string(data) == "native DirFS", "native DirFS read")
	_, err = filesystem.Open("../escape")
	check(errors.Is(err, fs.ErrInvalid), "DirFS invalid relative path")
	file, err := filesystem.Open("provider-values.txt")
	must(err)
	_, ok = file.(io.ReadSeekCloser)
	check(ok, "ReadSeekCloser")
	must(file.Close())
	name, err := os.Executable()
	must(err)
	check(name == kos.LoaderPath(), "native executable loader path")
	check(time.Unix(1, 0).Compare(time.Unix(2, 0)) == -1, "time Compare before")
	check(time.Unix(1, 0).Compare(time.Unix(1, 0).In(time.FixedZone("offset", 3600))) == 0, "time Compare ignores location")
	for _, paths := range [][3]string{
		{"/a/b/c", "/a/d/e", "../../d/e"},
		{"/a", "/a", "."},
		{"a/b", "a/c", "../c"},
	} {
		relative, err := filepath.Rel(paths[0], paths[1])
		must(err)
		check(relative == paths[2], "lexical relative path")
	}
	_, err = filepath.Rel("relative", "/absolute")
	check(err != nil, "relative path incompatible roots")
	buffers := net.Buffers{[]byte("ab"), nil, []byte("cd")}
	var output bytes.Buffer
	n, err := buffers.WriteTo(&output)
	must(err)
	check(n == 4 && output.String() == "abcd" && len(buffers) == 0, "scatter buffers WriteTo and consume")
	buffers = net.Buffers{[]byte("ab"), []byte("cd")}
	var first [3]byte
	nn, err := buffers.Read(first[:])
	must(err)
	check(nn == 3 && string(first[:]) == "abc", "scatter buffers partial read")
	nn, err = buffers.Read(first[:])
	check(nn == 1 && first[0] == 'd' && err == io.EOF, "scatter buffers EOF")
	process, err := os.FindProcess(os.Getpid())
	must(err)
	must(process.Signal(syscall.Signal(0)))
	check(errors.Is(process.Signal(syscall.SIGTERM), syscall.ENOTSUP), "unsupported native POSIX signal is explicit")
	must(process.Release())
	check(process.Signal(syscall.Signal(0)) != nil, "released process invalid")
	cmd := exec.Command("/hd0/1/missing-provider-values-command.kex")
	writer, err := cmd.StdinPipe()
	must(err)
	_, err = cmd.StdinPipe()
	check(err != nil, "duplicate StdinPipe rejected")
	check(errors.Is(cmd.Start(), os.ErrNotExist), "missing process reports ErrNotExist")
	check(errors.Is(writer.Close(), os.ErrClosed), "failed Start closes parent pipe")
	const clipboardText = "OpenCode: Привет 世界"
	must(clipboard.WriteAll(clipboardText))
	pasted, err := clipboard.ReadAll()
	must(err)
	check(pasted == clipboardText, "native UTF-8 clipboard round trip")
	check(kos.ClipboardCopyTextWithEncoding(string([]byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2}), kos.ClipboardEncodingCP866) == kos.ClipboardOK, "native CP866 clipboard write")
	pasted, err = clipboard.ReadAll()
	must(err)
	check(pasted == "Привет", "native CP866 clipboard conversion")
	check(kos.ClipboardCopyTextWithEncoding(string([]byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2}), kos.ClipboardEncodingCP1251) == kos.ClipboardOK, "native CP1251 clipboard write")
	pasted, err = clipboard.ReadAll()
	must(err)
	check(pasted == "Привет", "native CP1251 clipboard conversion")
	runtime.GOMAXPROCS(2)
	oldDirectory, err := os.Getwd(); must(err)
	must(os.Mkdir("/hd0/1/cwd-one", 0700)); must(os.Mkdir("/hd0/1/cwd-two", 0700))
	must(os.Chdir("/hd0/1/cwd-one"))
	must(os.WriteFile("marker.txt", []byte("one"), 0600))
	opened, err := os.Open("marker.txt"); must(err)
	check(opened.Name()=="marker.txt", "open preserves original File.Name")
	must(os.Chdir("../cwd-two"))
	must(os.WriteFile("marker.txt", []byte("two"), 0600))
	data, err = io.ReadAll(opened); must(err); must(opened.Close())
	check(string(data)=="one", "opened file stays bound across Chdir")
	checkedDirectory := make(chan bool, 4)
	for i:=0;i<4;i++ { go func() {
		wd, err := os.Getwd(); check(err==nil && wd=="/hd0/1/cwd-two", "working directory shared across goroutines")
		data, err := os.ReadFile("marker.txt"); check(err==nil && string(data)=="two", "relative file shared across native threads")
		checkedDirectory <- true
	}() }
	for i:=0;i<4;i++ { <-checkedDirectory }
	child, status := kos.StartApplication(name, "cwd-child", false)
	check(status==kos.FileSystemOK && child>0,"working directory child launch")
	for kos.ThreadSlotByIdentifier(child)!=0 { time.Sleep(10*time.Millisecond) }
	data, err = os.ReadFile("/hd0/1/cwd-child.txt"); must(err)
	check(string(data)=="/hd0/1/cwd-two:two", "child inherits application working directory")
	check(errors.Is(os.Chdir("marker.txt"), syscall.ENOTDIR), "Chdir non-directory rejected")
	must(os.Chdir(oldDirectory))
	var before, allocated, collected runtime.MemStats
	runtime.ReadMemStats(&before)
	held := make([]*[4096]byte, 256)
	for i:=range held { held[i]=new([4096]byte); held[i][0]=byte(i) }
	runtime.ReadMemStats(&allocated)
	check(allocated.Alloc==allocated.HeapAlloc && allocated.Alloc>=before.Alloc+1<<20, "native live heap snapshot")
	check(allocated.TotalAlloc>=before.TotalAlloc+1<<20 && allocated.Mallocs>=before.Mallocs+256, "native allocation counters")
	check(allocated.Mallocs-allocated.Frees==allocated.HeapObjects && allocated.EnableGC, "native snapshot consistency")
	runtime.KeepAlive(held)
	held=nil
	runtime.GC()
	runtime.ReadMemStats(&collected)
	check(collected.NumGC>allocated.NumGC && collected.TotalAlloc>=allocated.TotalAlloc, "native GC count snapshot")
	results := make(chan error, 8)
	for i:=0;i<8;i++ { go func() {
		file, err := os.OpenFile("/hd0/1/exclusive-race.txt", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err==nil { _, err=file.WriteString("winner"); if closeErr:=file.Close(); err==nil { err=closeErr } }
		results <- err
	}() }
	winners:=0
	for i:=0;i<8;i++ { err:=<-results; if err==nil { winners++ } else { check(errors.Is(err,os.ErrExist), "exclusive loser returns ErrExist") } }
	check(winners==1, "atomic exclusive creation has one winner")
	data, err = os.ReadFile("/hd0/1/exclusive-race.txt"); must(err)
	check(string(data)=="winner", "exclusive losing writes do not truncate")
	child, status = kos.StartApplication(name, "exclusive-child", false)
	check(status==kos.FileSystemOK && child>0, "exclusive child launch")
	for kos.ThreadSlotByIdentifier(child)!=0 { time.Sleep(10*time.Millisecond) }
	data, err = os.ReadFile("/hd0/1/exclusive-child-state.txt"); must(err)
	check(string(data)=="rejected", "separate process exclusive creation")
	const dirName="/hd0/1/exclusive-directory"
	must(os.Mkdir(dirName, 0700))
	check(errors.Is(os.Mkdir(dirName, 0700),os.ErrExist), "Mkdir existing directory returns ErrExist")
	must(os.WriteFile(dirName+"/untouched.txt", []byte("safe"),0600))
	_, err = os.OpenFile(dirName,os.O_CREATE|os.O_EXCL|os.O_RDWR,0600)
	check(errors.Is(err,os.ErrExist), "exclusive file rejects existing directory")
	check(errors.Is(os.Mkdir("/hd0/1/exclusive-race.txt",0700),os.ErrExist), "exclusive directory rejects existing file")
	must(os.MkdirAll(dirName+"/nested/child",0700)); must(os.MkdirAll(dirName+"/nested/child",0700))
	temporary, err := os.CreateTemp("/hd0/1","temp-*.txt"); must(err)
	_, err = temporary.WriteString("temporary"); must(err); must(temporary.Close())
	data, err = os.ReadFile(temporary.Name()); must(err)
	check(string(data)=="temporary", "upstream CreateTemp native exclusive backend")
	temporaryDirectory, err := os.MkdirTemp("/hd0/1","temp-directory-*"); must(err)
	info, err := os.Stat(temporaryDirectory); must(err); check(info.IsDir(),"upstream MkdirTemp native exclusive backend")
	data, err = os.ReadFile(dirName+"/untouched.txt"); must(err); check(string(data)=="safe", "existing directory contents preserved")
	kos.DebugString("OPENCODE_PROVIDER_VALUES_PASS")
	console.WriteString("OPENCODE_PROVIDER_VALUES_PASS\n")
}
