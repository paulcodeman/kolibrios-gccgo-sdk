// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io

type WriterAt interface {
	WriteAt(p []byte, off int64) (n int, err error)
}
