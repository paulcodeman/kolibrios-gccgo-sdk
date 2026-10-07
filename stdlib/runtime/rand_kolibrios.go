// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios && gccgo

package runtime

import (
	"internal/chacha8rand"
	_ "unsafe" // for go:linkname
)

// Go 1.23 bootstrapRand and readTimeRandom adapted to the native scheduler.
// There is one protected stream instead of per-m state. The OS currently has
// no entropy source wired to this backend, so initialization uses upstream's
// time fallback. This powers math/rand/v2, not crypto/rand.
type randomMutex struct{ state uint32 }

var globalRand struct {
	lock  randomMutex
	seed  [32]byte
	state chacha8rand.State
	init  bool
}

func randomLock(lock *randomMutex) __asm__("runtime.lock")
func randomUnlock(lock *randomMutex) __asm__("runtime.unlock")
func randomNanotime() int64 __asm__("runtime.nanotime")

// readTimeRandom stretches any entropy in the current time
// into entropy the length of r and XORs it into r.
// This is a fallback for when readRandom does not read
// the full requested amount.
// Whatever entropy r already contained is preserved.
func readTimeRandom(r []byte) {
	// Inspired by wyrand.
	// An earlier version of this code used getg().m.procid as well,
	// but note that this is called so early in startup that procid
	// is not initialized yet.
	v := uint64(randomNanotime())
	for len(r) > 0 {
		v ^= 0xa0761d6478bd642f
		v *= 0xe7037ed1a0b428db
		size := 8
		if len(r) < 8 {
			size = len(r)
		}
		for i := 0; i < size; i++ {
			r[i] ^= byte(v >> (8 * i))
		}
		r = r[size:]
		v = v>>32 | v<<32
	}
}

//go:linkname kolibriRand runtime.rand
func kolibriRand() uint64 {
	randomLock(&globalRand.lock)
	if !globalRand.init {
		readTimeRandom(globalRand.seed[:])
		globalRand.state.Init(globalRand.seed)
		clear(globalRand.seed[:])
		globalRand.init = true
	}
	for {
		if x, ok := globalRand.state.Next(); ok {
			randomUnlock(&globalRand.lock)
			return x
		}
		globalRand.state.Refill()
	}
}
