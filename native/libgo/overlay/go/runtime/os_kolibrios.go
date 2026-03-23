// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package runtime

import (
	"runtime/internal/atomic"
	_ "unsafe"
)

// KolibriOS does not provide futexes or POSIX semaphores. The first native
// port slice uses polling backed by explicit yield/sleep until a stronger wait
// primitive is wired in.
type mOS struct{}

// For C code to call:
//go:linkname minit

func goenvs() {
	// TODO: switch to loader-backed argv/env bootstrap once the native
	// startup path is wired in for libgo.
	envs = make([]string, 0)
}

// Called to initialize a new m (including the bootstrap m).
// Called on the parent thread, can allocate memory.
func mpreinit(mp *m) {
	mp.gsignal = malg(true, true, &mp.gsignalstack, &mp.gsignalstacksize)
	mp.gsignal.m = mp
}

// Called on the new thread, cannot allocate memory.
func minit() {
	getg().m.procid = getProcID()
}

//go:nosplit
//go:nowritebarrierrec
func unminit() {
}

func mdestroy(mp *m) {
	_ = mp
}

func getRandomData(r []byte) {
	if startupRandomData != nil {
		n := copy(r, startupRandomData)
		extendRandom(r, n)
		return
	}

	seed := uint64(nanotime()) ^ getProcID() ^ 0x9e3779b97f4a7c15
	for i := range r {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		r[i] = byte(seed)
	}
}

func getProcID() uint64 {
	return uint64(kolibriosThreadSlot())
}

//go:nosplit
func futexsleep(addr *uint32, val uint32, ns int64) {
	if atomic.Load(addr) != val {
		return
	}

	if ns < 0 {
		for atomic.Load(addr) == val {
			usleep_no_g(10_000)
		}
		return
	}

	deadline := nanotime() + ns
	if deadline < 0 {
		deadline = 1<<63 - 1
	}

	for atomic.Load(addr) == val {
		now := nanotime()
		if now >= deadline {
			return
		}

		remaining := deadline - now
		switch {
		case remaining >= 10_000_000:
			usleep_no_g(10_000)
		case remaining >= 100_000:
			usleep_no_g(uint32((remaining + 999) / 1000))
		default:
			osyield_no_g()
		}
	}
}

//go:nosplit
func futexwakeup(addr *uint32, cnt uint32) {
	_, _ = addr, cnt
}

func osinit() {
	// TODO: query actual CPU count when the native target exposes it cleanly.
	ncpu = 1
	physPageSize = 4096
	physHugePageSize = 0
}

// For gccgo this hook is provided by the C runtime side.
func osyield()

//go:nosplit
func osyield_no_g() {
	osyield()
}

func kolibriosThreadSlot() uint32
