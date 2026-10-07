// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// libgo MakeFunc and libffi callback, adapted to the SDK helper names.
package reflect

import (
	"runtime"
	"unsafe"
)

type makeFuncImpl struct {
	code    uintptr
	ffi_cif unsafe.Pointer
	ffi_fun func(unsafe.Pointer, unsafe.Pointer)

	typ *funcType
	fn  func([]Value) []Value

	// For gccgo we use the same entry point for functions and for
	// method values.
	method int
	rcvr   Value
}

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	impl := &makeFuncImpl{
		typ:    ftyp,
		fn:     fn,
		method: -1,
	}

	makeFuncFFI(unsafe.Pointer(makeCIF(ftyp)), unsafe.Pointer(impl))

	return Value{t, unsafe.Pointer(&impl), flag(Func) | flagIndir}
}

func (c *makeFuncImpl) call(in []Value) []Value {
	if c.method == -1 {
		return c.fn(in)
	} else if c.method == -2 {
		if c.typ.IsVariadic() {
			return c.rcvr.CallSlice(in)
		} else {
			return c.rcvr.Call(in)
		}
	} else {
		m := c.rcvr.Method(c.method)
		if c.typ.IsVariadic() {
			return m.CallSlice(in)
		} else {
			return m.Call(in)
		}
	}
}
func makeFuncFFI(unsafe.Pointer, unsafe.Pointer)

//go:linkname ffiCallbackGo

// ffiCallbackGo implements the Go side of the libffi callback.
//
// The call chain arriving here looks like
//
//	some_go_caller
//	->some_ffi_internals
//	  ->ffi_callback (in C)
//	    ->ffiCallbackGo
//
// The ffi_callback handles __go_makefunc_can_recover, and
// then passes off the data as received from ffi here.
func ffiCallbackGo(results unsafe.Pointer, params unsafe.Pointer, impl *makeFuncImpl, wordsize int32, bigEndian bool) {
	ftyp := impl.typ
	in := make([]Value, 0, len(ftyp.in))
	ap := params
	for _, rt := range ftyp.in {
		p := runtimeNewObject(rt)
		runtimeTypedmemmove(rt, p, *(*unsafe.Pointer)(ap))
		v := Value{rt, p, flag(rt.Kind()) | flagIndir}
		in = append(in, v)
		ap = (unsafe.Pointer)(uintptr(ap) + unsafe.Sizeof(uintptr(0)))
	}

	out := impl.call(in)
	if len(out) != len(ftyp.out) {
		panic("reflect: wrong return count from function created by MakeFunc")
	}

	checkValue := func(v Value, typ *rtype, addr unsafe.Pointer) Value {
		if v.flag&flagRO != 0 {
			panic("reflect: function created by MakeFunc using " + funcName(impl.fn) +
				" returned value obtained from unexported field")
		}

		// Convert v to type typ if v is assignable to a variable
		// of type t in the language spec.
		// See issue 28761.
		return assignToType("reflect.MakeFunc", v, typ)
	}

	// In libffi a single integer return value is always promoted
	// to a full word. This only matters for integers whose size
	// is less than the size of a full word. There is similar code
	// in libgo/runtime/go-reflect-call.c.
	if len(ftyp.out) == 1 {
		typ := ftyp.out[0]
		switch typ.Kind() {
		case Bool, Int8, Int16, Int32, Uint8, Uint16, Uint32:
			v := out[0]
			v = checkValue(v, typ, nil)

			if bigEndian {
				results = unsafe.Pointer(uintptr(results) + uintptr(wordsize) - typ.size)
			}

			runtimeTypedmemmove(typ, results, valueDataPointer(v))
			return
		}
	}

	off := uintptr(0)
	for i, typ := range ftyp.out {
		v := out[i]

		off = align(off, uintptr(typ.fieldAlign))
		addr := unsafe.Pointer(uintptr(results) + off)

		v = checkValue(v, typ, addr)

		runtimeTypedmemmove(typ, addr, valueDataPointer(v))
		off += typ.size
	}
}

func funcName(fn func([]Value) []Value) string {
	p := *(*unsafe.Pointer)(unsafe.Pointer(&fn))
	if p != nil {
		if f := runtime.FuncForPC(*(*uintptr)(p)); f != nil {
			return f.Name()
		}
	}
	return "closure"
}
