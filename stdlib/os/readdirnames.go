package os

// Readdirnames shares the directory cursor and error handling with Readdir.
func (file *File) Readdirnames(n int) ([]string, error) {
	entries, err := file.Readdir(n)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, err
}
