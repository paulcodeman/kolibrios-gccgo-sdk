package net

import "errors"

// The documented KolibriOS socket ABI has close but no half-close operation.
// Returning an error preserves the peer's read direction instead of silently
// substituting a full close for a requested write-side shutdown.
func (c *TCPConn) CloseWrite() error {
	if c == nil {
		return &OpError{Op: "close", Net: "tcp", Err: ErrClosed}
	}
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return &OpError{Op: "close", Net: "tcp", Err: ErrClosed}
	}
	return &OpError{Op: "close", Net: "tcp", Addr: c.raddr,
		Err: errors.New("TCP half-close is unsupported by the KolibriOS socket ABI")}
}
