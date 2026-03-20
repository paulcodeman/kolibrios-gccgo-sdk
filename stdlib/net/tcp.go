package net

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"kos"
)

const (
	syscallNetworkSocket = 75
	socketOpenOp         = 0
	socketCloseOp        = 1
	socketBindOp         = 2
	socketListenOp       = 3
	socketConnectOp      = 4
	socketAcceptOp       = 5
	socketSendOp         = 6
	socketReceiveOp      = 7

	socketFlagDontWait   = 0x40
	socketErrWouldBlock  = 11
	socketErrWouldBlock2 = 6
)

type socketError struct {
	op   string
	code uint32
}

func (e *socketError) Error() string {
	return fmt.Sprintf("socket %s: error %d", e.op, e.code)
}

func (e *socketError) Timeout() bool { return false }

func (e *socketError) Temporary() bool {
	return e != nil && (e.code == socketErrWouldBlock || e.code == socketErrWouldBlock2)
}

// TCPAddr represents the address of a TCP end point.
type TCPAddr struct {
	IP   IP
	Port int
	Zone string
}

func (a *TCPAddr) Network() string { return "tcp" }

func (a *TCPAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	host := ""
	if len(a.IP) > 0 {
		host = a.IP.String()
	}
	if host == "" {
		host = IPv4zero.String()
	}
	return JoinHostPort(host, strconv.Itoa(a.Port))
}

// UnixAddr represents the address of a Unix domain socket end point.
type UnixAddr struct {
	Name string
	Net  string
}

func (a *UnixAddr) Network() string {
	if a == nil {
		return ""
	}
	if a.Net != "" {
		return a.Net
	}
	return "unix"
}

func (a *UnixAddr) String() string {
	if a == nil {
		return "<nil>"
	}
	return a.Name
}

// TCPConn implements a TCP connection using KolibriOS socket syscalls.
type TCPConn struct {
	fd     int
	laddr  *TCPAddr
	raddr  *TCPAddr
	closed bool
}

// TCPListener implements a TCP listener using KolibriOS socket syscalls.
type TCPListener struct {
	fd     int
	addr   *TCPAddr
	mu     sync.Mutex
	closed bool
}

func (c *TCPConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	spins := 0
	for {
		n, err := socketRecv(c.fd, b, socketFlagDontWait)
		if err == nil {
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}
		if !isWouldBlock(err) {
			return n, err
		}
		spins = yieldSocketWait(spins)
	}
}

func (c *TCPConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	written := 0
	spins := 0
	for written < len(b) {
		n, err := socketSend(c.fd, b[written:], socketFlagDontWait)
		if err != nil {
			if isWouldBlock(err) {
				spins = yieldSocketWait(spins)
				continue
			}
			return written, err
		}
		if n == 0 {
			return written, io.ErrShortWrite
		}
		spins = 0
		written += n
	}
	return written, nil
}

func (c *TCPConn) Close() error {
	if c == nil || c.closed {
		return nil
	}
	c.closed = true
	return socketCloseCall(c.fd)
}

func (c *TCPConn) LocalAddr() Addr {
	if c == nil {
		return nil
	}
	return c.laddr
}

func (c *TCPConn) RemoteAddr() Addr {
	if c == nil {
		return nil
	}
	return c.raddr
}

func (c *TCPConn) SetDeadline(t time.Time) error      { return nil }
func (c *TCPConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *TCPConn) SetWriteDeadline(t time.Time) error { return nil }

func (l *TCPListener) Accept() (Conn, error) {
	if l == nil {
		return nil, &OpError{Op: "accept", Net: "tcp", Err: ErrClosed}
	}

	for {
		l.mu.Lock()
		closed := l.closed
		fd := l.fd
		addr := l.addr
		l.mu.Unlock()
		if closed {
			return nil, &OpError{Op: "accept", Net: "tcp", Addr: addr, Err: ErrClosed}
		}

		var sa sockaddr
		acceptedFD, remoteAddr, err := socketAcceptCall(fd, &sa)
		if err == nil {
			return &TCPConn{
				fd:    acceptedFD,
				laddr: cloneTCPAddr(addr),
				raddr: remoteAddr,
			}, nil
		}

		l.mu.Lock()
		closed = l.closed
		addr = l.addr
		l.mu.Unlock()
		if closed {
			return nil, &OpError{Op: "accept", Net: "tcp", Addr: addr, Err: ErrClosed}
		}
		return nil, &OpError{Op: "accept", Net: "tcp", Addr: addr, Err: err}
	}
}

func (l *TCPListener) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	fd := l.fd
	l.mu.Unlock()

	return socketCloseCall(fd)
}

func (l *TCPListener) Addr() Addr {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	addr := cloneTCPAddr(l.addr)
	l.mu.Unlock()
	return addr
}

