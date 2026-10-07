// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "path"

// SameFile reports whether fi1 and fi2 describe the same file.
// KolibriOS does not expose inode identifiers; as allowed by the portable
// Go API, its decision is based on the path names. It cannot track identity
// across renames or distinguish replacements at an identical path.
func SameFile(fi1, fi2 FileInfo) bool {
	fs1, ok1 := fi1.(fileInfo)
	fs2, ok2 := fi2.(fileInfo)
	if !ok1 || !ok2 {
		return false
	}
	return fs1.path != "" && fs1.path == fs2.path
}

// Capture the absolute name when creating FileInfo, before a later Chdir.
func fileInfoPath(name string) string {
	if path.IsAbs(name) {
		return path.Clean(name)
	}
	wd, err := Getwd()
	if err != nil {
		return ""
	}
	return path.Join(wd, name)
}
