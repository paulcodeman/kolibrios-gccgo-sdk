// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

// Go 1.23 runtime/panic.go range-function checks. Numeric states match
// internal/abi/rangefuncconsts.go and the compiler compatibility pass.
const (
    rfDONE = iota
    rfREADY
    rfPANIC
    rfEXHAUSTED
    rfMISSING_PANIC
)

var rangeDoneError = error(errorString("range function continued iteration after function for loop body returned false"))
var rangePanicError = error(errorString("range function continued iteration after loop body panic"))
var rangeExhaustedError = error(errorString("range function continued iteration after whole loop exit"))
var rangeMissingPanicError = error(errorString("range function recovered a loop body panic and did not resume panicking"))

//go:noinline
func panicrangestate(state int) {
	switch state {
	case rfDONE:
		panic(rangeDoneError)
	case rfPANIC:
		panic(rangePanicError)
	case rfEXHAUSTED:
		panic(rangeExhaustedError)
	case rfMISSING_PANIC:
		panic(rangeMissingPanicError)
	}
	panic(errorString("unexpected state passed to panicrangestate"))
}

// GccgoCompatPanicRangeState is a compiler-only entrypoint for lowered
// Go 1.23 function ranges; user source is checked before this symbol is used.
func GccgoCompatPanicRangeState(state int) { panicrangestate(state) }
