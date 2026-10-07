package main

type scalar interface{ ~int | ~int32 | ~string }
type number int32
type numberAlias = number
type valueAlias = any

var copied []int

func identity[T any](v T) T { return v }
func minimum[T scalar](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func clone[S ~[]E, E any](v S) S { return append(S(nil), v...) }
func nested[T scalar](a, b T) T  { return minimum(identity(a), identity[T](b)) }
func recursive[T ~int](v T) T {
	if v == 0 {
		return v
	}
	return v + recursive(v-1)
}

type box[T any] struct{ value T }

func (b box[T]) get() T       { return b.value }
func (b *box[T]) set(value T) { b.value = value }

type pair[K, V any] struct {
	key   K
	value V
}

func (p pair[K, V]) values() (K, V) { return p.key, p.value }

type node[T any] struct {
	value T
	next  *node[T]
}

func main() {
	aliases := []valueAlias{numberAlias(7), "alias"}
	if identity(aliases)[0] != number(7) || (box[valueAlias]{value: "value"}).get() != "value" {
		panic("generic alias arguments")
	}
	if identity(3) != 3 || identity[string]("ok") != "ok" {
		panic("generic identity")
	}
	if minimum(number(7), number(2)) != 2 || nested("b", "a") != "a" {
		panic("generic constraints and nested inference")
	}
	x := []int{1, 2, 3}
	y := clone(x)
	copied = y
	y[0] = 9
	if x[0] != 1 || len(copied) != 3 || recursive(3) != 6 {
		panic("generic slice inference and recursion")
	}
	fn := identity[int]
	if fn(5) != 5 {
		panic("generic function value")
	}
	b := box[string]{value: "old"}
	b.set("new")
	if b.get() != "new" {
		panic("generic type methods")
	}
	p := pair[int, string]{key: 7, value: "v"}
	k, v := p.values()
	if k != 7 || v != "v" {
		panic("generic multiple receiver parameters")
	}
	n := node[int]{value: 1, next: &node[int]{value: 2}}
	if n.next.value != 2 {
		panic("generic recursive type")
	}
	println("GENERICS_PASS")
}
