package marker

type Sealed interface{ seal() }
type Base struct{}

func (Base) seal() {}
