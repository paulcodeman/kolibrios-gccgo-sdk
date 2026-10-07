package os

// Fd returns the SDK descriptor for consoles and pipes. KolibriOS opens
// ordinary files by path rather than an OS file handle; those File values
// cannot be passed to a descriptor syscall and return the invalid descriptor.
func (file *File) Fd() uintptr {
	if file == nil || file.closed || !file.fdBacked {
		return ^uintptr(0)
	}
	return uintptr(file.fd)
}
