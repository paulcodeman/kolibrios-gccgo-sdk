// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import _ "unsafe"

// A TypeAssertionError explains a failed type assertion.
type TypeAssertionError struct {
	_interface    *_type
	concrete      *_type
	asserted      *_type
	missingMethod string // one method needed by Interface, missing from Concrete
}

func (*TypeAssertionError) RuntimeError() {}

func (e *TypeAssertionError) Error() string {
	inter := "interface"
	if e._interface != nil {
		inter = e._interface.string()
	}
	as := e.asserted.string()
	if e.concrete == nil {
		return "interface conversion: " + inter + " is nil, not " + as
	}
	cs := e.concrete.string()
	if e.missingMethod == "" {
		msg := "interface conversion: " + inter + " is " + cs + ", not " + as
		if cs == as {
			// provide slightly clearer error message
			if e.concrete.pkgpath() != e.asserted.pkgpath() {
				msg += " (types from different packages)"
			} else {
				msg += " (types from different scopes)"
			}
		}
		return msg
	}
	return "interface conversion: " + cs + " is not " + as +
		": missing method " + e.missingMethod
}

//go:linkname panicdottype runtime.panicdottype
func panicdottype(lhs, rhs, inter *_type) {
	panic(&TypeAssertionError{inter, rhs, lhs, ""})
}

// Adapt libgo getitab/assertitab's failure to the native itab cache.
//
//go:linkname panicInterfaceAssertion runtime.panicInterfaceAssertion
func panicInterfaceAssertion(lhs, rhs *_type, missing string) {
	panic(&TypeAssertionError{nil, rhs, lhs, missing})
}
