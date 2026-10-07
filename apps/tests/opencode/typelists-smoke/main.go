package main

import (
	"reflect"
	"unsafe"
	"kos"
)

func lookupType(name string) unsafe.Pointer __asm__("reflect.lookupType")

type record struct { Value string }
type interfaceHeader struct { methods, data unsafe.Pointer }
type descriptorPrefix struct {
	size, ptrdata uintptr
	hash uint32
	tflag, align, fieldAlign, kind uint8
	equal, gcdata unsafe.Pointer
	name *string
}

// Match libgo runtime._type.string, including quoted package metadata in
// composite descriptors. The public reflect.Type.String removes all quotes.
func descriptorName(pointer unsafe.Pointer) string {
	s := *(*descriptorPrefix)(pointer).name
	quoted, started := false, false
	start, end := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' { quoted = !quoted } else if !quoted {
			if !started { start, started = i, true }
			end = i
		}
	}
	return s[start:end+1]
}

func main() {
	console, ok := kos.OpenConsole("OpenCode runtime type registry test")
	if !ok { panic("console") }
	defer console.Close()
	kos.DebugString("OPENCODE_TYPELISTS_START")
	for _, typ := range []reflect.Type{
		reflect.TypeOf((*record)(nil)),
		reflect.TypeOf([]int(nil)), reflect.TypeOf(map[string]int(nil)),
		reflect.TypeOf([3]string{}), reflect.TypeOf((chan int)(nil)),
	} {
		pointer := (*interfaceHeader)(unsafe.Pointer(&typ)).data
		name := descriptorName(pointer)
		if lookupType(name) != pointer {
			panic("compiler descriptor registration or lookup")
		}
		if lookupType(name) != pointer { panic("repeated lookup changed identity") }
	}
	if lookupType("__unknown_type__") != nil || lookupType("") != nil {
		panic("unknown descriptor lookup")
	}
	kos.DebugString("OPENCODE_TYPELISTS_PASS")
	console.WriteString("OPENCODE_TYPELISTS_PASS\n")
}
