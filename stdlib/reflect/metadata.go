// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Extracted from GCC 13.3.0 libgo reflect/type.go.
// The SDK keeps the same gccgo type and method layouts.

package reflect

import (
	"sync"
	"unsafe"
)

func (t *rtype) exportedMethods() []method {
	ut := t.uncommon()
	if ut == nil {
		return nil
	}
	return ut.exportedMethods()
}

func (t *rtype) NumMethod() int {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.NumMethod()
	}
	return len(t.exportedMethods())
}

func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.Method(i)
	}
	methods := t.exportedMethods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	if p.name != nil {
		m.Name = *p.name
	}
	fl := flag(Func)
	mt := p.typ
	m.Type = toType(mt)
	x := new(unsafe.Pointer)
	*x = unsafe.Pointer(&p.tfn)
	m.Func = Value{mt, unsafe.Pointer(x), fl | flagIndir | flagMethodFn}
	m.Index = i
	return m
}

func (t *rtype) MethodByName(name string) (m Method, ok bool) {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.MethodByName(name)
	}
	ut := t.uncommon()
	if ut == nil {
		return Method{}, false
	}
	utmethods := ut.methods
	var eidx int
	for i := 0; i < len(utmethods); i++ {
		p := utmethods[i]
		if p.pkgPath == nil {
			if p.name != nil && *p.name == name {
				return t.Method(eidx), true
			}
			eidx++
		}
	}
	return Method{}, false
}

