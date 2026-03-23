// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package runtime

var defaultGOROOT string
var buildVersion string

func gogetenv(key string) string {
	_ = key
	return ""
}
