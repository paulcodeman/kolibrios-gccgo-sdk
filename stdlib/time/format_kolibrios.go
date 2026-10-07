// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package time

// The original parser uses these internal accessors. The SDK currently
// supplies UTC and fixed zones; timezone database/DST loading is separate.
func (value Time) unixSec() int64 { return value.unixSeconds }

func (value *Time) addSec(seconds int64) { value.unixSeconds += seconds }

func (value *Time) setLoc(loc *Location) { value.loc = locationOrUTC(loc) }

func (loc *Location) lookup(seconds int64) (name string, offset int, start, end int64, isDST bool) {
	loc = locationOrUTC(loc)
	return loc.name, loc.offset, minInt64, maxInt64, false
}

func (loc *Location) lookupName(name string, seconds int64) (offset int, ok bool) {
	loc = locationOrUTC(loc)
	if name == loc.name {
		return loc.offset, true
	}
	return 0, false
}

// YearDay returns the day of the year, from 1 through 365 or 366.
func (value Time) YearDay() int {
	year, month, day := value.Date()
	result := int(daysBefore[month-1]) + day
	if month > February && isLeap(year) {
		result++
	}
	return result
}
