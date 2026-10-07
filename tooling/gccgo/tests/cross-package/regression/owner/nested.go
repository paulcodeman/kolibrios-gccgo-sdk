package owner

// The private fields themselves have instantiated generic named types.
// Their ownership must survive instantiation of NewNested in a consumer.
type Nested[T any] struct {
	inner   Box[T]
	backing Box[[]T]
}

func NewNested[T any](value T) Nested[T] {
	return Nested[T]{inner: Make(value), backing: Make([]T{value})}
}

func (value Nested[T]) Get() T     { return value.inner.Get() }
func (value Nested[T]) Slice() []T { return value.backing.Get() }
