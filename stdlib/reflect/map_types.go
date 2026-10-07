// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Map type construction and GC metadata from GCC 13.3.0 libgo reflect/type.go.
package reflect

import (
	"internal/goarch"
	"unsafe"
)

const kindGCProg = 1 << 6

//go:noescape
func typehash(t *rtype, p unsafe.Pointer, h uintptr) uintptr

// MapOf returns the map type with the given key and element types.
// For example, if k represents int and e represents string,
// MapOf(k, e) represents map[int]string.
//
// If the key type is not a valid map key type (that is, if it does
// not implement Go's == operator), MapOf panics.
func MapOf(key, elem Type) Type {
	ktyp := key.(*rtype)
	etyp := elem.(*rtype)

	if ktyp.equal == nil {
		panic("reflect.MapOf: invalid key type " + ktyp.String())
	}

	// Look in cache.
	ckey := cacheKey{Map, ktyp, etyp, 0}
	if mt, ok := lookupCache.Load(ckey); ok {
		return mt.(Type)
	}

	// Look in known types.
	s := "map[" + *ktyp.string + "]" + *etyp.string
	if tt := lookupType(s); tt != nil {
		mt := (*mapType)(unsafe.Pointer(toType(tt).(*rtype)))
		if mt.key == ktyp && mt.elem == etyp {
			ti, _ := lookupCache.LoadOrStore(ckey, tt)
			return ti.(Type)
		}
	}

	// Make a map type.
	// Note: flag values must match those used in the TMAP case
	// in ../cmd/compile/internal/reflectdata/reflect.go:writeType.
	var imap any = (map[unsafe.Pointer]unsafe.Pointer)(nil)
	mt := **(**mapType)(unsafe.Pointer(&imap))
	mt.string = &s

	// gccgo uses a different hash
	// mt.hash = fnv1(etyp.hash, 'm', byte(ktyp.hash>>24), byte(ktyp.hash>>16), byte(ktyp.hash>>8), byte(ktyp.hash))
	mt.hash = ktyp.hash + etyp.hash + 2 + 14

	mt.key = ktyp
	mt.elem = etyp
	mt.uncommonType = nil
	mt.ptrToThis = nil

	mt.bucket = bucketOf(ktyp, etyp)
	mt.hasher = func(p unsafe.Pointer, seed uintptr) uintptr {
		return typehash(ktyp, p, seed)
	}
	mt.flags = 0
	if ktyp.size > maxKeySize {
		mt.keysize = uint8(goarch.PtrSize)
		mt.flags |= 1 // indirect key
	} else {
		mt.keysize = uint8(ktyp.size)
	}
	if etyp.size > maxValSize {
		mt.valuesize = uint8(goarch.PtrSize)
		mt.flags |= 2 // indirect value
	} else {
		mt.valuesize = uint8(etyp.size)
	}
	mt.bucketsize = uint16(mt.bucket.size)
	if isReflexive(ktyp) {
		mt.flags |= 4
	}
	if needKeyUpdate(ktyp) {
		mt.flags |= 8
	}
	if hashMightPanic(ktyp) {
		mt.flags |= 16
	}

	ti, _ := lookupCache.LoadOrStore(ckey, toType(&mt.rtype).(*rtype))
	return ti.(Type)
}

// isReflexive reports whether the == operation on the type is reflexive.
// That is, x == x for all values x of type t.
func isReflexive(t *rtype) bool {
	switch t.Kind() {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Pointer, String, UnsafePointer:
		return true
	case Float32, Float64, Complex64, Complex128, Interface:
		return false
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return isReflexive(tt.elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if !isReflexive(f.typ) {
				return false
			}
		}
		return true
	default:
		// Func, Map, Slice, Invalid
		panic("isReflexive called on non-key type " + t.String())
	}
}

// needKeyUpdate reports whether map overwrites require the key to be copied.
func needKeyUpdate(t *rtype) bool {
	switch t.Kind() {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Pointer, UnsafePointer:
		return false
	case Float32, Float64, Complex64, Complex128, Interface, String:
		// Float keys can be updated from +0 to -0.
		// String keys can be updated to use a smaller backing store.
		// Interfaces might have floats of strings in them.
		return true
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return needKeyUpdate(tt.elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if needKeyUpdate(f.typ) {
				return true
			}
		}
		return false
	default:
		// Func, Map, Slice, Invalid
		panic("needKeyUpdate called on non-key type " + t.String())
	}
}

