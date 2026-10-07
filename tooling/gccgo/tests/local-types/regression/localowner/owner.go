package localowner

type Box[T any] struct{ Value T }

func Make[T any](v T) Box[T] { return Box[T]{v} }

func First() any {
	type item struct{ value int }
	return Make(item{7})
}

func Second() any {
	type item struct{ value int }
	return Make(item{11})
}

func FirstAgain() any { return First() }

type Iterator[T any] struct{ Limit int }
type Mapper[T, R any] Iterator[T]

func (it Iterator[T]) Each(input []T, f func(int, *T)) {
	for i := range input {
		f(i, &input[i])
	}
}

func (m Mapper[T, R]) Map(input []T, f func(*T) R) []R {
	out := make([]R, len(input))
	Iterator[T](m).Each(input, func(i int, value *T) { out[i] = f(value) })
	return out
}

func Map[T, R any](input []T, f func(*T) R) []R {
	return Mapper[T, R](Iterator[T]{}).Map(input, f)
}
