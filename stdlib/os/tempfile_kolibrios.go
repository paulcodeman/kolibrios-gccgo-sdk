package os

// ErrExclusiveCreateUnsupported reports a filesystem operation that cannot be
// implemented atomically with the original KolibriOS syscall 70/2.
// Kernels built with the SDK exclusive-create extension support this mode;
// older kernels and other filesystems return this error without touching a path.
var ErrExclusiveCreateUnsupported error = &osError{text: "exclusive file creation is unsupported on KolibriOS"}

func IsPathSeparator(c uint8) bool { return c == '/' }
