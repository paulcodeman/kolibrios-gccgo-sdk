package net

import (
	"errors"
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
	Op   string
	Net  string
	Addr Addr
	Err  error
}

func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Addr != nil {
		return e.Op + " " + e.Net + " " + e.Addr.String() + ": " + e.Err.Error()
	}
	return e.Op + " " + e.Net + ": " + e.Err.Error()
}

func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *OpError) Timeout() bool {
	if e == nil {
		return false
	}
	if err, ok := e.Err.(Error); ok {
		return err.Timeout()
	}
	return false
}

func (e *OpError) Temporary() bool {
	if e == nil {
		return false
	}
	if err, ok := e.Err.(Error); ok {
		return err.Temporary()
	}
	return false
}
