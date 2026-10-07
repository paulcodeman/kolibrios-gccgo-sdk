// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

// Adapted from Go 1.23 crypto/rand/rand_wasip1.go: the platform call is
// backed by upstream OpenSSL CPU random routines in the native runtime.
package rand

import "errors"

func init() { Reader = &reader{} }

type reader struct{}

var errRandomUnavailable = errors.New("crypto/rand: secure CPU random source unavailable")

func nativeReadRandom(*byte, int) int __asm__("runtime.kolibriReadRandom")

func (r *reader) Read(b []byte) (int, error) {
 if len(b)==0 { return 0,nil }
 n:=nativeReadRandom(&b[0],len(b))
 if n!=len(b) { return n,errRandomUnavailable }
 return len(b),nil
}
