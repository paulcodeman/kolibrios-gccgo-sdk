// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package syscall

// Standard descriptor indexes, as in Go syscall_linux.go. The SDK console
// uses these same indexes; socket and file handles have separate namespaces.
var (
	Stdin = 0
	Stdout = 1
	Stderr = 2
)
