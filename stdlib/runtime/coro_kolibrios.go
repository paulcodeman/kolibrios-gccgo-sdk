// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from Go 1.23.0 runtime/coro.go. Coroutine user code and its
// deferred cleanup retain the upstream flow; the scheduler entrypoints
// use the SDK's gccgo goroutine and function-value ABI.

package runtime

import "unsafe"

// A coro represents concurrency without additional parallelism. A switch
// exchanges the calling goroutine with the goroutine waiting in gp.
// Keep this layout aligned with runtime_kolibri_coro in runtime_gccgo.c.
type coro struct {
	gp unsafe.Pointer
	f  func(*coro)
	mp unsafe.Pointer
}

//go:linkname newcoro
func newcoro(f func(*coro)) *coro {
	c := new(coro)
	c.f = f
	start := corostart
	// gccgo stores a function as a pointer to its code/context descriptor.
	// corostart is a top-level function and needs no captured context.
	startfv := *(*unsafe.Pointer)(unsafe.Pointer(&start))
	coronew(c, *(*uintptr)(startfv))
	return c
}

func corostart(c *coro) {
	defer coroexit(c)
	c.f(c)
}

func coronew(c *coro, entry uintptr) __asm__("runtime_kolibri_coronew")
func coroswitch(c *coro) __asm__("runtime_kolibri_coroswitch")
func coroexit(c *coro) __asm__("runtime_kolibri_coroexit")
