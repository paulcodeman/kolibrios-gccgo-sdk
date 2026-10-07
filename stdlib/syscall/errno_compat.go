// Go errno compatibility values from KolibriOS newlib sys/errno.h.
package syscall

const (
	ENOMEM Errno = 12
	EACCES Errno = 13
	EAGAIN Errno = 11
	EEXIST Errno = 17
	ELOOP Errno = 92
	ENAMETOOLONG Errno = 91
	ENOTEMPTY Errno = 90
	ENOTSOCK Errno = 108
	ENOTSUP Errno = 134
	EOPNOTSUPP Errno = 95
	EROFS Errno = 30
	ETIMEDOUT Errno = 116
	EWOULDBLOCK = EAGAIN
)
