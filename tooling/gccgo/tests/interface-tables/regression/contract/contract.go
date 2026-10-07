package contract

import "regression/marker"

type Value interface {
	marker.Sealed
	Value() int
}

func Identity[T any](value T) T { return value }
