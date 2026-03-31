package ioutil

import (
	"bytes"
	"io"
	"os"
	"sort"
)

var Discard io.Writer = io.Discard

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

func ReadDir(dirname string) ([]os.FileInfo, error) {
	file, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	list, err := file.Readdir(-1)
	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i int, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list, nil
}
