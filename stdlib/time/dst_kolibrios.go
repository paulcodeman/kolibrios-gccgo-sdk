// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package time

// IsDST reports whether the time in the configured location is in Daylight Savings Time.
// The SDK currently supports UTC and fixed-offset locations only; neither
// includes daylight-saving transitions. LoadLocation rejects other locations.
// This adapts upstream Time.IsDST's lookup to the supported Location model.
func (t Time) IsDST() bool { return false }
