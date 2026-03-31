//go:build kolibrios
// +build kolibrios

package oksvg

import (
	"errors"
	"io"
	"strings"
)

func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "", "utf-8", "utf8":
		return input, nil
	default:
		return nil, errors.New("oksvg: unsupported XML charset on KolibriOS: " + label)
	}
}
