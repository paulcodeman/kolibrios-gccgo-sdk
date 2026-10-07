// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect

import (
	"errors"
	"unsafe"
)

func (m Method) IsExported() bool { return m.PkgPath == "" }

func (t *rtype) ChanDir() ChanDir {
	if t.Kind() != Chan {
		panic("reflect: ChanDir of non-chan type " + t.String())
	}
	return ChanDir((*chanType)(unsafe.Pointer(t)).dir)
}

// FieldByIndexErr follows the upstream field traversal, retaining its nil
// embedded-pointer error instead of changing it to a panic.
func (v Value) FieldByIndexErr(index []int) (Value, error) {
	if len(index) == 1 {
		return v.Field(index[0]), nil
	}
	v.flag.mustBe(Struct, "reflect.Value.FieldByIndexErr")
	for i, x := range index {
		if i > 0 && v.Kind() == Pointer && v.typ.Elem().Kind() == Struct {
			if v.IsNil() {
				return Value{}, errors.New("reflect: indirection through nil pointer to embedded struct field " + v.typ.Elem().Name())
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v, nil
}

func runtimeChanrecv2(ch unsafe.Pointer, elem unsafe.Pointer) bool __asm__("runtime.chanrecv2")

// Recv uses upstream's Value construction and the existing gccgo channel ABI.
func (v Value) Recv() (val Value, ok bool) {
	v.flag.mustBe(Chan, "reflect.Value.Recv")
	if v.flag&flagRO != 0 {
		panic("reflect: reflect.Value.Recv using value obtained using unexported field")
	}
	tt := (*chanType)(unsafe.Pointer(v.typ))
	if ChanDir(tt.dir)&RecvDir == 0 {
		panic("reflect: recv on send-only channel")
	}
	t := tt.elem
	val = Value{typ: t, flag: flag(t.Kind())}
	var p unsafe.Pointer
	if ifaceIndir(t) {
		p = runtimeNewObject(t)
		val.ptr = p
		val.flag |= flagIndir
	} else {
		p = unsafe.Pointer(&val.ptr)
	}
	ok = runtimeChanrecv2(rawValuePointer(v), p)
	return
}
