package main

import (
	"fmt"
	"kos"
	"strings"
)

var tagixDebugSanitizer = strings.NewReplacer("\r", " ", "\n", " ")

func tagixDebugf(format string, args ...interface{}) {
	line := strings.TrimSpace(fmt.Sprintf(format, args...))
	if line == "" {
		return
	}
	line = tagixDebugSanitizer.Replace(line)
	kos.DebugString("tagix_browser: ")
	kos.DebugString(line)
	kos.DebugString("\r\n")
}

func (app *App) debugf(format string, args ...interface{}) {
	tagixDebugf(format, args...)
}

func (app *App) debugError(context string, err error) {
	context = strings.TrimSpace(context)
	if err == nil {
		if context != "" {
			app.debugf("%s", context)
		}
		return
	}
	if context == "" {
		app.debugf("error: %v", err)
		return
	}
	app.debugf("%s: %v", context, err)
}
