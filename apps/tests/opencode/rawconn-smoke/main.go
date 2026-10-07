package main

import (
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"time"
	"unsafe"

	"kos"
)

func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}

func transfer(fd uintptr, data []byte, write bool) (int, uint32) {
	op := uint32(7)
	if write {
		op = 6
	}
	regs := kos.SyscallRegs{EAX: 75, EBX: op, ECX: uint32(fd), EDX: uint32(uintptr(unsafe.Pointer(&data[0]))), ESI: uint32(len(data)), EDI: 0x40}
	kos.SyscallRaw(&regs)
	runtime.KeepAlive(data)
	return int(int32(regs.EAX)), regs.EBX
}

func main() {
	console, ok := kos.OpenConsole("OpenCode raw socket port test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_RAWCONN_START")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		accepted <- c
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		panic(err)
	}
	defer client.Close()
	server := <-accepted
	defer server.Close()
	tcp := client.(*net.TCPConn)
	raw, err := tcp.SyscallConn()
	if err != nil {
		panic(err)
	}
	called := false
	check(raw.Control(func(fd uintptr) { called = true }) == nil && called, "native raw Control")
	sent := []byte("native raw bytes")
	check(raw.Write(func(fd uintptr) bool {
		n, code := transfer(fd, sent, true)
		if n < 0 {
			check(code == 6, "raw send error")
			return false
		}
		check(n == len(sent), "raw send length")
		return true
	}) == nil, "native raw Write")
	buffer := make([]byte, len(sent))
	_, err = io.ReadFull(server, buffer)
	check(err == nil && string(buffer) == string(sent), "raw send delivered bytes")
	check(raw.Control(func(uintptr) {}) == nil, "Control reusable")
	go func() {
		time.Sleep(30 * time.Millisecond)
		_, err := server.Write([]byte("reply"))
		if err != nil {
			panic(err)
		}
	}()
	buffer = make([]byte, 5)
	attempts := 0
	check(raw.Read(func(fd uintptr) bool {
		attempts++
		n, code := transfer(fd, buffer, false)
		if n < 0 {
			check(code == 6, "raw recv error")
			return false
		}
		check(n == 5 && string(buffer) == "reply", "native raw received bytes")
		return true
	}) == nil && attempts > 1, "raw Read polling")
	check(tcp.SetReadDeadline(time.Now().Add(30*time.Millisecond)) == nil, "deadline")
	err = raw.Read(func(uintptr) bool { return false })
	check(errors.Is(err, os.ErrDeadlineExceeded), "raw Read deadline")
	check(tcp.SetReadDeadline(time.Time{}) == nil, "reset deadline")
	started := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- raw.Read(func(uintptr) bool {
			select {
			case started <- struct{}{}:
			default:
			}
			return false
		})
	}()
	<-started
	check(tcp.Close() == nil, "Close interrupts raw wait")
	check(errors.Is(<-done, net.ErrClosed), "raw wait closed error")
	called = false
	err = raw.Control(func(uintptr) { called = true })
	check(errors.Is(err, net.ErrClosed) && !called, "closed handle callback suppressed")
	kos.DebugString("OPENCODE_RAWCONN_PASS")
	console.WriteString("OPENCODE_RAWCONN_PASS\n")
}
