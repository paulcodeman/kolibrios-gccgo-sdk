// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflectlite provides a minimal reflection API used by the standard library.
package reflectlite

import "unsafe"

type Type interface {
	Comparable() bool
	String() string
}

type Value struct {
	typ *rtype
	ptr unsafe.Pointer
}

type eface struct {
	typ *rtype
	ptr unsafe.Pointer
}

type goString struct {
	str *byte
	len uintptr
}

type rtype struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      uint8
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      unsafe.Pointer
	gcdata     unsafe.Pointer
	name       *goString
	uncommon   unsafe.Pointer
	ptrToThis  *rtype
}

type sliceType struct {
	rtype
	elem *rtype
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

const (
	kindMask  = (1 << 5) - 1
	kindSlice = 23
)

func TypeOf(i any) Type {
	e := *(*eface)(unsafe.Pointer(&i))
	if e.typ == nil {
		return nil
	}
	return e.typ
}

func ValueOf(i any) Value {
	e := *(*eface)(unsafe.Pointer(&i))
	return Value{typ: e.typ, ptr: e.ptr}
}

func (v Value) Len() int {
	if v.typ == nil {
		panic("reflectlite: Len of invalid value")
	}
	if v.typ.kind&kindMask != kindSlice {
		panic("reflectlite: Len of non-slice value")
	}
	hdr := (*sliceHeader)(v.ptr)
	return hdr.Len
}

func Swapper(slice any) func(i, j int) {
	v := ValueOf(slice)
	if v.typ == nil || v.typ.kind&kindMask != kindSlice {
		panic("reflectlite: Swapper of non-slice value")
	}
	st := (*sliceType)(unsafe.Pointer(v.typ))
	es := st.elem.size
	if es == 0 {
		return func(i, j int) {}
	}
	hdr := (*sliceHeader)(v.ptr)
	data := hdr.Data
	tmp := make([]byte, int(es))

	return func(i, j int) {
		if i == j {
			return
		}
		offI := data + uintptr(i)*es
		offJ := data + uintptr(j)*es
		bi := byteSlice(unsafe.Pointer(offI), es)
		bj := byteSlice(unsafe.Pointer(offJ), es)
		copy(tmp, bi)
		copy(bi, bj)
		copy(bj, tmp)
	}
}

func (t *rtype) Comparable() bool {
	return t != nil && t.equal != nil
}

func (t *rtype) String() string {
	if t == nil {
		return ""
	}
	if t.name != nil && t.name.str != nil && t.name.len > 0 {
		return goStringToString(t.name)
	}
	return kindName(t.kind & kindMask)
}

func goStringToString(s *goString) string {
	if s == nil || s.str == nil || s.len == 0 {
		return ""
	}
	b := byteSlice(unsafe.Pointer(s.str), s.len)
	clone := make([]byte, len(b))
	copy(clone, b)
	return string(clone)
}

func byteSlice(ptr unsafe.Pointer, n uintptr) []byte {
	var b []byte
	sh := (*sliceHeader)(unsafe.Pointer(&b))
	sh.Data = uintptr(ptr)
	sh.Len = int(n)
	sh.Cap = int(n)
	return b
}

func kindName(kind uint8) string {
	switch kind {
	case 0:
		return "invalid"
	case 1:
		return "bool"
	case 2:
		return "int"
	case 3:
		return "int8"
	case 4:
		return "int16"
	case 5:
		return "int32"
	case 6:
		return "int64"
	case 7:
		return "uint"
	case 8:
		return "uint8"
	case 9:
		return "uint16"
	case 10:
		return "uint32"
	case 11:
		return "uint64"
	case 12:
		return "uintptr"
	case 13:
		return "float32"
	case 14:
		return "float64"
	case 15:
		return "complex64"
	case 16:
		return "complex128"
	case 17:
		return "array"
	case 18:
		return "chan"
	case 19:
		return "func"
	case 20:
		return "interface"
	case 21:
		return "map"
	case 22:
		return "ptr"
	case 23:
		return "slice"
	case 24:
		return "string"
	case 25:
		return "struct"
	case 26:
		return "unsafe.Pointer"
	default:
		return "unknown"
	}
}

