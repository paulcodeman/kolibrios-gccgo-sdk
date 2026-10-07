package main

type namedSlice []int
type namedMap map[int]string

var calls int
var strings = []string{"a", "b"}

func sliceOperand() []int {
	calls++
	return []int{1, 2}
}

func mapOperand() map[int]int {
	calls++
	return map[int]int{1: 2}
}

func assert(ok bool) {
	if !ok {
		panic("clear regression")
	}
}

func main() {
	var nilMap map[int]int
	var nilSlice []int
	clear(nilMap)
	clear(nilSlice)
	backing := namedSlice{1, 2, 3, 4}
	clear(backing[:2])
	assert(backing[0] == 0 && backing[1] == 0 && backing[2] == 3 && backing[3] == 4)
	clear(backing[:0])
	assert(backing[2] == 3)
	pointers := []*int{&backing[2], &backing[3]}
	clear(pointers)
	assert(pointers[0] == nil && pointers[1] == nil)
	clear(strings)
	assert(strings[0] == "" && strings[1] == "")
	items := []struct {
		N int
		P *int
	}{{N: 7, P: &backing[2]}}
	clear(items)
	assert(items[0].N == 0 && items[0].P == nil)
	zeroSized := make([]struct{}, 3)
	clear(zeroSized)
	assert(len(zeroSized) == 3)
	m := namedMap{1: "a", 2: "b"}
	alias := m
	clear(m)
	assert(len(m) == 0 && len(alias) == 0)
	m[3] = "c"
	assert(m[3] == "c")
	// NaN keys cannot be removed by a for-range/delete implementation.
	zero := 0.0
	nan := zero / zero
	nanMap := map[float64]int{nan: 1, nan: 2}
	assert(len(nanMap) == 2)
	clear(nanMap)
	assert(len(nanMap) == 0)
	clear(sliceOperand())
	clear(mapOperand())
	assert(calls == 2)
	deferred := []int{9}
	func() { defer clear(deferred) }()
	assert(deferred[0] == 0)
	async := []int{9}
	done := make(chan struct{})
	go func() { clear(async); close(done) }()
	<-done
	assert(async[0] == 0)
	println("CLEAR_PASS")
}
