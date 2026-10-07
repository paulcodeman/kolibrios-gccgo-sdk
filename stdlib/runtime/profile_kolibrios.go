//go:build kolibrios && gccgo

package runtime

// The native runtime does not yet have event tracing or profile sampling.
// The normal application path keeps them disabled. Requests to enable an
// unavailable profiler fail explicitly rather than returning an empty profile.
type unavailableProfiler string
func (e unavailableProfiler) Error() string { return string(e) }

// MemProfileRate is zero because heap sampling is currently disabled.
// The allocation counters in ReadMemStats do not depend on this setting.
var MemProfileRate int

func StartTrace() error { return unavailableProfiler("runtime: tracing is not implemented on KolibriOS") }
func ReadTrace() []byte { return nil } // No trace can be started.
func StopTrace() {} // Upstream StopTrace is harmless when tracing is inactive.

func SetBlockProfileRate(rate int) {
	if rate > 0 { panic(unavailableProfiler("runtime: block profiling is not implemented on KolibriOS")) }
}

func SetMutexProfileFraction(rate int) int {
	if rate > 0 { panic(unavailableProfiler("runtime: mutex profiling is not implemented on KolibriOS")) }
	return 0 // Negative values query the disabled sampler; zero disables it.
}
