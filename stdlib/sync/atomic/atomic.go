package atomic

import "unsafe"

// Note: This implementation provides the sync/atomic API for KolibriOS
// in a single-threaded bootstrap context. Operations are not atomic and
// provide no memory ordering guarantees.

func SwapInt32(addr *int32, new int32) (old int32) {
	old = *addr
	*addr = new
	return old
}

func SwapInt64(addr *int64, new int64) (old int64) {
	old = *addr
	*addr = new
	return old
}

func SwapUint32(addr *uint32, new uint32) (old uint32) {
	old = *addr
	*addr = new
	return old
}

func SwapUint64(addr *uint64, new uint64) (old uint64) {
	old = *addr
	*addr = new
	return old
}

func SwapUintptr(addr *uintptr, new uintptr) (old uintptr) {
	old = *addr
	*addr = new
	return old
}

func SwapPointer(addr *unsafe.Pointer, new unsafe.Pointer) (old unsafe.Pointer) {
	old = *addr
	*addr = new
	return old
}

func CompareAndSwapInt32(addr *int32, old, new int32) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapInt64(addr *int64, old, new int64) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapUint32(addr *uint32, old, new uint32) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapUint64(addr *uint64, old, new uint64) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapUintptr(addr *uintptr, old, new uintptr) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func CompareAndSwapPointer(addr *unsafe.Pointer, old, new unsafe.Pointer) (swapped bool) {
	if *addr != old {
		return false
	}
	*addr = new
	return true
}

func AddInt32(addr *int32, delta int32) (new int32) {
	*addr += delta
	return *addr
}

func AddUint32(addr *uint32, delta uint32) (new uint32) {
	*addr += delta
	return *addr
}

func AddInt64(addr *int64, delta int64) (new int64) {
	*addr += delta
	return *addr
}

func AddUint64(addr *uint64, delta uint64) (new uint64) {
	*addr += delta
	return *addr
}

func AddUintptr(addr *uintptr, delta uintptr) (new uintptr) {
	*addr += delta
	return *addr
}

func LoadInt32(addr *int32) (val int32) {
	return *addr
}

func LoadInt64(addr *int64) (val int64) {
	return *addr
}

func LoadUint32(addr *uint32) (val uint32) {
	return *addr
}

func LoadUint64(addr *uint64) (val uint64) {
	return *addr
}

func LoadUintptr(addr *uintptr) (val uintptr) {
	return *addr
}

func LoadPointer(addr *unsafe.Pointer) (val unsafe.Pointer) {
	return *addr
}

func StoreInt32(addr *int32, val int32) {
	*addr = val
}

func StoreInt64(addr *int64, val int64) {
	*addr = val
}

func StoreUint32(addr *uint32, val uint32) {
	*addr = val
}

func StoreUint64(addr *uint64, val uint64) {
	*addr = val
}

func StoreUintptr(addr *uintptr, val uintptr) {
	*addr = val
}

func StorePointer(addr *unsafe.Pointer, val unsafe.Pointer) {
	*addr = val
}

// Value provides a minimal implementation of atomic.Value.
type Value struct {
	v any
	t unsafe.Pointer
}

type ifaceWords struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

func (v *Value) Load() (val any) {
	if v == nil || v.t == nil {
		return nil
	}
	return v.v
}

func (v *Value) Store(val any) {
	if val == nil {
		panic("sync/atomic: store of nil value into Value")
	}
	if v == nil {
		return
	}
	typ := (*ifaceWords)(unsafe.Pointer(&val)).typ
	if v.t != nil && v.t != typ {
		panic("sync/atomic: store of inconsistently typed value into Value")
	}
	v.v = val
	v.t = typ
}

func (v *Value) Swap(new any) (old any) {
	if new == nil {
		panic("sync/atomic: swap of nil value into Value")
	}
	if v == nil {
		return nil
	}
	typ := (*ifaceWords)(unsafe.Pointer(&new)).typ
	if v.t != nil && v.t != typ {
		panic("sync/atomic: swap of inconsistently typed value into Value")
	}
	old = v.v
	v.v = new
	v.t = typ
	return old
}

func (v *Value) CompareAndSwap(old, new any) (swapped bool) {
	if new == nil {
		panic("sync/atomic: compare-and-swap of nil value into Value")
	}
	if v == nil {
		return false
	}
	oldTyp := (*ifaceWords)(unsafe.Pointer(&old)).typ
	newTyp := (*ifaceWords)(unsafe.Pointer(&new)).typ
	if v.t == nil {
		if old != nil {
			return false
		}
		v.v = new
		v.t = newTyp
		return true
	}
	if v.t != oldTyp || v.t != newTyp {
		if v.t != newTyp {
			panic("sync/atomic: compare-and-swap of inconsistently typed value into Value")
		}
		return false
	}
	if v.v != old {
		return false
	}
	v.v = new
	return true
}
