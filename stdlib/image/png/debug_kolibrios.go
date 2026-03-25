// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package png

import (
	"fmt"
	"kos"
	"strings"
)

var pngDebugSanitizer = strings.NewReplacer("\r", " ", "\n", " ")

func pngDebugf(format string, args ...interface{}) {
	line := strings.TrimSpace(fmt.Sprintf(format, args...))
	if line == "" {
		return
	}
	line = pngDebugSanitizer.Replace(line)
	kos.DebugString("png: ")
	kos.DebugString(line)
	kos.DebugString("\r\n")
}
