package reflect

import (
	"internal/unsafeheader"
	"strconv"
	"unsafe"
)

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

type tflag uint8

const (
	kindDirectIface = 1 << 5
	kindMask        = (1 << 5) - 1
)

type rtype struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata     *byte
	string     *string
	*uncommonType
	ptrToThis *rtype
}

type method struct {
	name    *string
	pkgPath *string
	mtyp    *rtype
	typ     *rtype
	tfn     unsafe.Pointer
}

type uncommonType struct {
	name    *string
	pkgPath *string
	methods []method
}

type arrayType struct {
	rtype
	elem  *rtype
	slice *rtype
	len   uintptr
}

type chanType struct {
	rtype
	elem *rtype
	dir  uintptr
}

type funcType struct {
	rtype
	dotdotdot bool
	in        []*rtype
	out       []*rtype
}

type imethod struct {
	name    *string
	pkgPath *string
	typ     *rtype
}

type interfaceType struct {
	rtype
	methods []imethod
}

type mapType struct {
	rtype
	key        *rtype
	elem       *rtype
	bucket     *rtype
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8
	valuesize  uint8
	bucketsize uint16
	flags      uint32
}

type ptrType struct {
	rtype
	elem *rtype
}

type sliceType struct {
	rtype
	elem *rtype
}

type rawStructField struct {
	name        *string
	pkgPath     *string
	typ         *rtype
	tag         *string
	offsetEmbed uintptr
}

func (f *rawStructField) offset() uintptr {
	return f.offsetEmbed >> 1
}

func (f *rawStructField) embedded() bool {
	return f.offsetEmbed&1 != 0
}

type structType struct {
	rtype
	fields []rawStructField
}

type StructTag string

func (tag StructTag) Get(key string) string {
	value, _ := tag.Lookup(key)
	return value
}

func (tag StructTag) Lookup(key string) (string, bool) {
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

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

type flag uintptr

const (
	flagKindWidth      = 5
	flagKindMask  flag = 1<<flagKindWidth - 1
	flagStickyRO  flag = 1 << 5
	flagEmbedRO   flag = 1 << 6
	flagIndir     flag = 1 << 7
	flagAddr      flag = 1 << 8
	flagRO             = flagStickyRO | flagEmbedRO
)

type Value struct {
	typ  *rtype
	ptr  unsafe.Pointer
	flag flag
}

type emptyInterface struct {
	typ  *rtype
	word unsafe.Pointer
}

type ValueError struct {
	Method string
	Kind   Kind
}

func (e *ValueError) Error() string {
	if e.Kind == Invalid {
		return "reflect: call of " + e.Method + " on zero Value"
	}
	return "reflect: call of " + e.Method + " on " + e.Kind.String() + " Value"
}

func toType(t *rtype) Type {
	if t == nil {
		return nil
	}
	return t
}

func ifaceIndir(t *rtype) bool {
	return t.kind&kindDirectIface == 0
}

func add(ptr unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) + offset)
}

func arrayAt(base unsafe.Pointer, index int, elemSize uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(base) + uintptr(index)*elemSize)
}

func (f flag) kind() Kind {
	return Kind(f & flagKindMask)
}

func (f flag) ro() flag {
	if f&flagRO != 0 {
		return flagStickyRO
	}
	return 0
}

func (f flag) mustBe(expected Kind, method string) {
	if f.kind() != expected {
		panic(&ValueError{Method: method, Kind: f.kind()})
	}
}

func (f flag) mustBeAssignable(method string) {
	if f == 0 {
		panic(&ValueError{Method: method, Kind: Invalid})
	}
	if f&flagRO != 0 {
		panic("reflect: " + method + " using value obtained using unexported field")
	}
	if f&flagAddr == 0 {
		panic("reflect: " + method + " using unaddressable value")
	}
}

