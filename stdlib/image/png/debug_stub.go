// +build !kolibrios !gccgo

//go:build !kolibrios || !gccgo

package png

func pngDebugf(format string, args ...interface{}) {}
