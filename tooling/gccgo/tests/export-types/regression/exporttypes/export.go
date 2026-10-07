package exporttypes

// Exporting builtin min/max inline bodies alongside an ordinary func()
// used to crash the comparator on their equal backend names.
func NoOp() {}

var Callback = NoOp
var Variadic = func(values ...int) int { return len(values) }
var Slice = func(values []int) int { return len(values) }

func Clamp(value, low, high int) int {
	return min(high, max(low, value))
}
