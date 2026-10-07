// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

// Adapted from Go 1.23 runtime/map.go mapclone. The bootstrap uses gccgo
// interface/type descriptors and a different map table layout; cloning that
// table belongs to the native map backend. Preserve the dynamic map type.
type mapCloneInterface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

//go:linkname mapclone maps.clone
func mapclone(m any) any {
	e := (*mapCloneInterface)(unsafe.Pointer(&m))
	e.data = cloneMapTable(e.typ, e.data)
	return m
}

func cloneMapTable(typ unsafe.Pointer, mapData unsafe.Pointer) unsafe.Pointer __asm__("runtime.cloneMapTable")
