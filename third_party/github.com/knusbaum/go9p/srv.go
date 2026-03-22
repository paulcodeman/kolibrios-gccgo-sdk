package go9p

import (
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"path/filepath"

	"9fans.net/go/plan9/client"
)

type readWriteCloser interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close()
}

type tcpPipe struct {
	net.Conn
}

func (p *tcpPipe) Close() {
	if p != nil && p.Conn != nil {
		_ = p.Conn.Close()
	}
}

func postfd(name string) (readWriteCloser, *os.File, error) {
	ns := client.Namespace()
	if err := os.MkdirAll(ns, 0700); err != nil {
		return nil, nil, err
	}
	registryPath := filepath.Join(ns, name)
	listener, addr, err := listenService(name)
	if err != nil {
		return nil, nil, err
	}
	defer listener.Close()
	if err := os.WriteFile(registryPath, []byte(addr), 0600); err != nil {
		return nil, nil, err
	}
	defer os.Remove(registryPath)
	conn, err := listener.Accept()
	if err != nil {
		return nil, nil, fmt.Errorf("accept service %s: %w", name, err)
	}
	return &tcpPipe{Conn: conn}, nil, nil
}

func listenService(name string) (net.Listener, string, error) {
	start := servicePort(name)
	for i := 0; i < 64; i++ {
		port := start + i
		listenAddr := fmt.Sprintf("0.0.0.0:%d", port)
		publicAddr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", listenAddr)
		if err == nil {
			return listener, publicAddr, nil
		}
	}
	return nil, "", fmt.Errorf("listen service %s: no free port in range", name)
}

func servicePort(name string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return 20000 + int(h.Sum32()%20000)
}
