package main

// Documentation and string examples must not be parsed as native annotations.
// __asm__("not-a-symbol")
const example = `__asm__("not-a-symbol")`

var fixtureRoot = []int{5}

func nativeValue(value int) int __asm__("gccgo_fixture_native_value")

func NativeGeneric[T ~int](value T) T { return T(nativeValue(int(value))) }

func main() {
	if NativeGeneric(fixtureRoot[0]) != 22 || example != `__asm__("not-a-symbol")` {
		panic("native symbol annotation was lost during generic lowering")
	}
	println("NATIVE_ASM_PASS")
}
