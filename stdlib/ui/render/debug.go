package ui

// Optional debug hooks used by stress tests and diagnostics.
var (
	DebugTrace               func(label string)
	DebugTraceElement        *Element
	DebugTraceRenderStride   int
	DebugTraceRangeStart     int
	DebugTraceRangeEnd       int
	DebugRenderOnlyIndex     int
	DebugSlowNodeThresholdNs uint64
)