func (k Kind) String() string {
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

func (t *rtype) Kind() Kind {
	if t == nil {
		return Invalid
	}
	return Kind(t.kind & kindMask)
}

func (t *rtype) String() string {
	if t == nil || t.string == nil {
		return ""
	}
	s := *t.string
	if s == "" {
		return ""
	}
	var quoted bool
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			quoted = !quoted
			continue
		}
		if !quoted {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func (t *rtype) Name() string {
	if t == nil || t.uncommonType == nil || t.uncommonType.name == nil {
		return ""
	}
	return *t.uncommonType.name
}

func (t *rtype) PkgPath() string {
	if t == nil || t.uncommonType == nil || t.uncommonType.pkgPath == nil {
		return ""
	}
	return *t.uncommonType.pkgPath
}

func (t *rtype) Size() uintptr {
	if t == nil {
		return 0
	}
	return t.size
}

func (t *rtype) Bits() int {
	if t == nil {
		panic("reflect: Bits of nil Type")
	}
	k := t.Kind()
	if k < Int || k > Complex128 {
		panic("reflect: Bits of non-arithmetic Type " + t.String())
	}
	return int(t.size) * 8
}

func (t *rtype) Align() int {
	if t == nil {
		return 0
	}
	return int(t.align)
}

func (t *rtype) FieldAlign() int {
	if t == nil {
		return 0
	}
	return int(t.fieldAlign)
}

func (t *rtype) Len() int {
	if t.Kind() != Array {
		panic("reflect: Len of non-array type " + t.String())
	}
	return int((*arrayType)(unsafe.Pointer(t)).len)
}

func (t *rtype) NumField() int {
	if t.Kind() != Struct {
		panic("reflect: NumField of non-struct type " + t.String())
	}
	return len((*structType)(unsafe.Pointer(t)).fields)
}

func (t *rtype) Field(i int) StructField {
	if t.Kind() != Struct {
		panic("reflect: Field of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	if i < 0 || i >= len(tt.fields) {
		panic("reflect: Field index out of bounds")
	}
	raw := &tt.fields[i]
	field := StructField{
		Type:      toType(raw.typ),
		Offset:    raw.offset(),
		Index:     []int{i},
		Anonymous: raw.embedded(),
	}
	if raw.name != nil {
		field.Name = *raw.name
	}
	if raw.pkgPath != nil {
		field.PkgPath = *raw.pkgPath
	}
	if raw.tag != nil {
		field.Tag = StructTag(*raw.tag)
	}
	return field
}

func (t *rtype) FieldByIndex(index []int) StructField {
	field := t.Field(index[0])
	for _, i := range index[1:] {
		ft := field.Type.(*rtype)
		if ft.Kind() == Pointer {
			ft = (*ptrType)(unsafe.Pointer(ft)).elem
		}
		field = ft.Field(i)
	}
	return field
}

func (t *rtype) FieldByName(name string) (StructField, bool) {
	if t.Kind() != Struct {
		panic("reflect: FieldByName of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	for i := range tt.fields {
		if tt.fields[i].name != nil && *tt.fields[i].name == name {
			return tt.Field(i), true
		}
	}
	return StructField{}, false
}

func (t *rtype) NumMethod() int                          { return 0 }
func (t *rtype) Method(i int) Method                     { return Method{} }
func (t *rtype) MethodByName(name string) (Method, bool) { return Method{}, false }
func (t *rtype) NumIn() int                              { return 0 }
func (t *rtype) In(i int) Type                           { return nil }
func (t *rtype) NumOut() int                             { return 0 }
func (t *rtype) Out(i int) Type                          { return nil }
func (t *rtype) IsVariadic() bool                        { return false }
func (t *rtype) Key() Type                               { return nil }
func (t *rtype) Implements(u Type) bool                  { return false }
func (t *rtype) AssignableTo(u Type) bool                { return t == u }
func (t *rtype) ConvertibleTo(u Type) bool               { return t == u }
func (t *rtype) Comparable() bool                        { return t != nil && t.equal != nil }

func (t *rtype) Elem() Type {
	switch t.Kind() {
	case Array:
		return toType((*arrayType)(unsafe.Pointer(t)).elem)
	case Chan:
		return toType((*chanType)(unsafe.Pointer(t)).elem)
	case Map:
		return toType((*mapType)(unsafe.Pointer(t)).elem)
	case Pointer:
		return toType((*ptrType)(unsafe.Pointer(t)).elem)
	case Slice:
		return toType((*sliceType)(unsafe.Pointer(t)).elem)
	default:
		panic("reflect: Elem of invalid type " + t.String())
	}
}

func TypeOf(i interface{}) Type {
	e := (*emptyInterface)(unsafe.Pointer(&i))
	return toType(e.typ)
}

func ValueOf(i interface{}) Value {
	if i == nil {
		return Value{}
	}
	e := (*emptyInterface)(unsafe.Pointer(&i))
	if e.typ == nil {
		return Value{}
	}
	fl := flag(e.typ.Kind())
	if ifaceIndir(e.typ) {
		fl |= flagIndir
	}
	return Value{typ: e.typ, ptr: e.word, flag: fl}
}

func (v Value) kind() Kind {
	if v.flag == 0 {
		return Invalid
	}
	return v.flag.kind()
}

func (v Value) Kind() Kind { return v.kind() }

func (v Value) Type() Type {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Type", Invalid})
	}
	return toType(v.typ)
}

func (v Value) IsValid() bool {
	return v.flag != 0
}

func (v Value) IsNil() bool {
	switch v.kind() {
	case Pointer, Map, Slice, Chan, Func, UnsafePointer:
		if v.flag&flagIndir != 0 {
			return *(*unsafe.Pointer)(v.ptr) == nil
		}
		return v.ptr == nil
	case Interface:
		if v.ptr == nil {
			return true
		}
		e := (*emptyInterface)(v.ptr)
		return e.typ == nil
	default:
		return false
	}
}

func (v Value) CanInterface() bool {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.CanInterface", Invalid})
	}
	return v.flag&flagRO == 0
}

func packEface(v Value) interface{} {
	var i interface{}
	e := (*emptyInterface)(unsafe.Pointer(&i))
	t := v.typ
	switch {
	case ifaceIndir(t):
		if v.flag&flagIndir == 0 {
			panic("reflect: bad indir")
		}
		e.word = v.ptr
	case v.flag&flagIndir != 0:
		e.word = *(*unsafe.Pointer)(v.ptr)
	default:
		e.word = v.ptr
	}
	e.typ = t
	return i
}

func (v Value) Interface() interface{} {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", Invalid})
	}
	if v.flag&flagRO != 0 {
		panic("reflect: reflect.Value.Interface using value obtained using unexported field")
	}
	return packEface(v)
}

func (v Value) Bool() bool {
	v.flag.mustBe(Bool, "reflect.Value.Bool")
	return *(*bool)(v.ptr)
}

func (v Value) Int() int64 {
	switch v.kind() {
	case Int:
		return int64(*(*int)(v.ptr))
	case Int8:
		return int64(*(*int8)(v.ptr))
	case Int16:
		return int64(*(*int16)(v.ptr))
	case Int32:
		return int64(*(*int32)(v.ptr))
	case Int64:
		return *(*int64)(v.ptr)
	default:
		panic(&ValueError{"reflect.Value.Int", v.kind()})
	}
}

func (v Value) Uint() uint64 {
	switch v.kind() {
	case Uint:
		return uint64(*(*uint)(v.ptr))
	case Uint8:
		return uint64(*(*uint8)(v.ptr))
	case Uint16:
		return uint64(*(*uint16)(v.ptr))
	case Uint32:
		return uint64(*(*uint32)(v.ptr))
	case Uint64:
		return *(*uint64)(v.ptr)
	case Uintptr:
		return uint64(*(*uintptr)(v.ptr))
	default:
		panic(&ValueError{"reflect.Value.Uint", v.kind()})
	}
}

func (v Value) Float() float64 {
	switch v.kind() {
	case Float32:
		return float64(*(*float32)(v.ptr))
	case Float64:
		return *(*float64)(v.ptr)
	default:
		return 0
	}
}

func (v Value) String() string {
	if v.kind() == Invalid {
		return "<invalid Value>"
	}
	if v.kind() == String {
		return *(*string)(v.ptr)
	}
	return "<" + v.Type().String() + " Value>"
}

func (v Value) Bytes() []byte {
	v.flag.mustBe(Slice, "reflect.Value.Bytes")
	if v.typ.Elem().Kind() != Uint8 {
		panic("reflect.Value.Bytes of non-byte slice")
	}
	return *(*[]byte)(v.ptr)
}

func (v Value) CanSet() bool {
	return v.flag&(flagAddr|flagRO) == flagAddr
}

func (v Value) CanAddr() bool {
	return v.flag&flagAddr != 0
}

func (v Value) Addr() Value {
	if v.flag&flagAddr == 0 {
		panic("reflect.Value.Addr of unaddressable value")
	}
	return Value{typ: ptrTo(v.typ), ptr: v.ptr, flag: v.flag&flagRO | flag(Pointer)}
}

func (v Value) Elem() Value {
	switch v.kind() {
	case Pointer:
		ptr := v.ptr
		if v.flag&flagIndir != 0 {
			ptr = *(*unsafe.Pointer)(ptr)
		}
		if ptr == nil {
			return Value{}
		}
		typ := (*ptrType)(unsafe.Pointer(v.typ)).elem
		fl := v.flag&flagRO | flagIndir | flagAddr | flag(typ.Kind())
		return Value{typ: typ, ptr: ptr, flag: fl}
	case Interface:
		if v.ptr == nil {
			return Value{}
		}
		return ValueOf(*(*interface{})(v.ptr))
	default:
		panic(&ValueError{"reflect.Value.Elem", v.kind()})
	}
}

func (v Value) Len() int {
	switch v.kind() {
	case Array:
		return int((*arrayType)(unsafe.Pointer(v.typ)).len)
	case Slice:
		return (*unsafeheader.Slice)(v.ptr).Len
	case String:
		return (*unsafeheader.String)(v.ptr).Len
	default:
		panic(&ValueError{"reflect.Value.Len", v.kind()})
	}
}

func (v Value) Cap() int {
	switch v.kind() {
	case Slice:
		return (*unsafeheader.Slice)(v.ptr).Cap
	default:
		return 0
	}
}

func (v Value) Index(i int) Value {
	switch v.kind() {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		if uint(i) >= uint(tt.len) {
			panic("reflect: array index out of range")
		}
		typ := tt.elem
		val := add(v.ptr, uintptr(i)*typ.size)
		fl := v.flag&(flagIndir|flagAddr) | v.flag.ro() | flag(typ.Kind())
		return Value{typ: typ, ptr: val, flag: fl}
	case Slice:
		s := (*unsafeheader.Slice)(v.ptr)
		if uint(i) >= uint(s.Len) {
			panic("reflect: slice index out of range")
		}
		typ := (*sliceType)(unsafe.Pointer(v.typ)).elem
		val := arrayAt(s.Data, i, typ.size)
		fl := flagAddr | flagIndir | v.flag.ro() | flag(typ.Kind())
		return Value{typ: typ, ptr: val, flag: fl}
	default:
		panic(&ValueError{"reflect.Value.Index", v.kind()})
	}
}

func (v Value) Slice(i, j int) Value     { return Value{} }
func (v Value) Slice3(i, j, k int) Value { return Value{} }

func (v Value) NumField() int {
	v.flag.mustBe(Struct, "reflect.Value.NumField")
	return len((*structType)(unsafe.Pointer(v.typ)).fields)
}

func (v Value) Field(i int) Value {
	v.flag.mustBe(Struct, "reflect.Value.Field")
	tt := (*structType)(unsafe.Pointer(v.typ))
	if uint(i) >= uint(len(tt.fields)) {
		panic("reflect: Field index out of range")
	}
	field := &tt.fields[i]
	typ := field.typ
	fl := v.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind())
	if field.pkgPath != nil {
		if field.embedded() {
			fl |= flagEmbedRO
		} else {
			fl |= flagStickyRO
		}
	}
	ptr := add(v.ptr, field.offset())
	return Value{typ: typ, ptr: ptr, flag: fl}
}

