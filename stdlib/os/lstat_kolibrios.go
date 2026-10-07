package os

// Lstat uses the same metadata operation as Stat: KolibriOS syscall 70/5 does
// not expose symbolic links or a separate link-following mode.
func Lstat(name string) (FileInfo, error) { return Stat(name) }
