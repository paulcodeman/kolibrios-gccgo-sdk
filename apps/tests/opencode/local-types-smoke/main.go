package main

import (
	"kos"
	"reflect"
	"regression/localowner"
)

var roots = []*int{new(int)}

func identity[T any](v T) T { return v }

func mapItems[T any](items []T, f func(*T) int) []int {
	out := make([]int, len(items))
	for i := range items {
		out[i] = f(&items[i])
	}
	return out
}

func main() {
	kos.DebugString("OPENCODE_LOCAL_TYPES_START")
	console, ok := kos.OpenConsole("OpenCode local generic type test")
	if !ok {
		panic("console unavailable")
	}
	defer console.Close()
	*roots[0] = 1
	type item struct{ value int }
	items := []item{{19}, {23}}
	values := mapItems(items, func(v *item) int { return v.value })
	if values[0] != 19 || values[1] != 23 {
		panic("local search item callback")
	}
	mapped := localowner.Map(items, func(v *item) int { return v.value + 1 })
	if mapped[0] != 20 || mapped[1] != 24 {
		panic("generic mapper conversion and callback")
	}
	b := localowner.Make(item{29})
	var exact localowner.Box[item] = b
	if identity(exact).Value.value != 29 {
		panic("local generic type")
	}
	t := reflect.TypeOf(b)
	if t.Name() != "Box[main.item]" || t.PkgPath() != "regression/localowner" {
		println(t.Name(), t.PkgPath())
		panic("local argument reflection")
	}
	outer := reflect.TypeOf(identity(item{}))
	if outer.Name() != "item" || outer.PkgPath() != "main" {
		panic("local type reflection")
	}
	{
		type item struct{ value int }
		inner := reflect.TypeOf(identity(item{}))
		if inner == outer || reflect.TypeOf(localowner.Make(item{})) == t {
			panic("shadowed local types collapsed")
		}
	}
	type alias = item
	if reflect.TypeOf(localowner.Make(alias{})) != t {
		panic("local alias identity")
	}
	type node struct {
		next  *node
		value item
	}
	n := &node{value: item{31}}
	n.next = n
	if identity(n).next != n || identity(n).value.value != 31 {
		panic("recursive local type")
	}
	const count = 3
	type vector [count + 2]int
	v := identity(vector{1, 2, 3, 4, 5})
	if len(v) != 5 || v[4] != 5 {
		panic("local constant array bound")
	}
	type embedded struct{ item }
	e := identity(embedded{item{37}})
	if e.item.value != 37 || e.value != 37 {
		panic("local embedded field")
	}
	f := reflect.TypeOf(e).Field(0)
	if f.Name != "item" || !f.Anonymous || f.PkgPath != "main" {
		panic("local embedded field reflection")
	}
	type embeddedAlias struct{ alias }
	a := identity(embeddedAlias{item{41}})
	if a.alias.value != 41 {
		panic("local embedded alias")
	}
	f = reflect.TypeOf(a).Field(0)
	if f.Name != "alias" || !f.Anonymous {
		panic("local embedded alias reflection")
	}
	if identity(map[item]int{{43}: 47})[item{43}] != 47 {
		panic("local type in map")
	}
	ch := make(chan item, 1)
	identity(ch) <- item{53}
	if (<-ch).value != 53 {
		panic("local type in channel")
	}
	first := reflect.TypeOf(localowner.First())
	if first == reflect.TypeOf(localowner.Second()) || first != reflect.TypeOf(localowner.FirstAgain()) {
		panic("owner local type identity")
	}
	kos.DebugString("OPENCODE_LOCAL_TYPES_PASS")
	console.WriteString("OPENCODE_LOCAL_TYPES_PASS\n")
}
