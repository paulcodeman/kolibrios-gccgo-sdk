// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import "io"

// NopCloser wraps an io.Reader with a no-op Close method.
func NopCloser(r io.Reader) io.ReadCloser { return io.NopCloser(r) }
