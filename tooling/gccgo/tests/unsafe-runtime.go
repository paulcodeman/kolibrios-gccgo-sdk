// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "unsafe"

//go:linkname unsafestring runtime.unsafestring
func unsafestring(ptr unsafe.Pointer, len int) {
	if len < 0 {
		panicunsafestringlen()
	}

	if uintptr(len) > -uintptr(ptr) {
		if ptr == nil {
			panicunsafestringnilptr()
		}
		panicunsafestringlen()
	}
}

// Keep this code in sync with cmd/compile/internal/walk/builtin.go:walkUnsafeString
//go:linkname unsafestring64 runtime.unsafestring64
func unsafestring64(ptr unsafe.Pointer, len64 int64) {
	len := int(len64)
	if int64(len) != len64 {
		panicunsafestringlen()
	}
	unsafestring(ptr, len)
}

func panicunsafestringlen() {
	panic(errorString("unsafe.String: len out of range"))
}

func panicunsafestringnilptr() {
	panic(errorString("unsafe.String: ptr is nil and len is not zero"))
}

type errorString string

func (e errorString) RuntimeError() {}

func (e errorString) Error() string {
	return "runtime error: " + string(e)
}