func (v Value) FieldByIndex(index []int) Value {
	if len(index) == 1 {
		return v.Field(index[0])
	}
	v.flag.mustBe(Struct, "reflect.Value.FieldByIndex")
	for i, x := range index {
		if i > 0 && v.Kind() == Pointer && v.typ.Elem().Kind() == Struct {
			if v.IsNil() {
				panic("reflect: indirection through nil pointer to embedded struct")
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v
}

func (v Value) FieldByName(name string) Value {
	v.flag.mustBe(Struct, "reflect.Value.FieldByName")
	if f, ok := v.typ.FieldByName(name); ok {
		return v.FieldByIndex(f.Index)
	}
	return Value{}
}

func (v Value) NumMethod() int                 { return 0 }
func (v Value) Method(i int) Value             { return Value{} }
func (v Value) MethodByName(name string) Value { return Value{} }
func (v Value) Pointer() uintptr               { return 0 }
func (v Value) Convert(t Type) Value           { return Value{} }
func (v Value) Call(in []Value) []Value        { return nil }
func (v Value) CallSlice(in []Value) []Value   { return nil }
func (v Value) MapKeys() []Value               { return nil }
func (v Value) MapIndex(key Value) Value       { return Value{} }
func (v Value) SetMapIndex(key, value Value)   {}

func rawValuePointer(v Value) unsafe.Pointer {
	if v.flag&flagIndir != 0 && !ifaceIndir(v.typ) {
		return *(*unsafe.Pointer)(v.ptr)
	}
	return v.ptr
}

func (v Value) Set(x Value) {
	v.flag.mustBeAssignable("reflect.Set")
	if !x.IsValid() {
		panic("reflect: zero Value passed to Set")
	}
	if v.typ != x.typ {
		panic("reflect: Set of incompatible value")
	}
	switch v.kind() {
	case Bool:
		*(*bool)(v.ptr) = *(*bool)(x.ptr)
	case Int:
		*(*int)(v.ptr) = *(*int)(x.ptr)
	case Int8:
		*(*int8)(v.ptr) = *(*int8)(x.ptr)
	case Int16:
		*(*int16)(v.ptr) = *(*int16)(x.ptr)
	case Int32:
		*(*int32)(v.ptr) = *(*int32)(x.ptr)
	case Int64:
		*(*int64)(v.ptr) = *(*int64)(x.ptr)
	case Uint:
		*(*uint)(v.ptr) = *(*uint)(x.ptr)
	case Uint8:
		*(*uint8)(v.ptr) = *(*uint8)(x.ptr)
	case Uint16:
		*(*uint16)(v.ptr) = *(*uint16)(x.ptr)
	case Uint32:
		*(*uint32)(v.ptr) = *(*uint32)(x.ptr)
	case Uint64:
		*(*uint64)(v.ptr) = *(*uint64)(x.ptr)
	case Uintptr:
		*(*uintptr)(v.ptr) = *(*uintptr)(x.ptr)
	case String:
		*(*string)(v.ptr) = *(*string)(x.ptr)
	case Slice:
		switch v.typ.Elem().Kind() {
		case Uint8:
			*(*[]byte)(v.ptr) = *(*[]byte)(x.ptr)
		case String:
			*(*[]string)(v.ptr) = *(*[]string)(x.ptr)
		default:
			panic("reflect: Set unsupported slice type")
		}
	case Pointer, UnsafePointer:
		*(*unsafe.Pointer)(v.ptr) = rawValuePointer(x)
	default:
		panic("reflect: Set unsupported kind " + v.kind().String())
	}
}

func (v Value) SetInt(x int64) {
	v.flag.mustBeAssignable("reflect.Value.SetInt")
	switch v.kind() {
	case Int:
		*(*int)(v.ptr) = int(x)
	case Int8:
		*(*int8)(v.ptr) = int8(x)
	case Int16:
		*(*int16)(v.ptr) = int16(x)
	case Int32:
		*(*int32)(v.ptr) = int32(x)
	case Int64:
		*(*int64)(v.ptr) = x
	default:
		panic(&ValueError{"reflect.Value.SetInt", v.kind()})
	}
}

func (v Value) SetUint(x uint64) {
	v.flag.mustBeAssignable("reflect.Value.SetUint")
	switch v.kind() {
	case Uint:
		*(*uint)(v.ptr) = uint(x)
	case Uint8:
		*(*uint8)(v.ptr) = uint8(x)
	case Uint16:
		*(*uint16)(v.ptr) = uint16(x)
	case Uint32:
		*(*uint32)(v.ptr) = uint32(x)
	case Uint64:
		*(*uint64)(v.ptr) = x
	case Uintptr:
		*(*uintptr)(v.ptr) = uintptr(x)
	default:
		panic(&ValueError{"reflect.Value.SetUint", v.kind()})
	}
}

func (v Value) SetFloat(x float64) {
	v.flag.mustBeAssignable("reflect.Value.SetFloat")
	switch v.kind() {
	case Float32:
		*(*float32)(v.ptr) = float32(x)
	case Float64:
		*(*float64)(v.ptr) = x
	default:
		panic(&ValueError{"reflect.Value.SetFloat", v.kind()})
	}
}

func (v Value) SetString(x string) {
	v.flag.mustBeAssignable("reflect.Value.SetString")
	v.flag.mustBe(String, "reflect.Value.SetString")
	*(*string)(v.ptr) = x
}

func (v Value) SetBool(x bool) {
	v.flag.mustBeAssignable("reflect.Value.SetBool")
	v.flag.mustBe(Bool, "reflect.Value.SetBool")
	*(*bool)(v.ptr) = x
}

func (v Value) SetBytes(x []byte) {
	v.flag.mustBeAssignable("reflect.Value.SetBytes")
	v.flag.mustBe(Slice, "reflect.Value.SetBytes")
	if v.typ.Elem().Kind() != Uint8 {
		panic("reflect.Value.SetBytes of non-byte slice")
	}
	*(*[]byte)(v.ptr) = x
}

func (v Value) SetLen(x int) {}
func (v Value) SetCap(x int) {}

func (v Value) OverflowInt(x int64) bool {
	switch v.kind() {
	case Int, Int8, Int16, Int32, Int64:
		bitSize := v.typ.size * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	default:
		return false
	}
}

func (v Value) OverflowUint(x uint64) bool {
	switch v.kind() {
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		bitSize := v.typ.size * 8
		trunc := (x << (64 - bitSize)) >> (64 - bitSize)
		return x != trunc
	default:
		return false
	}
}

func (v Value) OverflowFloat(x float64) bool { return false }

func Zero(t Type) Value {
	if t == nil {
		panic("reflect: Zero(nil)")
	}
	rt := t.(*rtype)
	fl := flag(rt.Kind())
	if ifaceIndir(rt) {
		return Value{typ: rt, ptr: runtimeNewObject(rt), flag: fl | flagIndir}
	}
	return Value{typ: rt, flag: fl}
}

func New(t Type) Value {
	if t == nil {
		panic("reflect: New(nil)")
	}
	rt := t.(*rtype)
	return Value{typ: ptrTo(rt), ptr: runtimeNewObject(rt), flag: flag(Pointer)}
}

func MakeSlice(t Type, len, cap int) Value {
	if t == nil {
		panic("reflect: MakeSlice(nil)")
	}
	rt := t.(*rtype)
	if rt.Kind() != Slice {
		panic("reflect: MakeSlice of non-slice type " + rt.String())
	}
	st := (*sliceType)(unsafe.Pointer(rt))
	headerPtr := runtimeNewObject(rt)
	header := (*unsafeheader.Slice)(headerPtr)
	header.Data = runtimeMakeSlice(st.elem, len, cap)
	header.Len = len
	header.Cap = cap
	return Value{typ: rt, ptr: headerPtr, flag: flag(Slice) | flagIndir}
}
func MakeMap(t Type) Value                                 { return Value{} }
func MakeMapWithSize(t Type, n int) Value                  { return Value{} }
func MakeFunc(t Type, fn func(args []Value) []Value) Value { return Value{} }
func PtrTo(t Type) Type {
	if t == nil {
		panic("reflect: PtrTo(nil)")
	}
	return ptrTo(t.(*rtype))
}
func PointerTo(t Type) Type                                { return PtrTo(t) }
func SliceOf(t Type) Type                                  { return &rtype{kind: uint8(Slice)} }
func MapOf(key, elem Type) Type                            { return &rtype{kind: uint8(Map)} }
func ArrayOf(len int, elem Type) Type                      { return &rtype{kind: uint8(Array)} }
func ChanOf(dir ChanDir, t Type) Type                      { return &rtype{kind: uint8(Chan)} }

func Indirect(v Value) Value {
	if v.Kind() != Pointer {
		return v
	}
	return v.Elem()
}

func DeepEqual(x, y interface{}) bool {
	vx := ValueOf(x)
	vy := ValueOf(y)
	if !vx.IsValid() || !vy.IsValid() {
		return !vx.IsValid() && !vy.IsValid()
	}
	if vx.typ != vy.typ {
		return false
	}
	return deepValueEqual(vx, vy)
}

func Copy(dst, src Value) int {
	dstData, dstLen, dstElem := copyData(dst)
	var srcData unsafe.Pointer
	var srcLen int
	var srcElem *rtype
	switch src.kind() {
	case String:
		if dst.kind() != Slice || dstElem.Kind() != Uint8 {
			panic("reflect: Copy from string requires []byte destination")
		}
		sh := (*unsafeheader.String)(src.ptr)
		srcData = sh.Data
		srcLen = sh.Len
		srcElem = dstElem
	default:
		srcData, srcLen, srcElem = copyData(src)
		if dstElem != srcElem {
			panic("reflect: Copy of incompatible types")
		}
	}
	return runtimeTypedSliceCopy(dstElem, dstData, dstLen, srcData, srcLen)
}

func ValueOfPtr(ptr unsafe.Pointer) Value { return Value{} }

type MapIter struct{}

func (it *MapIter) Next() bool   { return false }
func (it *MapIter) Key() Value   { return Value{} }
func (it *MapIter) Value() Value { return Value{} }

func (v Value) MapRange() *MapIter { return &MapIter{} }

func ptrTo(t *rtype) *rtype {
	if t == nil {
		return nil
	}
	if t.ptrToThis != nil {
		return t.ptrToThis
	}
	s := "*" + t.String()
	align := uint8(unsafe.Alignof(uintptr(0)))
	base := &ptrType{
		rtype: rtype{
			size:       unsafe.Sizeof(uintptr(0)),
			ptrdata:    unsafe.Sizeof(uintptr(0)),
			align:      align,
			fieldAlign: align,
			kind:       uint8(Pointer) | kindDirectIface,
			string:     &s,
		},
		elem: t,
	}
	return &base.rtype
}

func copyData(v Value) (unsafe.Pointer, int, *rtype) {
	switch v.kind() {
	case Slice:
		h := (*unsafeheader.Slice)(v.ptr)
		return h.Data, h.Len, (*sliceType)(unsafe.Pointer(v.typ)).elem
	case Array:
		return v.ptr, v.Len(), (*arrayType)(unsafe.Pointer(v.typ)).elem
	default:
		panic("reflect: Copy of non-slice/array value")
	}
}

func deepValueEqual(x, y Value) bool {
	switch x.kind() {
	case Invalid:
		return true
	case Bool:
		return x.Bool() == y.Bool()
	case Int, Int8, Int16, Int32, Int64:
		return x.Int() == y.Int()
	case Uint, Uint8, Uint16, Uint32, Uint64, Uintptr:
		return x.Uint() == y.Uint()
	case Float32, Float64:
		return x.Float() == y.Float()
	case String:
		return x.String() == y.String()
	case Pointer, UnsafePointer:
		if x.IsNil() || y.IsNil() {
			return x.IsNil() == y.IsNil()
		}
		if x.ptr == y.ptr {
			return true
		}
		return deepValueEqual(x.Elem(), y.Elem())
	case Interface:
		if x.IsNil() || y.IsNil() {
			return x.IsNil() == y.IsNil()
		}
		return deepValueEqual(x.Elem(), y.Elem())
	case Slice:
		if x.IsNil() || y.IsNil() {
			return x.IsNil() == y.IsNil()
		}
		xh := (*unsafeheader.Slice)(x.ptr)
		yh := (*unsafeheader.Slice)(y.ptr)
		if xh.Data == yh.Data && xh.Len == yh.Len {
			return true
		}
		if xh.Len != yh.Len {
			return false
		}
		for i := 0; i < xh.Len; i++ {
			if !deepValueEqual(x.Index(i), y.Index(i)) {
				return false
			}
		}
		return true
	case Array:
		if x.Len() != y.Len() {
			return false
		}
		for i := 0; i < x.Len(); i++ {
			if !deepValueEqual(x.Index(i), y.Index(i)) {
				return false
			}
		}
		return true
	case Struct:
		if x.NumField() != y.NumField() {
			return false
		}
		for i := 0; i < x.NumField(); i++ {
			if !deepValueEqual(x.Field(i), y.Field(i)) {
				return false
			}
		}
		return true
	case Func:
		return x.IsNil() && y.IsNil()
	default:
		if x.typ.Comparable() {
			return x.Interface() == y.Interface()
		}
		return false
	}
}

func runtimeNewObject(t *rtype) unsafe.Pointer __asm__("runtime.newobject")
func runtimeMakeSlice(t *rtype, len int, cap int) unsafe.Pointer __asm__("runtime.makeslice")
func runtimeTypedSliceCopy(t *rtype, dst unsafe.Pointer, dstLen int, src unsafe.Pointer, srcLen int) int __asm__("runtime.typedslicecopy")
