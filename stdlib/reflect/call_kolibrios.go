// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// libgo reflect/value.go call implementation, adapted to SDK helper names.
package reflect

import "unsafe"

func (v Value) Call(in []Value) []Value {
	v.flag.mustBe(Func, "reflect.Value.Call")
	v.mustBeExported("reflect.Value.Call")
	return v.call("Call", in)
}

func (v Value) CallSlice(in []Value) []Value {
	v.flag.mustBe(Func, "reflect.Value.Call")
	v.mustBeExported("reflect.Value.Call")
	return v.call("CallSlice", in)
}

func (v Value) call(op string, in []Value) []Value {
	// Get function pointer, type.
	t := (*funcType)(unsafe.Pointer(v.typ))
	var (
		fn   unsafe.Pointer
		rcvr Value
	)
	if v.flag&flagMethod != 0 {
		rcvr = v
		_, t, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	} else if v.flag&flagIndir != 0 {
		fn = *(*unsafe.Pointer)(v.ptr)
	} else {
		fn = v.ptr
	}

	if fn == nil {
		panic("reflect.Value.Call: call of nil function")
	}

	isSlice := op == "CallSlice"
	n := t.NumIn()
	isVariadic := t.IsVariadic()
	if isSlice {
		if !isVariadic {
			panic("reflect: CallSlice of non-variadic function")
		}
		if len(in) < n {
			panic("reflect: CallSlice with too few input arguments")
		}
		if len(in) > n {
			panic("reflect: CallSlice with too many input arguments")
		}
	} else {
		if isVariadic {
			n--
		}
		if len(in) < n {
			panic("reflect: Call with too few input arguments")
		}
		if !isVariadic && len(in) > n {
			panic("reflect: Call with too many input arguments")
		}
	}
	for _, x := range in {
		if x.Kind() == Invalid {
			panic("reflect: " + op + " using zero Value argument")
		}
	}
	for i := 0; i < n; i++ {
		if xt, targ := in[i].Type(), t.In(i); !xt.AssignableTo(targ) {
			panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String())
		}
	}
	if !isSlice && isVariadic {
		// prepare slice for remaining values
		m := len(in) - n
		slice := MakeSlice(t.In(n), m, m)
		elem := t.In(n).Elem()
		for i := 0; i < m; i++ {
			x := in[n+i]
			if xt := x.Type(); !xt.AssignableTo(elem) {
				panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op)
			}
			slice.Index(i).Set(x)
		}
		origIn := in
		in = make([]Value, n+1)
		copy(in[:n], origIn)
		in[n] = slice
	}

	nin := len(in)
	if nin != t.NumIn() {
		panic("reflect.Value.Call: wrong argument count")
	}
	nout := t.NumOut()

	if v.flag&flagMethod != 0 {
		nin++
	}
	firstPointer := len(in) > 0 && ifaceIndir(t.In(0).common()) && v.flag&flagMethodFn != 0
	params := make([]unsafe.Pointer, nin)
	off := 0
	if v.flag&flagMethod != 0 {
		// Hard-wired first argument.
		p := new(unsafe.Pointer)
		if rcvr.typ.Kind() == Interface {
			*p = unsafe.Pointer((*nonEmptyInterface)(v.ptr).word)
		} else if rcvr.typ.Kind() == Ptr || rcvr.typ.Kind() == UnsafePointer {
			*p = rawValuePointer(rcvr)
		} else {
			*p = rcvr.ptr
		}
		params[0] = unsafe.Pointer(p)
		off = 1
	}
	for i, pv := range in {
		pv.mustBeExported("reflect.Value.Call")
		targ := t.In(i).(*rtype)
		pv = assignToType("reflect.Value.Call", pv, targ)
		if pv.flag&flagIndir == 0 {
			p := new(unsafe.Pointer)
			*p = pv.ptr
			params[off] = unsafe.Pointer(p)
		} else {
			params[off] = pv.ptr
		}
		if i == 0 && firstPointer {
			p := new(unsafe.Pointer)
			*p = params[off]
			params[off] = unsafe.Pointer(p)
		}
		off++
	}

	ret := make([]Value, nout)
	results := make([]unsafe.Pointer, nout)
	for i := 0; i < nout; i++ {
		tv := t.Out(i)
		v := New(tv)
		results[i] = rawValuePointer(v)
		fl := flagIndir | flag(tv.Kind())
		ret[i] = Value{tv.common(), rawValuePointer(v), fl}
	}

	var pp *unsafe.Pointer
	if len(params) > 0 {
		pp = &params[0]
	}
	var pr *unsafe.Pointer
	if len(results) > 0 {
		pr = &results[0]
	}

	ffiCall(t, fn, v.flag&flagMethod != 0, firstPointer, pp, pr)

	return ret
}

func methodReceiver(op string, v Value, methodIndex int) (rcvrtype *rtype, t *funcType, fn unsafe.Pointer) {
	i := methodIndex
	if v.typ.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ))
		if uint(i) >= uint(len(tt.methods)) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.methods[i]
		if m.pkgPath != nil {
			panic("reflect: " + op + " of unexported method")
		}
		iface := (*nonEmptyInterface)(v.ptr)
		if iface.itab == nil {
			panic("reflect: " + op + " of method on nil interface value")
		}
		rcvrtype = iface.itab.typ
		fn = unsafe.Pointer(&iface.itab.fun[i])
		t = (*funcType)(unsafe.Pointer(m.typ))
	} else {
		rcvrtype = v.typ
		ms := v.typ.exportedMethods()
		if uint(i) >= uint(len(ms)) {
			panic("reflect: internal error: invalid method index")
		}
		m := ms[i]
		if m.pkgPath != nil {
			panic("reflect: " + op + " of unexported method")
		}
		fn = unsafe.Pointer(&m.tfn)
		t = (*funcType)(unsafe.Pointer(m.mtyp))
	}
	return
}

func align(x, n uintptr) uintptr {
	return (x + n - 1) &^ (n - 1)
}

type nonEmptyInterface struct {
	itab *interfaceMethodTable
	word unsafe.Pointer
}
type interfaceMethodTable struct {
	typ *rtype
	fun [100000]uintptr // libgo's variable-length method table view
}

func (v Value) mustBeExported(method string) {
	if v.flag == 0 {
		panic(&ValueError{method, Invalid})
	}
	if v.flag&flagRO != 0 {
		panic("reflect: " + method + " using value obtained using unexported field")
	}
}
func ffiCall(*funcType, unsafe.Pointer, bool, bool, *unsafe.Pointer, *unsafe.Pointer) __asm__("reflect_call")
