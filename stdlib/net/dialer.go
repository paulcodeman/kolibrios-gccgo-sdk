package net

import (
	"context"
	"errors"
	"time"
)

// Dialer is a minimal KolibriOS net.Dialer implementation for client-side code.
type Dialer struct {
	Timeout  time.Duration
	Deadline time.Time
}

func (d *Dialer) Dial(network, address string) (Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (Conn, error) {
	if ctx == nil {
		panic("nil context")
	}

	timeout, err := d.effectiveTimeout(ctx, network)
	if err != nil {
		return nil, err
	}

	conn, err := DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, &OpError{Op: "dial", Net: network, Err: ctx.Err()}
	default:
		return conn, nil
	}
}

func (d *Dialer) effectiveTimeout(ctx context.Context, network string) (time.Duration, error) {
	timeout := d.Timeout

	if !d.Deadline.IsZero() {
		remaining := d.Deadline.Sub(time.Now())
		if remaining <= 0 {
			return 0, &OpError{Op: "dial", Net: network, Err: context.DeadlineExceeded}
		}
		if timeout == 0 || remaining < timeout {
			timeout = remaining
		}
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := deadline.Sub(time.Now())
		if remaining <= 0 {
			return 0, &OpError{Op: "dial", Net: network, Err: context.DeadlineExceeded}
		}
		if timeout == 0 || remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case <-ctx.Done():
		return 0, &OpError{Op: "dial", Net: network, Err: ctx.Err()}
	default:
		return timeout, nil
	}
}

func Listen(network, address string) (Listener, error) {
	return nil, &OpError{Op: "listen", Net: network, Err: errors.New("listen not supported")}
}
