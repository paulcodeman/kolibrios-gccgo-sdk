// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filepath

import "strings"

// SplitList splits a list of paths joined by the OS-specific ListSeparator.
// Unlike strings.Split, SplitList returns an empty slice for an empty string.
// Functions imported unchanged from Go 1.23 path.go and path_unix.go.
func SplitList(path string) []string {
	return splitList(path)
}

func splitList(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, string(ListSeparator))
}
