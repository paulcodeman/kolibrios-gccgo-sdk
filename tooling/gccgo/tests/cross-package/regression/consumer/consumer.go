package consumer

import "regression/owner"

type Holder struct{ Box owner.Box[int] }

func New() Holder    { return Holder{Box: owner.Make(7)} }
func Interface() any { return owner.Make(11) }

func Nested() owner.Nested[int] { return owner.NewNested(37) }

type Embedded struct {
	*owner.Box[int] `json:"embedded"`
}

type EmbeddedValue struct {
	owner.Box[int]
}

func NewEmbedded() Embedded {
	box := owner.Make(19)
	return Embedded{Box: &box}
}

func NewEmbeddedValue() EmbeddedValue {
	return EmbeddedValue{Box: owner.Make(23)}
}
