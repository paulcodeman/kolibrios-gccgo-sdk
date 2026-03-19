package main

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
