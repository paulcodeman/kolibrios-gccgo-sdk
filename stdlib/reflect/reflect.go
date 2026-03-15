package reflect

import "unsafe"

type Kind uint8

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

// Ptr is the legacy name for Pointer.
const Ptr = Pointer

type ChanDir int

const (
	RecvDir ChanDir = 1 << iota
	SendDir
	BothDir = RecvDir | SendDir
)

type Type interface {
	Kind() Kind
	String() string
	Name() string
	PkgPath() string
	Size() uintptr
	Bits() int
	Align() int
	FieldAlign() int
	Len() int
	NumField() int
	Field(i int) StructField
	FieldByIndex(index []int) StructField
	FieldByName(name string) (StructField, bool)
	NumMethod() int
	Method(i int) Method
	MethodByName(name string) (Method, bool)
	NumIn() int
	In(i int) Type
	NumOut() int
	Out(i int) Type
	IsVariadic() bool
	Elem() Type
	Key() Type
	Comparable() bool
	Implements(u Type) bool
	AssignableTo(u Type) bool
	ConvertibleTo(u Type) bool
}

type rtype struct {
	kind Kind
}

func (t *rtype) Kind() Kind               { if t == nil { return Invalid }; return t.kind }
func (t *rtype) String() string           { return kindString(t.Kind()) }
func (t *rtype) Name() string             { return "" }
func (t *rtype) PkgPath() string          { return "" }
func (t *rtype) Size() uintptr            { return 0 }
func (t *rtype) Bits() int                { return 0 }
func (t *rtype) Align() int               { return 0 }
func (t *rtype) FieldAlign() int          { return 0 }
func (t *rtype) Len() int                 { return 0 }
func (t *rtype) NumField() int            { return 0 }
func (t *rtype) Field(i int) StructField  { return StructField{} }
func (t *rtype) FieldByIndex(index []int) StructField {
	return StructField{}
}
func (t *rtype) FieldByName(name string) (StructField, bool) { return StructField{}, false }
func (t *rtype) NumMethod() int                              { return 0 }
func (t *rtype) Method(i int) Method                         { return Method{} }
func (t *rtype) MethodByName(name string) (Method, bool)     { return Method{}, false }
func (t *rtype) NumIn() int                                  { return 0 }
func (t *rtype) In(i int) Type                               { return &rtype{kind: Invalid} }
func (t *rtype) NumOut() int                                 { return 0 }
func (t *rtype) Out(i int) Type                              { return &rtype{kind: Invalid} }
func (t *rtype) IsVariadic() bool                            { return false }
func (t *rtype) Elem() Type                                  { return &rtype{kind: Invalid} }
func (t *rtype) Key() Type                                   { return &rtype{kind: Invalid} }
func (t *rtype) Comparable() bool                            { return false }
func (t *rtype) Implements(u Type) bool                      { return false }
func (t *rtype) AssignableTo(u Type) bool                    { return false }
func (t *rtype) ConvertibleTo(u Type) bool                   { return false }

type StructTag string

func (tag StructTag) Get(key string) string { return "" }

func (tag StructTag) Lookup(key string) (string, bool) { return "", false }

type StructField struct {
	Name      string
	PkgPath   string
	Type      Type
	Tag       StructTag
	Offset    uintptr
	Index     []int
	Anonymous bool
}

func (f StructField) IsExported() bool {
	return f.PkgPath == ""
}

type Method struct {
	Name    string
	PkgPath string
	Type    Type
	Func    Value
	Index   int
}

type Value struct {
	typ  Type
	data interface{}
}

func ValueOf(i interface{}) Value {
	return Value{typ: TypeOf(i), data: i}
}

func TypeOf(i interface{}) Type {
	switch i.(type) {
	case nil:
		return nil
	case bool:
		return &rtype{kind: Bool}
	case int:
		return &rtype{kind: Int}
	case int8:
		return &rtype{kind: Int8}
	case int16:
		return &rtype{kind: Int16}
	case int32:
		return &rtype{kind: Int32}
	case int64:
		return &rtype{kind: Int64}
	case uint:
		return &rtype{kind: Uint}
	case uint8:
		return &rtype{kind: Uint8}
	case uint16:
		return &rtype{kind: Uint16}
	case uint32:
		return &rtype{kind: Uint32}
	case uint64:
		return &rtype{kind: Uint64}
	case uintptr:
		return &rtype{kind: Uintptr}
	case float32:
		return &rtype{kind: Float32}
	case float64:
		return &rtype{kind: Float64}
	case complex64:
		return &rtype{kind: Complex64}
	case complex128:
		return &rtype{kind: Complex128}
	case string:
		return &rtype{kind: String}
	default:
		return &rtype{kind: Invalid}
	}
}

