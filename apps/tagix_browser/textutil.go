package main

import "unicode/utf8"

func indexByte(value string, target byte) int {
	for i := 0; i < len(value); i++ {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func lastIndexByte(value string, target byte) int {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func toLowerASCII(value string) string {
	if value == "" {
		return ""
	}
	buf := make([]byte, len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		buf[i] = c
	}
	return string(buf)
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func clampShellInputIndex(value string, index int) int {
	if index <= 0 {
		return 0
	}
	if index >= len(value) {
		return len(value)
	}
	for index > 0 && index < len(value) && !utf8.RuneStart(value[index]) {
		index--
	}
	return index
}

func prevShellInputIndex(value string, index int) int {
	index = clampShellInputIndex(value, index)
	if index <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(value[:index])
	if size <= 0 {
		return index - 1
	}
	return index - size
}

func nextShellInputIndex(value string, index int) int {
	index = clampShellInputIndex(value, index)
	if index >= len(value) {
		return len(value)
	}
	_, size := utf8.DecodeRuneInString(value[index:])
	if size <= 0 {
		return index + 1
	}
	return index + size
}

func shellInputKeyString(code byte) string {
	if code >= 32 && code < 127 {
		return string([]byte{code})
	}
	return ""
}
