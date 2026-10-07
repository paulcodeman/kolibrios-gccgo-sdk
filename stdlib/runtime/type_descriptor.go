// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "unsafe"

// tflag is documented in reflect/type.go.
//
// tflag values must be kept in sync with copies in:
//
//		go/types.cc
//		reflect/type.go
//	     internal/reflectlite/type.go
type tflag uint8

const (
	tflagRegularMemory tflag = 1 << 3 // equal and hash can treat values of this type as a single region of t.size bytes
)

// Needs to be in sync with
// go/types.cc
// ../reflect/type.go:/^type.rtype.
// ../internal/reflectlite/type.go:/^type.rtype.
type _type struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata  *byte
	_string *string
	*uncommontype
	ptrToThis *_type
}

func (t *_type) string() string {
	// For gccgo, try to strip out quoted strings.
	s := *t._string
	q := false
	started := false
	var start int
	var end int
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			q = !q
		} else if !q {
			if !started {
				start = i
				started = true
			}
			end = i
		}
	}
	return s[start : end+1]
}

// pkgpath returns the path of the package where t was defined, if
// available. This is not the same as the reflect package's PkgPath
// method, in that it returns the package path for struct and interface
// types, not just named types.
func (t *_type) pkgpath() string {
	if u := t.uncommontype; u != nil {
		if u.pkgPath == nil {
			return ""
		}
		return *u.pkgPath
	}
	return ""
}

type method struct {
	name    *string
	pkgPath *string
	mtyp    *_type
	typ     *_type
	tfn     unsafe.Pointer
}

type uncommontype struct {
	name    *string
	pkgPath *string
	methods []method
}
