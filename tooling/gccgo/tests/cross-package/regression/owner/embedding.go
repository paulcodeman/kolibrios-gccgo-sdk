package owner

type EmbeddedBox[T any] struct {
	Box[T]
}

func MakeEmbedded[T any](value T) EmbeddedBox[T] {
	return EmbeddedBox[T]{Box: Make(value)}
}

type privateBox[T any] struct{ Value T }

func (box privateBox[T]) Get() T { return box.Value }

type PrivateEmbedded struct {
	privateBox[int]
}

func MakePrivateEmbedded(value int) PrivateEmbedded {
	return PrivateEmbedded{privateBox: privateBox[int]{Value: value}}
}
