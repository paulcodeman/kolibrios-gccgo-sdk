//go:build kolibrios && gccgo

package runtime

// ReadMemStats reads an application-wide snapshot of the native collector.
// Alloc, TotalAlloc, Mallocs, Frees, HeapAlloc, HeapObjects, NextGC, NumGC,
// and EnableGC are available. Span, stack, profiling and pause accounting
// are not yet maintained by this collector and remain zero.
func ReadMemStats(m *MemStats) __asm__("runtime_kolibri_read_memstats")
