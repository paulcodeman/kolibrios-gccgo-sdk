// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package intern

import "sync"

// This is upstream's safe, strong-reference mode. Weak-reference interning
// has not yet been validated against the SDK collector and finalizer queue.
// Values (including netip IPv6 zone names) remain interned for process lifetime.
type Value struct {
	_      [0]func()
	cmpVal any
}

func (v *Value) Get() any { return v.cmpVal }

type key struct {
	s        string
	cmpVal   any
	isString bool
}

func keyFor(cmpVal any) key {
	if s, ok := cmpVal.(string); ok {
		return key{s: s, isString: true}
	}
	return key{cmpVal: cmpVal}
}

func (k key) Value() *Value {
	if k.isString {
		return &Value{cmpVal: k.s}
	}
	return &Value{cmpVal: k.cmpVal}
}

var (
	mu      sync.Mutex
	valSafe = map[key]*Value{}
)

func Get(cmpVal any) *Value       { return get(keyFor(cmpVal)) }
func GetByString(s string) *Value { return get(key{s: s, isString: true}) }

func get(k key) *Value {
	mu.Lock()
	defer mu.Unlock()
	v := valSafe[k]
	if v != nil {
		return v
	}
	v = k.Value()
	valSafe[k] = v
	return v
}
