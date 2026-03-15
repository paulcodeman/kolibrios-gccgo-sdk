// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package charmap provides decoders and encoders for single-byte character sets.
package charmap // import "golang.org/x/text/encoding/charmap"

import (
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Charmap is a fixed-width single-byte encoding.
type Charmap struct {
	name   string
	decode [256]rune
	encode map[rune]byte
}

// String returns the canonical name of the charmap.
func (c *Charmap) String() string {
	if c == nil {
		return ""
	}
	return c.name
}

// NewDecoder returns a new decoder for this charmap.
func (c *Charmap) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: &charmapDecoder{charmap: c}}
}

// NewEncoder returns a new encoder for this charmap.
func (c *Charmap) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: &charmapEncoder{charmap: c}}
}

// EncodeRune returns the encoded byte for r and reports whether it is representable.
func (c *Charmap) EncodeRune(r rune) (byte, bool) {
	if c == nil {
		return 0, false
	}
	b, ok := c.encode[r]
	return b, ok
}

// CodePage866 is IBM Code Page 866 (CP866).
var CodePage866 = newCharmap("IBM Code Page 866", decodeCP866)

// IBM866 is an alias for CodePage866.
var IBM866 = CodePage866

// Windows1251 is the Windows Cyrillic code page (CP1251).
var Windows1251 = newCharmap("Windows-1251", decodeCP1251)

// CodePage1251 is an alias for Windows1251.
var CodePage1251 = Windows1251

// Macintosh is the Macintosh Roman encoding.
var Macintosh = newCharmap("Macintosh", decodeMacintosh)

type charmapDecoder struct {
	transform.NopResetter
	charmap *Charmap
}

func (d *charmapDecoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if d == nil || d.charmap == nil {
		return 0, 0, nil
	}
	for nSrc < len(src) {
		r := d.charmap.decode[src[nSrc]]
		need := utf8.RuneLen(r)
		if need < 0 {
			need = 3
		}
		if nDst+need > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc++
	}
	return nDst, nSrc, nil
}

type charmapEncoder struct {
	transform.NopResetter
	charmap *Charmap
}

func (e *charmapEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if e == nil || e.charmap == nil {
		return 0, 0, nil
	}
	for nSrc < len(src) {
		if nDst >= len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && size == 1 {
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				return nDst, nSrc, transform.ErrShortSrc
			}
			return nDst, nSrc, encoding.ErrInvalidUTF8
		}
		b, ok := e.charmap.encode[r]
		if !ok {
			return nDst, nSrc, encoding.RepertoireError{Rune: r}
		}
		dst[nDst] = b
		nDst++
		nSrc += size
	}
	return nDst, nSrc, nil
}

type decodeFunc func(byte) rune

func newCharmap(name string, decode decodeFunc) *Charmap {
	charmap := &Charmap{
		name:   name,
		encode: make(map[rune]byte, 256),
	}
	for i := 0; i < 256; i++ {
		r := decode(byte(i))
		charmap.decode[i] = r
		if _, exists := charmap.encode[r]; !exists {
			charmap.encode[r] = byte(i)
		}
	}
	return charmap
}

func decodeCP866(value byte) rune {
	if value < 0x80 {
		return rune(value)
	}
	if value < 0xB0 {
		return rune(value) + 0x0390
	}
	return rune(cp866ToUnicode[value-0xB0])
}

var cp866ToUnicode = [...]uint16{
	0x2591, 0x2592, 0x2593, 0x2502, 0x2524, 0x2561, 0x2562, 0x2556,
	0x2555, 0x2563, 0x2551, 0x2557, 0x255D, 0x255C, 0x255B, 0x2510,
	0x2514, 0x2534, 0x252C, 0x251C, 0x2500, 0x253C, 0x255E, 0x255F,
	0x255A, 0x2554, 0x2569, 0x2566, 0x2560, 0x2550, 0x256C, 0x2567,
	0x2568, 0x2564, 0x2565, 0x2559, 0x2558, 0x2552, 0x2553, 0x256B,
	0x256A, 0x2518, 0x250C, 0x2588, 0x2584, 0x258C, 0x2590, 0x2580,
	0x0440, 0x0441, 0x0442, 0x0443, 0x0444, 0x0445, 0x0446, 0x0447,
	0x0448, 0x0449, 0x044A, 0x044B, 0x044C, 0x044D, 0x044E, 0x044F,
	0x0401, 0x0451, 0x0404, 0x0454, 0x0407, 0x0457, 0x040E, 0x045E,
	0x00B0, 0x2219, 0x00B7, 0x221A, 0x2116, 0x00A4, 0x25A0, 0x00A0,
}