func (t *rtype) IsVariadic() bool {
	if t.Kind() != Func {
		panic("reflect: IsVariadic of non-func type " + t.String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return tt.dotdotdot
}

func (t *rtype) In(i int) Type {
	if t.Kind() != Func {
		panic("reflect: In of non-func type " + t.String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.in[i])
}

func (t *rtype) NumIn() int {
	if t.Kind() != Func {
		panic("reflect: NumIn of non-func type " + t.String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return len(tt.in)
}

func (t *rtype) NumOut() int {
	if t.Kind() != Func {
		panic("reflect: NumOut of non-func type " + t.String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return len(tt.out)
}

func (t *rtype) Out(i int) Type {
	if t.Kind() != Func {
		panic("reflect: Out of non-func type " + t.String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return toType(tt.out[i])
}

func (t *rtype) Implements(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.Implements")
	}
	if u.Kind() != Interface {
		panic("reflect: non-interface type passed to Type.Implements")
	}
	return implements(u.(*rtype), t)
}

func (t *rtype) AssignableTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo")
	}
	uu := u.(*rtype)
	return directlyAssignable(uu, t) || implements(uu, t)
}

func (t *rtype) hasName() bool {
	return t.uncommonType != nil && t.uncommonType.name != nil
}

func (t *interfaceType) Method(i int) (m Method) {
	if i < 0 || i >= len(t.methods) {
		return
	}
	p := &t.methods[i]
	m.Name = *p.name
	if p.pkgPath != nil {
		m.PkgPath = *p.pkgPath
	}
	m.Type = toType(p.typ)
	m.Index = i
	return
}

func (t *interfaceType) NumMethod() int { return len(t.methods) }

// MethodByName method with the given name in the type's method set.
func (t *interfaceType) MethodByName(name string) (m Method, ok bool) {
	if t == nil {
		return
	}
	var p *imethod
	for i := range t.methods {
		p = &t.methods[i]
		if *p.name == name {
			return t.Method(i), true
		}
	}
	return
}

func (t *uncommonType) exportedMethods() []method {
	methodsi, found := methodCache.Load(t)
	if found {
		return methodsi.([]method)
	}

	allm := t.methods
	allExported := true
	for _, m := range allm {
		if m.pkgPath != nil {
			allExported = false
			break
		}
	}
	var methods []method
	if allExported {
		methods = allm
	} else {
		methods = make([]method, 0, len(allm))
		for _, m := range allm {
			if m.pkgPath == nil {
				methods = append(methods, m)
			}
		}
		methods = methods[:len(methods):len(methods)]
	}

	methodsi, _ = methodCache.LoadOrStore(t, methods)
	return methodsi.([]method)
}

func implements(T, V *rtype) bool {
	if T.Kind() != Interface {
		return false
	}
	t := (*interfaceType)(unsafe.Pointer(T))
	if len(t.methods) == 0 {
		return true
	}

	// The same algorithm applies in both cases, but the
	// method tables for an interface type and a concrete type
	// are different, so the code is duplicated.
	// In both cases the algorithm is a linear scan over the two
	// lists - T's methods and V's methods - simultaneously.
	// Since method tables are stored in a unique sorted order
	// (alphabetical, with no duplicate method names), the scan
	// through V's methods must hit a match for each of T's
	// methods along the way, or else V does not implement T.
	// This lets us run the scan in overall linear time instead of
	// the quadratic time  a naive search would require.
	// See also ../runtime/iface.go.
	if V.Kind() == Interface {
		v := (*interfaceType)(unsafe.Pointer(V))
		i := 0
		for j := 0; j < len(v.methods); j++ {
			tm := &t.methods[i]
			vm := &v.methods[j]
			if *vm.name == *tm.name && (vm.pkgPath == tm.pkgPath || (vm.pkgPath != nil && tm.pkgPath != nil && *vm.pkgPath == *tm.pkgPath)) && toType(vm.typ).common() == toType(tm.typ).common() {
				if i++; i >= len(t.methods) {
					return true
				}
			}
		}
		return false
	}

	v := V.uncommon()
	if v == nil {
		return false
	}
	i := 0
	for j := 0; j < len(v.methods); j++ {
		tm := &t.methods[i]
		vm := &v.methods[j]
		if *vm.name == *tm.name && (vm.pkgPath == tm.pkgPath || (vm.pkgPath != nil && tm.pkgPath != nil && *vm.pkgPath == *tm.pkgPath)) && toType(vm.mtyp).common() == toType(tm.typ).common() {
			if i++; i >= len(t.methods) {
				return true
			}
		}
	}
	return false
}

func specialChannelAssignability(T, V *rtype) bool {
	// Special case:
	// x is a bidirectional channel value, T is a channel type,
	// x's type V and T have identical element types,
	// and at least one of V or T is not a defined type.
	return V.ChanDir() == BothDir && (T.Name() == "" || V.Name() == "") && haveIdenticalType(T.Elem(), V.Elem(), true)
}

func directlyAssignable(T, V *rtype) bool {
	// x's type V is identical to T?
	if rtypeEqual(T, V) {
		return true
	}

	// Otherwise at least one of T and V must not be defined
	// and they must have the same kind.
	if T.hasName() && V.hasName() || T.Kind() != V.Kind() {
		return false
	}

	if T.Kind() == Chan && specialChannelAssignability(T, V) {
		return true
	}

	// x's type T and V must have identical underlying types.
	return haveIdenticalUnderlyingType(T, V, true)
}

func haveIdenticalType(T, V Type, cmpTags bool) bool {
	if cmpTags {
		return T == V
	}

	if T.Name() != V.Name() || T.Kind() != V.Kind() || T.PkgPath() != V.PkgPath() {
		return false
	}

	return haveIdenticalUnderlyingType(T.common(), V.common(), false)
}

func haveIdenticalUnderlyingType(T, V *rtype, cmpTags bool) bool {
	if rtypeEqual(T, V) {
		return true
	}

	kind := T.Kind()
	if kind != V.Kind() {
		return false
	}

	// Non-composite types of equal kind have same underlying type
	// (the predefined instance of the type).
	if Bool <= kind && kind <= Complex128 || kind == String || kind == UnsafePointer {
		return true
	}

	// Composite types.
	switch kind {
	case Array:
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Chan:
		return V.ChanDir() == T.ChanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Func:
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if t.dotdotdot != v.dotdotdot || len(t.in) != len(v.in) || len(t.out) != len(v.out) {
			return false
		}
		for i, typ := range t.in {
			if !haveIdenticalType(typ, v.in[i], cmpTags) {
				return false
			}
		}
		for i, typ := range t.out {
			if !haveIdenticalType(typ, v.out[i], cmpTags) {
				return false
			}
		}
		return true

	case Interface:
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.methods) == 0 && len(v.methods) == 0 {
			return true
		}
		// Might have the same methods but still
		// need a run time conversion.
		return false

	case Map:
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Pointer, Slice:
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Struct:
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.fields) != len(v.fields) {
			return false
		}
		for i := range t.fields {
			tf := &t.fields[i]
			vf := &v.fields[i]
			if tf.name != vf.name && (tf.name == nil || vf.name == nil || *tf.name != *vf.name) {
				return false
			}
			if tf.pkgPath != vf.pkgPath && (tf.pkgPath == nil || vf.pkgPath == nil || *tf.pkgPath != *vf.pkgPath) {
				return false
			}
			if !haveIdenticalType(tf.typ, vf.typ, cmpTags) {
				return false
			}
			if cmpTags && tf.tag != vf.tag && (tf.tag == nil || vf.tag == nil || *tf.tag != *vf.tag) {
				return false
			}
			if tf.offsetEmbed != vf.offsetEmbed {
				return false
			}
		}
		return true
	}

	return false
}

var methodCache sync.Map

func (t *rtype) common() *rtype { return t }

func (t *rtype) uncommon() *uncommonType { return t.uncommonType }
