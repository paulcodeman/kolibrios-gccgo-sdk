//go:build kolibrios && 386

package cpu

const CacheLinePadSize = 64

// KolibriOS starts through the SDK rather than libgo runtime startup.
func init() { Initialize("") }
