package otto

import "testing"

func isCompactIntegerValue(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return true
	default:
		return false
	}
}

func TestPostfixDecrementKeepsCompactInteger(t *testing.T) {
	vm := New()

	if _, err := vm.Run(`i = 1000; while (i--) {}`); err != nil {
		t.Fatal(err)
	}

	value, err := vm.Get("i")
	if err != nil {
		t.Fatal(err)
	}
	if !isCompactIntegerValue(value.value) {
		t.Fatalf("expected compact integer representation, got %T", value.value)
	}
}
