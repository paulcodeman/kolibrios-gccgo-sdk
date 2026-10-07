// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios && gccgo

package chacha8rand

import _ "unsafe" // for go:linkname

// Keep both original upstream files unchanged. The native entrypoint selects
// their portable block implementation in place of the Go assembler backend.
//
//go:linkname blockKolibri internal_1chacha8rand.block
func blockKolibri(seed *[4]uint64, blocks *[32]uint64, counter uint32) {
	block_generic(seed, blocks, counter)
}