// Mapping data derived from the Unicode Consortium CP1251 mapping table.
// https://www.unicode.org/Public/MAPPINGS/VENDORS/MICSFT/WINDOWS/CP1251.TXT
func decodeCP1251(value byte) rune {
	if value < 0x80 {
		return rune(value)
	}
	if value >= 0xC0 {
		return rune(value) + 0x0350
	}
	return rune(cp1251ToUnicode[value-0x80])
}

var cp1251ToUnicode = [...]uint16{
	0x0402, 0x0403, 0x201A, 0x0453, 0x201E, 0x2026, 0x2020, 0x2021,
	0x20AC, 0x2030, 0x0409, 0x2039, 0x040A, 0x040C, 0x040B, 0x040F,
	0x0452, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0xFFFD, 0x2122, 0x0459, 0x203A, 0x045A, 0x045C, 0x045B, 0x045F,
	0x00A0, 0x040E, 0x045E, 0x0408, 0x00A4, 0x0490, 0x00A6, 0x00A7,
	0x0401, 0x00A9, 0x0404, 0x00AB, 0x00AC, 0x00AD, 0x00AE, 0x0407,
	0x00B0, 0x00B1, 0x0406, 0x0456, 0x0491, 0x00B5, 0x00B6, 0x00B7,
	0x0451, 0x2116, 0x0454, 0x00BB, 0x0458, 0x0405, 0x0455, 0x0457,
}

func decodeMacintosh(value byte) rune {
	if value < 0x80 {
		return rune(value)
	}
	return macintoshToUnicode[value-0x80]
}

var macintoshToUnicode = [...]rune{
	0x00c4, 0x00c5, 0x00c7, 0x00c9, 0x00d1, 0x00d6, 0x00dc, 0x00e1,
	0x00e0, 0x00e2, 0x00e4, 0x00e3, 0x00e5, 0x00e7, 0x00e9, 0x00e8,
	0x00ea, 0x00eb, 0x00ed, 0x00ec, 0x00ee, 0x00ef, 0x00f1, 0x00f3,
	0x00f2, 0x00f4, 0x00f6, 0x00f5, 0x00fa, 0x00f9, 0x00fb, 0x00fc,
	0x2020, 0x00b0, 0x00a2, 0x00a3, 0x00a7, 0x2022, 0x00b6, 0x00df,
	0x00ae, 0x00a9, 0x2122, 0x00b4, 0x00a8, 0x2260, 0x00c6, 0x00d8,
	0x221e, 0x00b1, 0x2264, 0x2265, 0x00a5, 0x00b5, 0x2202, 0x2211,
	0x220f, 0x03c0, 0x222b, 0x00aa, 0x00ba, 0x03a9, 0x00e6, 0x00f8,
	0x00bf, 0x00a1, 0x00ac, 0x221a, 0x0192, 0x2248, 0x2206, 0x00ab,
	0x00bb, 0x2026, 0x00a0, 0x00c0, 0x00c3, 0x00d5, 0x0152, 0x0153,
	0x2013, 0x2014, 0x201c, 0x201d, 0x2018, 0x2019, 0x00f7, 0x25ca,
	0x00ff, 0x0178, 0x2044, 0x20ac, 0x2039, 0x203a, 0xfb01, 0xfb02,
	0x2021, 0x00b7, 0x201a, 0x201e, 0x2030, 0x00c2, 0x00ca, 0x00c1,
	0x00cb, 0x00c8, 0x00cd, 0x00ce, 0x00cf, 0x00cc, 0x00d3, 0x00d4,
	0xf8ff, 0x00d2, 0x00da, 0x00db, 0x00d9, 0x0131, 0x02c6, 0x02dc,
	0x00af, 0x02d8, 0x02d9, 0x02da, 0x00b8, 0x02dd, 0x02db, 0x02c7,
}
