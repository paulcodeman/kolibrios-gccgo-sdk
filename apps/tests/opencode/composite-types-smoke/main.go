package main

import (
	"kos"
	"reflect"
	"runtime"
)

type item struct {
	Value int
	Next  *item
}

type label interface{ Label() string }

type key struct {
	Name  string
	Value float64
}

//go:noinline
func largeArray(t reflect.Type) reflect.Value {
	v := reflect.New(t).Elem()
	v.Index(v.Len() - 1).Field(1).Set(reflect.ValueOf(&item{Value: 91}))
	return v
}

func assertion(f func(), message string) {
	defer func() {
		value := recover()
		e, ok := value.(*runtime.TypeAssertionError)
		check(ok && e.Error() == message, "typed assertion error and text")
		_, ok = value.(runtime.Error)
		check(ok, "assertion implements runtime.Error")
	}()
	f()
	panic("missing assertion panic")
}

func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}

func panics(f func()) (result bool) {
	defer func() { result = recover() != nil }()
	f()
	return false
}

func main() {
	console, ok := kos.OpenConsole("OpenCode reflect type constructors")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_COMPOSITE_TYPES_START")
	t := reflect.TypeOf(item{})
	check(reflect.PointerTo(t) == reflect.TypeOf((*item)(nil)), "known pointer identity")
	check(reflect.SliceOf(t) == reflect.TypeOf([]item(nil)), "known slice identity")
	check(reflect.ChanOf(reflect.BothDir, t) == reflect.TypeOf((chan item)(nil)), "known channel identity")
	check(reflect.ChanOf(reflect.RecvDir, t) == reflect.TypeOf((<-chan item)(nil)), "known receive channel identity")
	check(reflect.ChanOf(reflect.SendDir, t) == reflect.TypeOf((chan<- item)(nil)), "known send channel identity")

	st := reflect.SliceOf(reflect.SliceOf(t))
	check(st.Kind() == reflect.Slice && st.Elem().Elem() == t, "dynamic nested slice")
	check(st == reflect.SliceOf(reflect.SliceOf(t)), "dynamic slice cache")
	check(st.String() == "[][]main.item", "dynamic slice name")
	value := reflect.MakeSlice(st, 2, 3)
	child := reflect.MakeSlice(st.Elem(), 1, 1)
	child.Index(0).Field(0).SetInt(42)
	value.Index(0).Set(child)
	runtime.GC()
	check(value.Index(0).Index(0).Field(0).Int() == 42, "dynamic slice allocation and GC")
	pt := reflect.PointerTo(st)
	check(pt == reflect.PointerTo(st) && pt.Elem() == st && pt.Comparable(), "dynamic pointer metadata and cache")
	pointer := reflect.New(st)
	pointer.Elem().Set(value)
	check(pointer.Type() == pt && pointer.Elem().Index(0).Index(0).Field(0).Int() == 42, "dynamic pointer allocation")

	ct := reflect.ChanOf(reflect.BothDir, st)
	check(ct == reflect.ChanOf(reflect.BothDir, st) && ct.Elem() == st && ct.ChanDir() == reflect.BothDir, "dynamic channel cache")
	check(reflect.Zero(ct).IsNil(), "dynamic nil channel value")
	check(reflect.MapOf(reflect.TypeOf(""), t) == reflect.TypeOf(map[string]item(nil)), "known map identity")
	mt := reflect.MapOf(pt, st)
	check(mt == reflect.MapOf(pt, st) && mt.Key() == pt && mt.Elem() == st, "dynamic map cache and metadata")
	m := reflect.MakeMapWithSize(mt, 1)
	m.SetMapIndex(pointer, value)
	runtime.GC()
	check(m.Len() == 1 && m.MapIndex(pointer).Index(0).Index(0).Field(0).Int() == 42, "dynamic map allocation and GC")
	m.SetMapIndex(pointer, reflect.Value{})
	check(m.Len() == 0, "dynamic map deletion")
	check(panics(func() { reflect.MapOf(st, t) }), "invalid map key")
	kt := reflect.TypeOf(key{})
	sm := reflect.MakeMap(reflect.MapOf(kt, st))
	sm.SetMapIndex(reflect.ValueOf(key{Name: string([]byte{'a'}), Value: 0}), value)
	check(sm.MapIndex(reflect.ValueOf(key{Name: "a", Value: 0})).IsValid(), "composite key semantic hashing")
	check(reflect.ArrayOf(2, t) == reflect.TypeOf([2]item{}), "known array identity")
	at := reflect.ArrayOf(3, reflect.TypeOf(0))
	a, b := reflect.New(at).Elem(), reflect.New(at).Elem()
	a.Index(2).SetInt(42)
	b.Index(2).SetInt(42)
	check(a.Interface() == b.Interface(), "dynamic array equality closure")
	b.Index(2).SetInt(43)
	check(a.Interface() != b.Interface(), "dynamic array inequality")
	check(at == reflect.ArrayOf(3, reflect.TypeOf(0)), "dynamic array cache")
	big := reflect.ArrayOf(3000, t)
	large := largeArray(big)
	runtime.GC()
	check(large.Index(2999).Field(1).Elem().Field(0).Int() == 91, "large array GC program")
	nested := reflect.New(reflect.ArrayOf(2, big)).Elem()
	nested.Index(1).Set(large)
	runtime.GC()
	check(nested.Index(1).Index(2999).Field(1).Elem().Field(0).Int() == 91, "nested GC program repetition")
	check(panics(func() { reflect.ArrayOf(-1, t) }), "negative array length")
	check(panics(func() { reflect.ChanOf(reflect.ChanDir(0), t) }), "invalid channel direction")
	check(panics(func() { reflect.SliceOf(nil) }), "nil slice element")
	check(panics(func() { reflect.PointerTo(nil) }), "nil pointer element")
	assertion(func() { var v any; _ = v.(int) }, "interface conversion: interface {} is nil, not int")
	assertion(func() { var v any = item{}; _ = v.(int) }, "interface conversion: interface {} is main.item, not int")
	assertion(func() { var v any; _ = v.(label) }, "interface conversion: interface is nil, not main.label")
	assertion(func() { var v any = 1; _ = v.(label) }, "interface conversion: int is not main.label: missing method Label")
	kos.DebugString("OPENCODE_COMPOSITE_TYPES_PASS")
	console.WriteString("OPENCODE_COMPOSITE_TYPES_PASS\n")
}
