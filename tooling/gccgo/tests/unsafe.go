package main

import (
    "runtime"
    "unsafe"
)

type text string
type integers []int

var escapedString string
var escapedData *byte

func storage() *byte {
    s := string([]byte{'h', 'e', 'a', 'p'})
    return unsafe.StringData(s)
}

func expectPanic(f func()) {
    defer func() {
        value := recover()
        if value == nil { panic("missing unsafe.String panic") }
        if _, ok := value.(runtime.Error); !ok { panic("unsafe panic is not runtime.Error") }
    }()
    f()
}

func main() {
    s := text("hello")
    p := unsafe.StringData(string(s))
    if p == nil || *p != 'h' { panic("StringData") }
    data := integers{7, 9}
    if unsafe.SliceData(data) != &data[0] || unsafe.SliceData(data[:0]) != &data[0] {
        panic("SliceData backing storage")
    }
    var nilSlice []int
    if unsafe.SliceData(nilSlice) != nil { panic("SliceData nil") }
    bytes := []byte{'a', 'b', 'c'}
    escapedString = unsafe.String(&bytes[0], uint64(len(bytes)))
    if escapedString != "abc" || unsafe.StringData(escapedString) != &bytes[0] {
        panic("String copies its backing memory")
    }
    escapedData = storage()
    if *escapedData != 'h' { panic("StringData escape") }
    order := 0
    pointer := func() *byte { if order != 0 { panic("pointer evaluation") }; order++; return &bytes[0] }
    length := func() uint8 { if order != 1 { panic("length evaluation") }; order++; return 2 }
    if unsafe.String(pointer(), length()) != "ab" || order != 2 { panic("evaluation count") }
    var zero *byte
    if unsafe.String(zero, 0) != "" { panic("nil empty String") }
    if unsafe.String(nil, 0) != "" { panic("untyped nil String") }
    negative := -1
    expectPanic(func() { _ = unsafe.String(p, negative) })
    expectPanic(func() { _ = unsafe.String(zero, 1) })
    wide := ^uint64(0)
    expectPanic(func() { _ = unsafe.String(p, wide) })
    edge := (*byte)(unsafe.Pointer(^uintptr(0)))
    expectPanic(func() { _ = unsafe.String(edge, 2) })
    println("PASS: unsafe string and slice backing storage, evaluation and checks")
}
