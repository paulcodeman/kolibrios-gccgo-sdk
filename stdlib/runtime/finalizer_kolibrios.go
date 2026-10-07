// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

// SetFinalizer associates obj with a finalizer that runs in a separate
// goroutine after the collector finds obj unreachable. The native backend
// follows libgo's argument validation and dependency ordering. The reflect
// package must be linked to provide the existing function-call FFI backend.
func SetFinalizer(obj any, finalizer any) __asm__("runtime.SetFinalizer")

// GC runs a garbage collection and blocks until collection finishes.
// Finalizers execute asynchronously when the Go scheduler next runs.
func GC() __asm__("runtime.GC")
