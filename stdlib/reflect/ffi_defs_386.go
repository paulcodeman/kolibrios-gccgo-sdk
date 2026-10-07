// Code generated from libffi ffi.h with gcc -m32 -fdump-go-spec.
// Only FFI definitions used by the KolibriOS adapter are retained.
package reflect

type _ffi_abi uint32
type __ffi_type struct {
	size      uint32
	alignment uint16
	_type     uint16
	elements  **__ffi_type
}
type _ffi_status uint32
type _ffi_cif struct {
	abi       uint32
	nargs     uint32
	arg_types **__ffi_type
	rtype     *__ffi_type
	bytes     uint32
	flags     uint32
}

const _FFI_TYPE_STRUCT = 13
const _FFI_CLOSURES = 1
const _FFI_TYPE_VOID = 0
const _FFI_BAD_TYPEDEF = 1
const _FFI_TYPE_SINT8 = 6
const _FFI_TYPE_SINT32 = 10
const _FFI_SIZEOF_JAVA_RAW = _FFI_SIZEOF_ARG
const _FFI_TYPE_UINT16 = 7
const _FFI_OK = 0
const _FFI_TRAMPOLINE_SIZE = 16
const _FFI_SIZEOF_ARG = 4
const _FFI_FIRST_ABI = 0
const _FFI_BAD_ABI = 2
const _FFI_PASCAL = 6
const _FFI_NATIVE_RAW_API = 1
const _FFI_TYPE_UINT8 = 5
const _FFI_DEFAULT_ABI = 1
const _FFI_TYPE_POINTER = 14
const _FFI_FASTCALL = 4
const _FFI_TYPE_LAST = _FFI_TYPE_COMPLEX
const _FFI_TYPE_UINT32 = 9
const _FFI_TYPE_UINT64 = 11
const _FFI_GO_CLOSURES = 1
const _FFI_REGISTER = 7
const _FFI_TYPE_FLOAT = 2
const _FFI_TYPE_COMPLEX = 15
const _FFI_THISCALL = 3
const _FFI_BAD_ARGTYPE = 3
const _FFI_TYPE_LONGDOUBLE = 4
const _FFI_STDCALL = 5
const _FFI_64_BIT_MAX = 9223372036854775807
const _FFI_LAST_ABI = 9
const _FFI_TYPE_INT = 1
const _FFI_SYSV = 1
const _FFI_TYPE_DOUBLE = 3
const _FFI_MS_CDECL = 8
const _FFI_TYPE_SINT64 = 12
const _FFI_TYPE_SINT16 = 8
