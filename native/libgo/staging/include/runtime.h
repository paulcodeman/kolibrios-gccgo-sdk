// Minimal runtime header for libgo helper C files compiled outside the main
// runtime C layer. The full upstream runtime/runtime.h depends on generated
// autotools artifacts such as config.h and runtime.inc. These helpers only
// need the basic ABI types and symbol prefix macros.

#ifndef KOLIBRI_LIBGO_MIN_RUNTIME_H
#define KOLIBRI_LIBGO_MIN_RUNTIME_H

#include <stdint.h>

#define _STRINGIFY2_(x) #x
#define _STRINGIFY_(x) _STRINGIFY2_(x)
#define GOSYM_PREFIX _STRINGIFY_(__USER_LABEL_PREFIX__)

typedef signed int int8 __attribute__((mode(QI)));
typedef unsigned int uint8 __attribute__((mode(QI)));
typedef signed int int16 __attribute__((mode(HI)));
typedef unsigned int uint16 __attribute__((mode(HI)));
typedef signed int int32 __attribute__((mode(SI)));
typedef unsigned int uint32 __attribute__((mode(SI)));
typedef signed int int64 __attribute__((mode(DI)));
typedef unsigned int uint64 __attribute__((mode(DI)));
typedef signed int intptr __attribute__((mode(pointer)));
typedef unsigned int uintptr __attribute__((mode(pointer)));

typedef intptr intgo;
typedef uintptr uintgo;

typedef _Bool bool;
typedef uint8 byte;

typedef struct String String;

struct String {
	const byte *str;
	intgo len;
};

#define nil ((void *)0)

#ifndef true
#define true 1
#endif

#ifndef false
#define false 0
#endif

#endif