func (v Value) Kind() Kind {
	if v.typ == nil {
		return Invalid
	}
	return v.typ.Kind()
}
func (v Value) Type() Type { return v.typ }
func (v Value) IsValid() bool {
	return v.typ != nil
}
func (v Value) IsNil() bool {
	return v.data == nil
}
func (v Value) CanInterface() bool { return true }
func (v Value) Interface() interface{} {
	return v.data
}
func (v Value) Bool() bool {
	if value, ok := v.data.(bool); ok {
		return value
	}
	return false
}
func (v Value) Int() int64 {
	switch value := v.data.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	}
	return 0
}
func (v Value) Uint() uint64 {
	switch value := v.data.(type) {
	case uint:
		return uint64(value)
	case uint8:
		return uint64(value)
	case uint16:
		return uint64(value)
	case uint32:
		return uint64(value)
	case uint64:
		return value
	case uintptr:
		return uint64(value)
	}
	return 0
}
func (v Value) Float() float64 {
	switch value := v.data.(type) {
	case float32:
		return float64(value)
	case float64:
		return value
	}
	return 0
}
func (v Value) String() string {
	if value, ok := v.data.(string); ok {
		return value
	}
	return ""
}
func (v Value) Bytes() []byte {
	switch value := v.data.(type) {
	case []byte:
		return value
	case string:
		return []byte(value)
	}
	return nil
}
func (v Value) CanSet() bool    { return false }
func (v Value) CanAddr() bool   { return false }
func (v Value) Addr() Value     { return Value{} }
func (v Value) Elem() Value     { return Value{} }
func (v Value) Len() int        { return 0 }
func (v Value) Cap() int        { return 0 }
func (v Value) Index(i int) Value { return Value{} }
func (v Value) Slice(i, j int) Value { return Value{} }
func (v Value) Slice3(i, j, k int) Value { return Value{} }
func (v Value) NumField() int   { return 0 }
func (v Value) Field(i int) Value { return Value{} }
func (v Value) FieldByIndex(index []int) Value { return Value{} }
func (v Value) FieldByName(name string) Value { return Value{} }
func (v Value) NumMethod() int  { return 0 }
func (v Value) Method(i int) Value { return Value{} }
func (v Value) MethodByName(name string) Value { return Value{} }
func (v Value) Pointer() uintptr { return 0 }
func (v Value) Convert(t Type) Value { return Value{} }
func (v Value) Call(in []Value) []Value { return nil }
func (v Value) CallSlice(in []Value) []Value { return nil }
func (v Value) MapKeys() []Value { return nil }
func (v Value) MapIndex(key Value) Value { return Value{} }
func (v Value) SetMapIndex(key, value Value) {}
func (v Value) Set(x Value) {}
func (v Value) SetInt(x int64) {}
func (v Value) SetUint(x uint64) {}
func (v Value) SetFloat(x float64) {}
func (v Value) SetString(x string) {}
func (v Value) SetBool(x bool) {}
func (v Value) SetBytes(x []byte) {}
func (v Value) SetLen(x int) {}
func (v Value) SetCap(x int) {}
func (v Value) OverflowInt(x int64) bool { return false }
func (v Value) OverflowUint(x uint64) bool { return false }
func (v Value) OverflowFloat(x float64) bool { return false }

func Zero(t Type) Value { return Value{} }
func New(t Type) Value { return Value{} }
func MakeSlice(t Type, len, cap int) Value { return Value{} }
func MakeMap(t Type) Value { return Value{} }
func MakeMapWithSize(t Type, n int) Value { return Value{} }
func MakeFunc(t Type, fn func(args []Value) []Value) Value { return Value{} }
func PtrTo(t Type) Type { return &rtype{kind: Ptr} }
func PointerTo(t Type) Type { return PtrTo(t) }
func SliceOf(t Type) Type { return &rtype{kind: Slice} }
func MapOf(key, elem Type) Type { return &rtype{kind: Map} }
func ArrayOf(len int, elem Type) Type { return &rtype{kind: Array} }
func ChanOf(dir ChanDir, t Type) Type { return &rtype{kind: Chan} }
func Indirect(v Value) Value { return v }

func DeepEqual(x, y interface{}) bool { return x == y }

func Copy(dst, src Value) int { return 0 }

func ValueOfPtr(ptr unsafe.Pointer) Value { return Value{} }

// MapIter is a stub iterator for map ranges.
type MapIter struct{}

func (it *MapIter) Next() bool { return false }
func (it *MapIter) Key() Value { return Value{} }
func (it *MapIter) Value() Value { return Value{} }

func (v Value) MapRange() *MapIter { return &MapIter{} }

func kindString(k Kind) string {
	switch k {
	case Invalid:
		return "invalid"
	case Bool:
		return "bool"
	case Int:
		return "int"
	case Int8:
		return "int8"
	case Int16:
		return "int16"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Uint:
		return "uint"
	case Uint8:
		return "uint8"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Uintptr:
		return "uintptr"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case Complex64:
		return "complex64"
	case Complex128:
		return "complex128"
	case Array:
		return "array"
	case Chan:
		return "chan"
	case Func:
		return "func"
	case Interface:
		return "interface"
	case Map:
		return "map"
	case Pointer:
		return "ptr"
	case Slice:
		return "slice"
	case String:
		return "string"
	case Struct:
		return "struct"
	case UnsafePointer:
		return "unsafe.Pointer"
	default:
		return "invalid"
	}
}
