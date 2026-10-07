package main

type count uint8

var calls int
var ptrs []*int

func bound() int { calls++; return 4 }

func assert(ok bool) {
	if !ok {
		panic("integer range regression")
	}
}

func main() {
	sum := 0
	for i := range bound() {
		sum += i
	}
	assert(sum == 6 && calls == 1)
	n := 4
	visits := 0
	for range n {
		n = 0
		visits++
	}
	assert(visits == 4)
	for range -3 {
		panic("negative bound")
	}
	for range 0 {
		panic("zero bound")
	}
	var k uint8 = 99
	for k = range 3 {
		assert(k < 3)
	}
	assert(k == 2)
	for k = range 0 {
	}
	assert(k == 2)
	for v := range count(3) {
		var typed count = v
		assert(typed < 3)
	}
	for i := range uint64(3) {
		var typed uint64 = i
		assert(typed < 3)
	}
	visits = 0
	for i := range 5 {
		if i == 1 {
			continue
		}
		if i == 4 {
			break
		}
		visits++
	}
	assert(visits == 3)
	outer := 0
Loop:
	for range 5 {
		for range 3 {
			outer++
			continue Loop
		}
	}
	assert(outer == 5)
	ptrs = make([]*int, 0)
	closures := make([]func() int, 0)
	for i := range 3 {
		ptrs = append(ptrs, &i)
		closures = append(closures, func() int { return i })
	}
	assert(*ptrs[0] == 0 && *ptrs[1] == 1 && *ptrs[2] == 2)
	assert(ptrs[0] != ptrs[1] && closures[0]() == 0 && closures[2]() == 2)
	var assigned int
	ptrs = nil
	for assigned = range 3 {
		ptrs = append(ptrs, &assigned)
	}
	assert(ptrs[0] == ptrs[1] && *ptrs[0] == 2)
	println("RANGEINT_PASS")
}
