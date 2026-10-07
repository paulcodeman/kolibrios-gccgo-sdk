package main

import "unsafe"

type number int16
type text string
type real float32

const small = min(10, 1000)
const large = max(uint64(1<<64-1), uint64(1<<64-2))
const first = min("z", "a", "b")
const floating = min(1, 2.0)
const character = min(1, 'a')
const typed = min(int(1), 2.0)

var _ int8 = small
var _ uint64 = large
var _ [min(5, 2)]byte

func fromBits(bits uint64) float64 { return *(*float64)(unsafe.Pointer(&bits)) }
func signbit(x float64) bool       { return *(*uint64)(unsafe.Pointer(&x))>>63 != 0 }
func isnan(x float64) bool         { return x != x }

func check(ok bool) {
	if !ok {
		panic("min/max regression")
	}
}

var calls []int

func arg(x int) int {
	calls = append(calls, x)
	return x
}

func pair() (int, int) { return 9, 4 }
func escaped() string {
	b := []byte{'a', 'b', 'c'}
	return min(string(b), "zzz")
}

func stringPointer(s string) *byte {
	return (*struct {
		data *byte
		len  int
	})(unsafe.Pointer(&s)).data
}

func main() {
	check(min(arg(9), arg(2), arg(5)) == 2)
	check(len(calls) == 3 && calls[0] == 9 && calls[1] == 2 && calls[2] == 5)
	check(max(pair()) == 9 && min(pair()) == 4)
	x, y := number(-100), number(200)
	var n number = max(x, y, 15)
	check(n == y && min(x) == x && min(x, y) == x)
	a, b := text("a\xff"), text("a\x00")
	var s text = min(a, b)
	check(s == b && max(a, b) == a && escaped() == "abc")
	left, right := escaped(), escaped()
	check(stringPointer(min(left, right)) == stringPointer(left))
	check(stringPointer(max(left, right)) == stringPointer(left))
	var f interface{} = floating
	var r interface{} = character
	_, floatOK := f.(float64)
	_, runeOK := r.(int32)
	check(floatOK && runeOK && small == 10 && first == "a" && large == 1<<64-1)
	check(min(1<<200, 1<<199) == 1<<199)
	check(max(1<<200, 1<<199) == 1<<200)
	check(typed == 1)
	var narrow int8 = min(1, 1000)
	check(narrow == 1)
	neg, pos := fromBits(1<<63), float64(0)
	nan := fromBits(0x7ff8000000000001)
	inf := fromBits(0x7ff0000000000000)
	for _, zeroes := range [][2]float64{{neg, pos}, {pos, neg}, {neg, neg}, {pos, pos}} {
		gotMin := min(zeroes[0], zeroes[1])
		gotMax := max(zeroes[0], zeroes[1])
		check(signbit(gotMin) == (signbit(zeroes[0]) || signbit(zeroes[1])))
		check(signbit(gotMax) == (signbit(zeroes[0]) && signbit(zeroes[1])))
	}
	check(isnan(min(nan, 1)) && isnan(min(1, nan)))
	check(isnan(max(nan, 1)) && isnan(max(1, nan)))
	check(isnan(min(1, nan, -inf)) && isnan(max(1, nan, inf)))
	check(min(-inf, inf) == -inf && max(-inf, inf) == inf)
	fneg, fpos := real(neg), real(pos)
	var fm real = min(fpos, fneg)
	check(signbit(float64(fm)) && !signbit(float64(max(fneg, fpos))))
	check(isnan(float64(min(real(nan), real(1)))))
	count := 0
	value := func(x float64) float64 { count++; return x }
	check(isnan(min(value(nan), value(1), value(2))) && count == 3)
	println("PASS: min/max constants, types, evaluation, NaN and signed zero")
}