// Dial connects to the address on the named network.
func Dial(network, address string) (Conn, error) {
	return DialTimeout(network, address, 0)
}

// DialTimeout connects to the address on the named network with a timeout.
func DialTimeout(network, address string, timeout time.Duration) (Conn, error) {
	_ = timeout
	if network == "" {
		network = "tcp"
	}
	switch network {
	case "tcp", "tcp4":
	default:
		return nil, UnknownNetworkError(network)
	}

	raddr, err := ResolveTCPAddr(network, address)
	if err != nil {
		return nil, err
	}
	if raddr == nil || raddr.IP == nil {
		return nil, &OpError{Op: "dial", Net: network, Addr: raddr, Err: errors.New("missing address")}
	}

	conn, err := dialTCPAddr(raddr)
	if err != nil {
		return nil, &OpError{Op: "dial", Net: network, Addr: raddr, Err: err}
	}
	return conn, nil
}

func listenTCP(network, address string) (Listener, error) {
	if network == "" {
		network = "tcp"
	}
	switch network {
	case "tcp", "tcp4":
	default:
		return nil, UnknownNetworkError(network)
	}

	laddr, err := resolveTCPListenAddr(network, address)
	if err != nil {
		return nil, err
	}
	if laddr == nil {
		return nil, &OpError{Op: "listen", Net: network, Err: errors.New("missing address")}
	}
	if laddr.IP == nil {
		laddr.IP = IPv4zero
	}

	listener, err := listenTCPAddr(laddr)
	if err != nil {
		return nil, &OpError{Op: "listen", Net: network, Addr: laddr, Err: err}
	}
	return listener, nil
}

// ResolveTCPAddr returns a TCP address from a host:port string.
func ResolveTCPAddr(network, address string) (*TCPAddr, error) {
	if network == "" {
		network = "tcp"
	}
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, UnknownNetworkError(network)
	}

	host, portStr, err := SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, &AddrError{Err: "invalid port", Addr: address}
	}

	var ip IP
	if host != "" {
		ip = ParseIP(host)
		if ip == nil {
			addrs, lookupErr := LookupHost(host)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if len(addrs) == 0 {
				return nil, &AddrError{Err: "no such host", Addr: host}
			}
			for _, addr := range addrs {
				ip = ParseIP(addr)
				if ip != nil {
					break
				}
			}
		}
		if ip == nil {
			return nil, &AddrError{Err: "invalid address", Addr: host}
		}
	}

	return &TCPAddr{IP: ip, Port: port}, nil
}

func resolveTCPListenAddr(network, address string) (*TCPAddr, error) {
	if network == "" {
		network = "tcp"
	}
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, UnknownNetworkError(network)
	}

	host, portStr, err := SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return nil, &AddrError{Err: "invalid port", Addr: address}
	}

	if host == "" {
		return &TCPAddr{IP: IPv4zero, Port: port}, nil
	}

	ip := ParseIP(host)
	if ip == nil {
		addrs, lookupErr := LookupHost(host)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if len(addrs) == 0 {
			return nil, &AddrError{Err: "no such host", Addr: host}
		}
		for _, addr := range addrs {
			ip = ParseIP(addr)
			if ip != nil {
				break
			}
		}
	}
	if ip == nil {
		return nil, &AddrError{Err: "invalid address", Addr: host}
	}

	return &TCPAddr{IP: ip, Port: port}, nil
}

func dialTCPAddr(raddr *TCPAddr) (*TCPConn, error) {
	if raddr == nil || raddr.IP == nil {
		return nil, errors.New("missing remote address")
	}
	ip4 := raddr.IP.To4()
	if ip4 == nil {
		return nil, errors.New("only IPv4 addresses are supported")
	}

	fd, err := socketOpenCall(uint32(kos.NetworkFamilyIPv4), uint32(kos.NetworkSockStream), 0)
	if err != nil {
		return nil, err
	}

	sa := sockaddrIPv4(ip4, raddr.Port)
	if err := socketConnectCall(fd, &sa); err != nil {
		_ = socketCloseCall(fd)
		return nil, err
	}

	laddr := &TCPAddr{IP: IPv4zero, Port: 0}
	return &TCPConn{fd: fd, laddr: laddr, raddr: raddr}, nil
}

func listenTCPAddr(laddr *TCPAddr) (*TCPListener, error) {
	if laddr == nil {
		return nil, errors.New("missing local address")
	}

	ip4 := laddr.IP.To4()
	if laddr.IP == nil {
		ip4 = IPv4zero.To4()
	}
	if ip4 == nil {
		return nil, errors.New("only IPv4 addresses are supported")
	}

	fd, err := socketOpenCall(uint32(kos.NetworkFamilyIPv4), uint32(kos.NetworkSockStream), 0)
	if err != nil {
		return nil, err
	}

	sa := sockaddrIPv4(ip4, laddr.Port)
	if err := socketBindCall(fd, &sa); err != nil {
		_ = socketCloseCall(fd)
		return nil, err
	}
	if err := socketListenCall(fd, 16); err != nil {
		_ = socketCloseCall(fd)
		return nil, err
	}

	return &TCPListener{
		fd:   fd,
		addr: cloneTCPAddr(laddr),
	}, nil
}

