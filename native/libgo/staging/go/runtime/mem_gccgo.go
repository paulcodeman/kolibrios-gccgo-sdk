// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package runtime

import "unsafe"

// Functions called by C code.
//go:linkname sysAlloc
//go:linkname sysFree

//extern runtime_kolibrios_libgo_reserve
func kolibriosLibgoReserve(n uintptr) unsafe.Pointer

//extern runtime_kolibrios_libgo_release
func kolibriosLibgoRelease(v unsafe.Pointer, n uintptr)

// Don't split the stack as this method may be invoked without a valid G, which
// prevents us from allocating more stack.
//
//go:nosplit
func sysAlloc(n uintptr, sysStat *sysMemStat) unsafe.Pointer {
	p := kolibriosLibgoReserve(n)
	if p == nil {
		return nil
	}
	if sysStat != nil {
		sysStat.add(int64(n))
	}
	return p
}

func sysUnused(v unsafe.Pointer, n uintptr) {
	_, _ = v, n
}

func sysUsed(v unsafe.Pointer, n uintptr) {
	_, _ = v, n
}

func sysHugePage(v unsafe.Pointer, n uintptr) {
	_, _ = v, n
}

// Don't split the stack as this function may be invoked without a valid G,
// which prevents us from allocating more stack.
//
//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *sysMemStat) {
	if sysStat != nil {
		sysStat.add(-int64(n))
	}
	kolibriosLibgoRelease(v, n)
}

func sysFault(v unsafe.Pointer, n uintptr) {
	_, _ = v, n
}

func sysReserve(v unsafe.Pointer, n uintptr) unsafe.Pointer {
	_ = v
	return kolibriosLibgoReserve(n)
}

func sysMap(v unsafe.Pointer, n uintptr, sysStat *sysMemStat) {
	_, _ = v, n
	if sysStat != nil {
		sysStat.add(int64(n))
	}
}
