package otto

import (
	"unsafe"

	ottojs "github.com/robertkrimen/otto"
)

const evalError = uint32(0xFFFFFFFF)

var vm *ottojs.Otto

func ensureVM() *ottojs.Otto {
	if vm == nil {
		vm = ottojs.New()
	}
	return vm
}

func bytesFromPtr(ptr uint32, length uint32) []byte {
	if ptr == 0 || length == 0 {
		return nil
	}
	const max = 1 << 30
	n := int(length)
	if n > max {
		n = max
	}
	return (*[max]byte)(unsafe.Pointer(uintptr(ptr)))[:n:n]
}

func writeCString(ptr uint32, length uint32, value string) {
	if ptr == 0 || length == 0 {
		return
	}
	buf := bytesFromPtr(ptr, length)
	if len(buf) == 0 {
		return
	}
	max := len(buf) - 1
	if max < 0 {
		return
	}
	n := len(value)
	if n > max {
		n = max
	}
	if n > 0 {
		copy(buf, value[:n])
	}
	buf[n] = 0
}

func panicString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return "panic"
	}
}

// JsEval evaluates JavaScript source and writes the result into the output buffer.
// It returns the result length, or 0xFFFFFFFF on error.
func JsEval(inputPtr, inputLen, outputPtr, outputLen uint32) (ret uint32) {
	var result string
	ok := true

	defer func() {
		if r := recover(); r != nil {
			ok = false
			result = panicString(r)
		}
		writeCString(outputPtr, outputLen, result)
		if ok {
			ret = uint32(len(result))
		} else {
			ret = evalError
		}
	}()

	if inputLen == 0 {
		result = ""
		return
	}
	if inputPtr == 0 {
		ok = false
		result = "invalid input buffer"
		return
	}

	input := bytesFromPtr(inputPtr, inputLen)
	if input == nil {
		ok = false
		result = "invalid input buffer"
		return
	}

	value, err := ensureVM().Run(string(input))
	if err != nil {
		ok = false
		result = err.Error()
		return
	}
	if value.IsUndefined() {
		result = "undefined"
		return
	}
	result = value.String()
	return
}
