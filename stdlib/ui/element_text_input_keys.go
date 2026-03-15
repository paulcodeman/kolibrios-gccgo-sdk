package ui

import (
	"kos"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const (
	scanCodeA        = 0x1E
	scanCodeC        = 0x2E
	scanCodeV        = 0x2F
	scanCodeX        = 0x2D
	scanCodeWinLeft  = 0x5B
	scanCodeWinRight = 0x5C
)

var (
	cp866Decoder = charmap.CodePage866.NewDecoder()
)

func keyMatchesLetter(key kos.KeyEvent, letter byte, scancode byte) bool {
	code := key.Code
	if code >= 'A' && code <= 'Z' {
		code = code + 32
	}
	if code == letter {
		return true
	}
	return key.ScanCode == scancode
}

func keyCodeToString(code byte) string {
	if code < 0x80 {
		return string([]byte{code})
	}
	if cp866Decoder == nil {
		return ""
	}
	decoded, err := cp866Decoder.Bytes([]byte{code})
	if err != nil || len(decoded) == 0 {
		return ""
	}
	r, size := utf8.DecodeRune(decoded)
	if r == utf8.RuneError && size == 1 {
		return ""
	}
	if r == unicode.ReplacementChar {
		return ""
	}
	if !unicode.IsPrint(r) && r != ' ' {
		return ""
	}
	return string(decoded)
}