type sockaddr struct {
	Family uint16
	Data   [14]byte
}

func sockaddrIPv4(ip IP, port int) sockaddr {
	var sa sockaddr
	sa.Family = uint16(kos.NetworkFamilyIPv4)
	binary.BigEndian.PutUint16(sa.Data[0:2], uint16(port))
	copy(sa.Data[2:6], ip.To4())
	return sa
}

func tcpAddrFromSockaddr(sa *sockaddr) *TCPAddr {
	if sa == nil {
		return nil
	}
	if sa.Family != uint16(kos.NetworkFamilyIPv4) {
		return nil
	}

	ip := IPv4(sa.Data[2], sa.Data[3], sa.Data[4], sa.Data[5])
	port := int(binary.BigEndian.Uint16(sa.Data[0:2]))
	return &TCPAddr{IP: ip, Port: port}
}

func cloneTCPAddr(addr *TCPAddr) *TCPAddr {
	if addr == nil {
		return nil
	}

	clone := *addr
	if addr.IP != nil {
		clone.IP = append(IP(nil), addr.IP...)
	}
	return &clone
}

func socketOpenCall(domain, sockType, proto uint32) (int, error) {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketOpenOp,
		ECX: domain,
		EDX: sockType,
		ESI: proto,
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return -1, &socketError{op: "open", code: regs.EBX}
	}
	return int(int32(regs.EAX)), nil
}

func socketCloseCall(fd int) error {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketCloseOp,
		ECX: uint32(fd),
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return &socketError{op: "close", code: regs.EBX}
	}
	return nil
}

func socketBindCall(fd int, addr *sockaddr) error {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketBindOp,
		ECX: uint32(fd),
		EDX: pointerValue(unsafe.Pointer(addr)),
		ESI: uint32(unsafe.Sizeof(*addr)),
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return &socketError{op: "bind", code: regs.EBX}
	}
	return nil
}

func socketListenCall(fd int, backlog uint32) error {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketListenOp,
		ECX: uint32(fd),
		EDX: backlog,
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return &socketError{op: "listen", code: regs.EBX}
	}
	return nil
}

func socketConnectCall(fd int, addr *sockaddr) error {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketConnectOp,
		ECX: uint32(fd),
		EDX: pointerValue(unsafe.Pointer(addr)),
		ESI: uint32(unsafe.Sizeof(*addr)),
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return &socketError{op: "connect", code: regs.EBX}
	}
	return nil
}

func socketAcceptCall(fd int, addr *sockaddr) (int, *TCPAddr, error) {
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketAcceptOp,
		ECX: uint32(fd),
		EDX: pointerValue(unsafe.Pointer(addr)),
		ESI: uint32(unsafe.Sizeof(*addr)),
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return 0, nil, &socketError{op: "accept", code: regs.EBX}
	}
	return int(int32(regs.EAX)), tcpAddrFromSockaddr(addr), nil
}

func socketSend(fd int, data []byte, flags uint32) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketSendOp,
		ECX: uint32(fd),
		EDX: pointerValue(unsafe.Pointer(&data[0])),
		ESI: uint32(len(data)),
		EDI: flags,
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return 0, &socketError{op: "send", code: regs.EBX}
	}
	return int(int32(regs.EAX)), nil
}

func socketRecv(fd int, buf []byte, flags uint32) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	regs := kos.SyscallRegs{
		EAX: syscallNetworkSocket,
		EBX: socketReceiveOp,
		ECX: uint32(fd),
		EDX: pointerValue(unsafe.Pointer(&buf[0])),
		ESI: uint32(len(buf)),
		EDI: flags,
	}
	kos.SyscallRaw(&regs)
	if int32(regs.EAX) < 0 {
		return 0, &socketError{op: "recv", code: regs.EBX}
	}
	return int(int32(regs.EAX)), nil
}

func isWouldBlock(err error) bool {
	socketErr, ok := err.(*socketError)
	return ok && (socketErr.code == socketErrWouldBlock || socketErr.code == socketErrWouldBlock2)
}

func yieldSocketWait(spins int) int {
	runtime.Gosched()
	switch {
	case spins < 4:
	case spins < 20:
		kos.SleepCentiseconds(1)
	default:
		kos.SleepCentiseconds(2)
	}
	return spins + 1
}

func pointerValue(ptr unsafe.Pointer) uint32 {
	return uint32(uintptr(ptr))
}
