package time

// Compare returns -1, 0, or +1 according to the ordering of t and u.
// The native time representation already compares monotonic readings when
// both values carry them, as required by the upstream Time.Compare API.
func (t Time) Compare(u Time) int { return t.compare(u) }
