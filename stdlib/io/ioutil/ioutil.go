package ioutil

import (
	"bytes"
	"io"
	"os"
)

// ReadAll reads from r until EOF and returns the data.
func ReadAll(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	return buf.Bytes(), err
}

// ReadFile reads the named file and returns its contents.
func ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// WriteFile writes data to the named file, creating it if needed.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
