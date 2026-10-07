// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import _ "unsafe"

//go:linkname panicmakeslicelen runtime.panicmakeslicelen
func panicmakeslicelen() {
	panic(errorString("makeslice: len out of range"))
}

//go:linkname panicmakeslicecap runtime.panicmakeslicecap
func panicmakeslicecap() {
	panic(errorString("makeslice: cap out of range"))
}

//go:linkname panicunsafeslicelen runtime.panicunsafeslicelen
func panicunsafeslicelen() {
	panic(errorString("unsafe.Slice: len out of range"))
}

//go:linkname panicunsafeslicenilptr runtime.panicunsafeslicenilptr
func panicunsafeslicenilptr() {
	panic(errorString("unsafe.Slice: ptr is nil and len is not zero"))
}
