// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios && gccgo

package runtime

import _ "unsafe" // for go:linkname

// libgo runtime/trace.go annotation entrypoints have no effect when tracing
// is disabled. StartTrace explicitly rejects tracing on this native backend,
// so only that upstream inactive path is reachable here.
//
//go:linkname traceUserTaskCreate runtime_1trace.userTaskCreate
func traceUserTaskCreate(id, parentID uint64, taskType string) {}

//go:linkname traceUserTaskEnd runtime_1trace.userTaskEnd
func traceUserTaskEnd(id uint64) {}

//go:linkname traceUserRegion runtime_1trace.userRegion
func traceUserRegion(id, mode uint64, regionType string) {}

//go:linkname traceUserLog runtime_1trace.userLog
func traceUserLog(id uint64, category, message string) {}

// All-goroutine fatal tracebacks need scheduler stack capture; do not silently
// accept a request that this runtime cannot honor. The ordinary CLI does not
// request it; testing's timeout path may, so its symbol must still link.
//
//go:linkname debugSetTraceback runtime_1debug.SetTraceback
func debugSetTraceback(level string) {
	panic(unavailableProfiler("runtime/debug.SetTraceback is not implemented on KolibriOS"))
}
