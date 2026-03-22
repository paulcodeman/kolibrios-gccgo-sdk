// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package encoding defines interfaces and helpers for character set encoders.
package textencoding // import "golang.org/x/text/encoding"

import (
	"errors"
	"io"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// Encoding is a character set that can be encoded to and decoded from UTF-8.
type Encoding interface {
	NewDecoder() *Decoder
	NewEncoder() *Encoder
}

// Decoder decodes bytes to UTF-8.
type Decoder struct {
	transform.Transformer
}

// Encoder encodes UTF-8 bytes to an encoding.
type Encoder struct {
	transform.Transformer
}

// ErrInvalidUTF8 is returned when invalid UTF-8 is encountered.
var ErrInvalidUTF8 = errors.New("encoding: invalid UTF-8")

// RepertoireError indicates a rune is not representable by the encoding.
type RepertoireError struct {
	Rune rune
}

func (e RepertoireError) Error() string {
	if e.Rune == utf8.RuneError {
		return "encoding: rune error"
	}
	return "encoding: rune not in repertoire"
}

// Bytes returns the result of transforming b.
func (d *Decoder) Bytes(b []byte) ([]byte, error) {
	result, _, err := transform.Bytes(d, b)
	return result, err
}

// String returns the result of transforming s.
func (d *Decoder) String(s string) (string, error) {
	result, _, err := transform.String(d, s)
	return result, err
}

// Reader returns a reader that decodes from r.
func (d *Decoder) Reader(r io.Reader) io.Reader {
	return transform.NewReader(r, d)
}

// Writer returns a writer that decodes to w.
func (d *Decoder) Writer(w io.Writer) io.Writer {
	return transform.NewWriter(w, d)
}

// Bytes returns the result of transforming b.
func (e *Encoder) Bytes(b []byte) ([]byte, error) {
	result, _, err := transform.Bytes(e, b)
	return result, err
}

// String returns the result of transforming s.
func (e *Encoder) String(s string) (string, error) {
	result, _, err := transform.String(e, s)
	return result, err
}

// Reader returns a reader that encodes from r.
func (e *Encoder) Reader(r io.Reader) io.Reader {
	return transform.NewReader(r, e)
}

// Writer returns a writer that encodes to w.
func (e *Encoder) Writer(w io.Writer) io.Writer {
	return transform.NewWriter(w, e)
}

// HTMLEscapeUnsupported is a compatibility stub for x/text users that expect
// unsupported runes to be HTML-escaped during encoding. The current KolibriOS
// port keeps the encoder unchanged.
func HTMLEscapeUnsupported(e *Encoder) *Encoder {
	return e
}
