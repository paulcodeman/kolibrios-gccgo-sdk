package otto

import (
	"fmt"
	"math"
	"unicode/utf16"
)

func (value Value) bool() bool {
	if value.kind == valueBoolean {
		return value.value.(bool)
	}
	if value.IsUndefined() {
		return false
	}
	if value.IsNull() {
		return false
	}
	switch value := value.value.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int8:
		return value != 0
	case int16:
		return value != 0
	case int32:
		return value != 0
	case int64:
		return value != 0
	case uint:
		return value != 0
	case uint8:
		return value != 0
	case uint16:
		return value != 0
	case uint32:
		return value != 0
	case uint64:
		return value != 0
	case float32:
		return 0 != value
	case float64:
		if math.IsNaN(value) || value == 0 {
			return false
		}
		return true
	case string:
		return 0 != len(value)
	case []uint16:
		return 0 != len(utf16.Decode(value))
	}
	if value.IsObject() {
		return true
	}
	panic(fmt.Errorf("toBoolean(%T)", value.value))
}