// hashMightPanic reports whether the hash of a map key of type t might panic.
func hashMightPanic(t *rtype) bool {
	switch t.Kind() {
	case Interface:
		return true
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return hashMightPanic(tt.elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.fields {
			if hashMightPanic(f.typ) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Make sure these routines stay in sync with ../../runtime/map.go!
// These types exist only for GC, so we only fill out GC relevant info.
// Currently, that's just size and the GC program. We also fill in string
// for possible debugging use.
const (
	bucketSize uintptr = 8
	maxKeySize uintptr = 128
	maxValSize uintptr = 128
)

func bucketOf(ktyp, etyp *rtype) *rtype {
	if ktyp.size > maxKeySize {
		ktyp = PointerTo(ktyp).(*rtype)
	}
	if etyp.size > maxValSize {
		etyp = PointerTo(etyp).(*rtype)
	}

	// Prepare GC data if any.
	// A bucket is at most bucketSize*(1+maxKeySize+maxValSize)+2*goarch.PtrSize bytes,
	// or 2072 bytes, or 259 pointer-size words, or 33 bytes of pointer bitmap.
	// Note that since the key and value are known to be <= 128 bytes,
	// they're guaranteed to have bitmaps instead of GC programs.
	var gcdata *byte
	var ptrdata uintptr

	size := bucketSize
	size = align(size, uintptr(ktyp.fieldAlign))
	size += bucketSize * ktyp.size
	size = align(size, uintptr(etyp.fieldAlign))
	size += bucketSize * etyp.size

	maxAlign := uintptr(ktyp.fieldAlign)
	if maxAlign < uintptr(etyp.fieldAlign) {
		maxAlign = uintptr(etyp.fieldAlign)
	}
	if maxAlign > goarch.PtrSize {
		size = align(size, maxAlign)
		size += align(goarch.PtrSize, maxAlign) - goarch.PtrSize
	} else if maxAlign < goarch.PtrSize {
		size = align(size, goarch.PtrSize)
		maxAlign = goarch.PtrSize
	}

	ovoff := size
	size += goarch.PtrSize

	if ktyp.ptrdata != 0 || etyp.ptrdata != 0 {
		nptr := size / goarch.PtrSize
		mask := make([]byte, (nptr+7)/8)
		psize := bucketSize
		psize = align(psize, uintptr(ktyp.fieldAlign))
		base := psize / goarch.PtrSize

		if ktyp.ptrdata != 0 {
			emitGCMask(mask, base, ktyp, bucketSize)
		}
		psize += bucketSize * ktyp.size
		psize = align(psize, uintptr(etyp.fieldAlign))
		base = psize / goarch.PtrSize

		if etyp.ptrdata != 0 {
			emitGCMask(mask, base, etyp, bucketSize)
		}

		word := ovoff / goarch.PtrSize
		mask[word/8] |= 1 << (word % 8)
		gcdata = &mask[0]
		ptrdata = (word + 1) * goarch.PtrSize

		// overflow word must be last
		if ptrdata != size {
			panic("reflect: bad layout computation in MapOf")
		}
	}

	b := &rtype{
		align:      uint8(maxAlign),
		fieldAlign: uint8(maxAlign),
		size:       size,
		kind:       uint8(Struct),
		ptrdata:    ptrdata,
		gcdata:     gcdata,
	}
	s := "bucket(" + *ktyp.string + "," + *etyp.string + ")"
	b.string = &s
	return b
}

func (t *rtype) gcSlice(begin, end uintptr) []byte {
	return (*[1 << 30]byte)(unsafe.Pointer(t.gcdata))[begin:end:end]
}

// emitGCMask writes the GC mask for [n]typ into out, starting at bit
// offset base.
func emitGCMask(out []byte, base uintptr, typ *rtype, n uintptr) {
	if typ.kind&kindGCProg != 0 {
		panic("reflect: unexpected GC program")
	}
	ptrs := typ.ptrdata / goarch.PtrSize
	words := typ.size / goarch.PtrSize
	mask := typ.gcSlice(0, (ptrs+7)/8)
	for j := uintptr(0); j < ptrs; j++ {
		if (mask[j/8]>>(j%8))&1 != 0 {
			for i := uintptr(0); i < n; i++ {
				k := base + i*words + j
				out[k/8] |= 1 << (k % 8)
			}
		}
	}
}

// appendGCProg appends the GC program for the first ptrdata bytes of
// typ to dst and returns the extended slice.
func appendGCProg(dst []byte, typ *rtype) []byte {
	if typ.kind&kindGCProg != 0 {
		// Element has GC program; emit one element.
		n := uintptr(*(*uint32)(unsafe.Pointer(typ.gcdata)))
		prog := typ.gcSlice(4, 4+n-1)
		return append(dst, prog...)
	}

	// Element is small with pointer mask; use as literal bits.
	ptrs := typ.ptrdata / goarch.PtrSize
	mask := typ.gcSlice(0, (ptrs+7)/8)

	// Emit 120-bit chunks of full bytes (max is 127 but we avoid using partial bytes).
	for ; ptrs > 120; ptrs -= 120 {
		dst = append(dst, 120)
		dst = append(dst, mask[:15]...)
		mask = mask[15:]
	}

	dst = append(dst, byte(ptrs))
	dst = append(dst, mask...)
	return dst
}
