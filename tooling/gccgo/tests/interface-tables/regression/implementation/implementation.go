package implementation

import (
	"regression/contract"
	"regression/marker"
)

type private struct{ marker.Base }

func (private) Value() int { return 42 }
var instance = new(private)
func New() contract.Value { return instance }
