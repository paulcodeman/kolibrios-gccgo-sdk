// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

// Source-compatible terminal event identifier from Go's linux/386 constants.
// Native console size changes supply this event; it adds no kernel syscall.
const SIGWINCH = Signal(0x1c)
