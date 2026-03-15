// +build kolibrios,gccgo

//go:build kolibrios && gccgo

package kos

func allocCString(value string) *byte __asm__("runtime_alloc_cstring")
func freeCString(ptr *byte) __asm__("runtime_free_cstring")
func pointerValue(ptr *byte) uint32 __asm__("runtime_pointer_value")

func stringAddress(value string) (ptr *byte, addr uint32) {
	ptr = allocCString(value)
	if ptr == nil {
		return nil, 0
	}

	return ptr, pointerValue(ptr)
}

func byteSliceAddress(data []byte) uint32 {
	if len(data) == 0 {
		return 0
	}

	return pointerValue(&data[0])
}
