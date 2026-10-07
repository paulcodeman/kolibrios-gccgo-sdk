// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Pointer, slice and channel type constructors from GCC 13.3.0 libgo.
package reflect

import (
	"sync"
	"unsafe"
)

const tflagRegularMemory tflag = 1 << 3

// Implemented by the native runtime using compiler-emitted type lists.
func lookupType(s string) *rtype

// ptrMap is the cache for PointerTo.
var ptrMap sync.Map // map[*rtype]*ptrType

// PtrTo returns the pointer type with element t.
// For example, if t represents type Foo, PtrTo(t) represents *Foo.
//
// PtrTo is the old spelling of PointerTo.
// The two functions behave identically.
func PtrTo(t Type) Type { return PointerTo(t) }

// PointerTo returns the pointer type with element t.
// For example, if t represents type Foo, PointerTo(t) represents *Foo.
func PointerTo(t Type) Type {
	return t.(*rtype).ptrTo()
}

func (t *rtype) ptrTo() *rtype {
	if p := t.ptrToThis; p != nil {
		return p
	}

	// Check the cache.
	if pi, ok := ptrMap.Load(t); ok {
		return &pi.(*ptrType).rtype
	}

	// Look in known types.
	s := "*" + *t.string
	if tt := lookupType(s); tt != nil {
		p := (*ptrType)(unsafe.Pointer(toType(tt).(*rtype)))
		if p.elem == t {
			pi, _ := ptrMap.LoadOrStore(t, p)
			return &pi.(*ptrType).rtype
		}
	}

	// Create a new ptrType starting with the description
	// of an *unsafe.Pointer.
	var iptr any = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	pp.string = &s
	pp.ptrToThis = nil

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	// p.hash = fnv1(t.hash, '*')
	// This is the gccgo version.
	pp.hash = (t.hash << 4) + 9

	pp.uncommonType = nil
	pp.ptrToThis = nil
	pp.elem = t

	q := toType(&pp.rtype).(*rtype)
	p := (*ptrType)(unsafe.Pointer(q))
	pi, _ := ptrMap.LoadOrStore(t, p)
	return &pi.(*ptrType).rtype
}

// The lookupCache caches ArrayOf, ChanOf, MapOf and SliceOf lookups.
var lookupCache sync.Map // map[cacheKey]*rtype

// A cacheKey is the key for use in the lookupCache.
// Four values describe any of the types we are looking for:
// type kind, one or two subtypes, and an extra integer.
type cacheKey struct {
	kind  Kind
	t1    *rtype
	t2    *rtype
	extra uintptr
}

// ChanOf returns the channel type with the given direction and element type.
// For example, if t represents int, ChanOf(RecvDir, t) represents <-chan int.
//
// The gc runtime imposes a limit of 64 kB on channel element types.
// If t's size is equal to or exceeds this limit, ChanOf panics.
func ChanOf(dir ChanDir, t Type) Type {
	typ := t.(*rtype)

	// Look in cache.
	ckey := cacheKey{Chan, typ, nil, uintptr(dir)}
	if ch, ok := lookupCache.Load(ckey); ok {
		return ch.(*rtype)
	}

	// This restriction is imposed by the gc compiler and the runtime.
	if typ.size >= 1<<16 {
		panic("reflect.ChanOf: element size too large")
	}

	// Look in known types.
	var s string
	switch dir {
	default:
		panic("reflect.ChanOf: invalid dir")
	case SendDir:
		s = "chan<- " + *typ.string
	case RecvDir:
		s = "<-chan " + *typ.string
	case BothDir:
		typeStr := *typ.string
		if typeStr[0] == '<' {
			// typ is recv chan, need parentheses as "<-" associates with leftmost
			// chan possible, see:
			// * https://golang.org/ref/spec#Channel_types
			// * https://github.com/golang/go/issues/39897
			s = "chan (" + typeStr + ")"
		} else {
			s = "chan " + typeStr
		}
	}
	if tt := lookupType(s); tt != nil {
		ch := (*chanType)(unsafe.Pointer(toType(tt).(*rtype)))
		if ch.elem == typ && ch.dir == uintptr(dir) {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make a channel type.
	var ichan any = (chan unsafe.Pointer)(nil)
	prototype := *(**chanType)(unsafe.Pointer(&ichan))
	ch := *prototype
	ch.tflag = tflagRegularMemory
	ch.dir = uintptr(dir)
	ch.string = &s

	// gccgo uses a different hash.
	// ch.hash = fnv1(typ.hash, 'c', byte(dir))
	ch.hash = 0
	if dir&SendDir != 0 {
		ch.hash += 1
	}
	if dir&RecvDir != 0 {
		ch.hash += 2
	}
	ch.hash += typ.hash << 2
	ch.hash <<= 3
	ch.hash += 15

	ch.elem = typ
	ch.uncommonType = nil
	ch.ptrToThis = nil

	ti, _ := lookupCache.LoadOrStore(ckey, toType(&ch.rtype).(*rtype))
	return ti.(Type)
}

// SliceOf returns the slice type with element type t.
// For example, if t represents int, SliceOf(t) represents []int.
func SliceOf(t Type) Type {
	typ := t.(*rtype)

	// Look in cache.
	ckey := cacheKey{Slice, typ, nil, 0}
	if slice, ok := lookupCache.Load(ckey); ok {
		return slice.(Type)
	}

	// Look in known types.
	s := "[]" + *typ.string
	if tt := lookupType(s); tt != nil {
		slice := (*sliceType)(unsafe.Pointer(toType(tt).(*rtype)))
		if slice.elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make a slice type.
	var islice any = ([]unsafe.Pointer)(nil)
	prototype := *(**sliceType)(unsafe.Pointer(&islice))
	slice := *prototype
	slice.string = &s

	// gccgo uses a different hash.
	// slice.hash = fnv1(typ.hash, '[')
	slice.hash = typ.hash + 1 + 13

	slice.elem = typ
	slice.uncommonType = nil
	slice.ptrToThis = nil

	ti, _ := lookupCache.LoadOrStore(ckey, toType(&slice.rtype).(*rtype))
	return ti.(Type)
}
