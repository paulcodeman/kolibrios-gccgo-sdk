// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// From Go 1.23 time/time.go; method calls use the equivalent SDK accessors.
package time

func UnixMilli(msec int64) Time {
	return Unix(msec/1e3, (msec%1e3)*1e6)
}

func UnixMicro(usec int64) Time {
	return Unix(usec/1e6, (usec%1e6)*1e3)
}

func (t Time) UnixMilli() int64 {
	return t.Unix()*1e3 + int64(t.Nanosecond())/1e6
}

func (t Time) UnixMicro() int64 {
	return t.Unix()*1e6 + int64(t.Nanosecond())/1e3
}
