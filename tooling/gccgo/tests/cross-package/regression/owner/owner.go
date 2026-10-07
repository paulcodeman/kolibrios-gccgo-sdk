package owner

import (
	"bytes"
	digest "hash/fnv"
	"regression/legacy"
	"unsafe"
)

type Sequence[T any] func(func(T) bool)

func Legacy[T ~int](value T) legacy.Record { return legacy.Record{Value: int(value)} }

func privateSequence() Sequence[int] { return nil }

func privatePointer(pointer unsafe.Pointer) unsafe.Pointer { return pointer }

func PointerRoundTrip[T any](pointer *T) *T {
	return (*T)(privatePointer(unsafe.Pointer(pointer)))
}

func Buffer[T any](value T) *bytes.Buffer {
	buffer := &bytes.Buffer{}
	buffer.WriteString("buffer")
	return buffer
}

const offset = 3

type Box[T any] struct {
	Value  T
	hidden T
}

func Make[T any](v T) Box[T]   { return Box[T]{Value: v, hidden: v} }
func (b Box[T]) privateGet() T { return b.hidden }
func (b Box[T]) Get() T        { return b.privateGet() }

func Add[T ~int](v T) T { return v + offset }

type state struct{ value int }

var stored int

func addOne(v int) int { return v + 1 }
func Touch[T ~int](v T) T {
	s := state{value: int(v)}
	s.value++
	stored = s.value
	return T(addOne(stored))
}
func Stored() int                { return stored }
func Pointer[T ~int](v T) *state { return &state{value: int(v)} }
func Read(s *state) int          { return s.value }
func privateVariadic(values ...int) int {
	sum := 0
	for _, value := range values {
		sum += value
	}
	return sum
}
func privateFunction() func(...int) int { return privateVariadic }
func Variadic[T ~int](v T) T            { return T(privateFunction()(int(v), 2, 3)) }
func Digest[T ~string](v T) uint32 {
	h := digest.New32a()
	h.Write([]byte(v))
	return h.Sum32()
}
