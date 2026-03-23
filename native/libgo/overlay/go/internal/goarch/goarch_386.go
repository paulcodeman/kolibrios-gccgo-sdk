// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build 386

package goarch

const (
	_ArchFamily          = I386
	_BigEndian           = false
	_DefaultPhysPageSize = 4096
	_PCQuantum           = 1
	_Int64Align          = 4
	_MinFrameSize        = 0
	_StackAlign          = PtrSize
)
