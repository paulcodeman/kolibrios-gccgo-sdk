//go:build !kolibrios
// +build !kolibrios

package fyne

import "runtime"

func callerLocation(skip int) (string, int, bool) {
	_, file, line, ok := runtime.Caller(skip)
	return file, line, ok
}
