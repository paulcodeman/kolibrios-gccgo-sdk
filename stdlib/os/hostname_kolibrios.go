//go:build kolibrios && gccgo

package os

import "syscall"

func hostname() (string, error) {
	// The public kernel ABI has no machine-hostname query. An error is
	// supported by the upstream API and lets gRPC skip host-name filtering.
	return "", NewSyscallError("hostname", syscall.ENOTSUP)
}
