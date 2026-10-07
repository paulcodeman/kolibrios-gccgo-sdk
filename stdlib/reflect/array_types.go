// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Array type construction from libgo; arrayAt uses the SDK's equivalent signature.
package reflect

import (
	"internal/goarch"
	"strconv"
	"unsafe"
)

const maxPtrmaskBytes = 2048

// ArrayOf returns the array type with the given length and element type.
// For example, if t represents int, ArrayOf(5, t) represents [5]int.
//
// If the resulting type would be larger than the available address space,
// ArrayOf panics.
func ArrayOf(length int, elem Type) Type {
	if length < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}

	typ := elem.(*rtype)

	// Look in cache.
	ckey := cacheKey{Array, typ, nil, uintptr(length)}
	if array, ok := lookupCache.Load(ckey); ok {
		return array.(Type)
	}

	// Look in known types.
	s := "[" + strconv.Itoa(length) + "]" + *typ.string
	if tt := lookupType(s); tt != nil {
		array := (*arrayType)(unsafe.Pointer(toType(tt).(*rtype)))
		if array.elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make an array type.
	var iarray any = [1]unsafe.Pointer{}
	prototype := *(**arrayType)(unsafe.Pointer(&iarray))
	array := *prototype
	array.tflag = typ.tflag & tflagRegularMemory
	array.string = &s

	// gccgo uses a different hash.
	// array.hash = fnv1(typ.hash, '[')
	// for n := uint32(length); n > 0; n >>= 8 {
	// 	array.hash = fnv1(array.hash, byte(n))
	// }
	// array.hash = fnv1(array.hash, ']')
	array.hash = typ.hash + 1 + 13
	array.elem = typ
	array.ptrToThis = nil
	if typ.size > 0 {
		max := ^uintptr(0) / typ.size
		if uintptr(length) > max {
			panic("reflect.ArrayOf: array size would exceed virtual address space")
		}
	}
	array.size = typ.size * uintptr(length)
	if length > 0 && typ.ptrdata != 0 {
		array.ptrdata = typ.size*uintptr(length-1) + typ.ptrdata
	}
	array.align = typ.align
	array.fieldAlign = typ.fieldAlign
	array.uncommonType = nil
	array.len = uintptr(length)
	array.slice = SliceOf(elem).(*rtype)

	switch {
	case typ.ptrdata == 0 || array.size == 0:
		// No pointers.
		array.gcdata = nil
		array.ptrdata = 0

	case length == 1:
		// In memory, 1-element array looks just like the element.
		array.kind |= typ.kind & kindGCProg
		array.gcdata = typ.gcdata
		array.ptrdata = typ.ptrdata

	case typ.kind&kindGCProg == 0 && array.size <= maxPtrmaskBytes*8*goarch.PtrSize:
		// Element is small with pointer mask; array is still small.
		// Create direct pointer mask by turning each 1 bit in elem
		// into length 1 bits in larger mask.
		mask := make([]byte, (array.ptrdata/goarch.PtrSize+7)/8)
		emitGCMask(mask, 0, typ, array.len)
		array.gcdata = &mask[0]

	default:
		// Create program that emits one element
		// and then repeats to make the array.
		prog := []byte{0, 0, 0, 0} // will be length of prog
		prog = appendGCProg(prog, typ)
		// Pad from ptrdata to size.
		elemPtrs := typ.ptrdata / goarch.PtrSize
		elemWords := typ.size / goarch.PtrSize
		if elemPtrs < elemWords {
			// Emit literal 0 bit, then repeat as needed.
			prog = append(prog, 0x01, 0x00)
			if elemPtrs+1 < elemWords {
				prog = append(prog, 0x81)
				prog = appendVarint(prog, elemWords-elemPtrs-1)
			}
		}
		// Repeat length-1 times.
		if elemWords < 0x80 {
			prog = append(prog, byte(elemWords|0x80))
		} else {
			prog = append(prog, 0x80)
			prog = appendVarint(prog, elemWords)
		}
		prog = appendVarint(prog, uintptr(length)-1)
		prog = append(prog, 0)
		*(*uint32)(unsafe.Pointer(&prog[0])) = uint32(len(prog) - 4)
		array.kind |= kindGCProg
		array.gcdata = &prog[0]
		array.ptrdata = array.size // overestimate but ok; must match program
	}

	etyp := typ.common()
	esize := typ.size

	array.equal = nil
	if eequal := etyp.equal; eequal != nil {
		array.equal = func(p, q unsafe.Pointer) bool {
			for i := 0; i < length; i++ {
				pi := arrayAt(p, i, esize)
				qi := arrayAt(q, i, esize)
				if !eequal(pi, qi) {
					return false
				}
			}
			return true
		}
	}

	switch {
	case length == 1 && !ifaceIndir(typ):
		// array of 1 direct iface type can be direct
		array.kind |= kindDirectIface
	default:
		array.kind &^= kindDirectIface
	}

	ti, _ := lookupCache.LoadOrStore(ckey, toType(&array.rtype).(*rtype))
	return ti.(Type)
}

func appendVarint(x []byte, v uintptr) []byte {
	for ; v >= 0x80; v >>= 7 {
		x = append(x, byte(v|0x80))
	}
	x = append(x, byte(v))
	return x
}
