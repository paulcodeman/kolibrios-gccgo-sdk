// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// GCC 13.3.0 libgo method values; adapted to the SDK libffi bridge.
package reflect

import "unsafe"

func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makeMethodValue")
	}

	// Ignoring the flagMethod bit, v describes the receiver, not the method type.
	fl := v.flag & (flagRO | flagAddr | flagIndir)
	fl |= flag(v.typ.Kind())
	rcvr := Value{v.typ, v.ptr, fl}

	// v.Type returns the actual type of the method value.
	ft := v.Type().(*rtype)

	// Cause panic if method is not appropriate.
	// The panic would still happen during the call if we omit this,
	// but we want Interface() and other operations to fail early.
	_, t, _ := methodReceiver(op, rcvr, int(v.flag)>>flagMethodShift)

	ftyp := (*funcType)(unsafe.Pointer(t))
	method := int(v.flag) >> flagMethodShift

	fv := &makeFuncImpl{
		typ:    ftyp,
		method: method,
		rcvr:   rcvr,
	}

	makeFuncFFI(unsafe.Pointer(makeCIF(ftyp)), unsafe.Pointer(fv))

	return Value{ft, unsafe.Pointer(&fv), v.flag&flagRO | flag(Func) | flagIndir}
}

func makeValueMethod(v Value) Value {
	typ := v.typ
	if typ.Kind() != Func {
		panic("reflect: call of makeValueMethod with non-Func type")
	}
	if v.flag&flagMethodFn == 0 {
		panic("reflect: call of makeValueMethod with non-MethodFn")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	impl := &makeFuncImpl{
		typ:    ftyp,
		method: -2,
		rcvr:   v,
	}

	makeFuncFFI(unsafe.Pointer(makeCIF(ftyp)), unsafe.Pointer(impl))

	return Value{t, unsafe.Pointer(&impl), v.flag&flagRO | flag(Func) | flagIndir}
}
