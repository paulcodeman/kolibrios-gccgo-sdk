//go:build !kolibrios
// +build !kolibrios

package oksvg

import (
	"io"

	"golang.org/x/net/html/charset"
)

func charsetReader(label string, input io.Reader) (io.Reader, error) {
	return charset.NewReaderLabel(label, input)
}
