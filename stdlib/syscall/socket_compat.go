// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

// Go socket shutdown directions from syscall/zerrors_linux_386.go. These
// are compatibility values; the native socket ABI has no shutdown syscall.
const (
	SHUT_RD   = 0x0
	SHUT_RDWR = 0x2
	SHUT_WR   = 0x1
)
