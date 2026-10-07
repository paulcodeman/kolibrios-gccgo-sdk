package net

import (
	"errors"
	"os"
	"time"
)

var ErrClosed = errors.New("use of closed network connection")

// Addr represents a network end point address.
type Addr interface {
	Network() string
	String() string
}

// Conn is a generic stream-oriented network connection.
type Conn interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
	LocalAddr() Addr
	RemoteAddr() Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// Listener is a generic network listener for stream-oriented protocols.
type Listener interface {
	Accept() (Conn, error)
	Close() error
	Addr() Addr
}

// Error represents a network error.
type Error interface {
	error
	Timeout() bool
	Temporary() bool
}

// UnknownNetworkError is returned for unsupported network types.
type UnknownNetworkError string

func (e UnknownNetworkError) Error() string {
	return "unknown network " + string(e)
}

// OpError is the error type usually returned by I/O operations.
type OpError struct {
	Source Addr
	Op     string
	Net    string
	Addr   Addr
	Err    error
}

func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}
	s := e.Op
	if e.Net != "" {
		s += " " + e.Net
	}
	if e.Source != nil {
		s += " " + e.Source.String()
	}
	if e.Addr != nil {
		if e.Source != nil {
			s += "->"
		} else {
			s += " "
		}
		s += e.Addr.String()
	}
	s += ": " + e.Err.Error()
	return s
}

func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type timeout interface {
	Timeout() bool
}

func (e *OpError) Timeout() bool {
	if ne, ok := e.Err.(*os.SyscallError); ok {
		t, ok := ne.Err.(timeout)
		return ok && t.Timeout()
	}
	t, ok := e.Err.(timeout)
	return ok && t.Timeout()
}

type temporary interface {
	Temporary() bool
}

func (e *OpError) Temporary() bool {
	// Treat ECONNRESET and ECONNABORTED as temporary errors when
	// they come from calling accept. See issue 6163.
	if e.Op == "accept" && isConnError(e.Err) {
		return true
	}

	if ne, ok := e.Err.(*os.SyscallError); ok {
		t, ok := ne.Err.(temporary)
		return ok && t.Temporary()
	}
	t, ok := e.Err.(temporary)
	return ok && t.Temporary()
}

// KolibriOS errno values come from kernel/trunk/network/stack.inc.
func isConnError(err error) bool {
	e, ok := err.(*socketError)
	return ok && (e.code == 52 || e.code == 53)
}
