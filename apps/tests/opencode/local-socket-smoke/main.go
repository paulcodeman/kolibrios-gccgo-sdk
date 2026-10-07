package main

import (
	"kos"
	"time"
)

func main() {
	console, ok := kos.OpenConsole("Native local stream EOF")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	reader, writer, code := kos.CreateLocalSocketPair()
	if code != 0 {
		panic("documented socketpair")
	}
	defer kos.CloseLocalSocket(reader)
	payload := make([]byte, 65537)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	done := make(chan struct{})
	go func() {
		for offset := 0; offset < len(payload); {
			n, code := kos.WriteLocalSocket(writer, payload[offset:], kos.LocalSocketDontWait)
			if code != 0 && code != kos.LocalSocketWouldBlock {
				panic("local stream write")
			}
			if n > 0 {
				offset += n
			} else {
				time.Sleep(time.Millisecond)
			}
		}
		if kos.CloseLocalSocket(writer) != 0 {
			panic("local writer close")
		}
		close(done)
	}()
	var buffer [4096]byte
	offset := 0
	for {
		n, code := kos.ReadLocalSocket(reader, buffer[:], kos.LocalSocketDontWait)
		if code == kos.LocalSocketWouldBlock {
			time.Sleep(time.Millisecond)
			continue
		}
		if code != 0 {
			panic("local stream read")
		}
		if n == 0 {
			break
		}
		for _, value := range buffer[:n] {
			if offset >= len(payload) || value != payload[offset] {
				panic("local stream bytes")
			}
			offset++
		}
	}
	<-done
	if offset != len(payload) {
		panic("local buffered EOF lost bytes")
	}
	if n, code := kos.WriteLocalSocket(reader, []byte("closed"), kos.LocalSocketDontWait); n != -1 || code == 0 {
		panic("local write after peer close")
	}
	console.WriteString("Original kernel socketpair: 65537 exact bytes, drain, EOF and closed-peer error PASS\r\n")
	kos.DebugString("OPENCODE_LOCAL_SOCKET_PASS\n")
}
