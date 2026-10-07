/* gccgo layouts shared with reflect.go and runtime_gccgo.c. */
#include <stdbool.h>
#include <stdint.h>
struct _type {
    uintptr_t size, ptrdata;
    uint32_t hash;
    uint8_t tflag, align, fieldAlign, kind;
    const void *equal, *gcdata, *name, *uncommon, *ptr_to_this;
};
typedef struct { void *__values; intptr_t __count, __capacity; } Slice;
struct functype { struct _type typ; bool dotdotdot; Slice in, out; };
typedef struct { uintptr_t fn; } FuncVal;
#define kindMask 31
#define kindBool 1
#define kindInt8 3
#define kindInt16 4
#define kindInt32 5
#define kindUint8 8
#define kindUint16 9
#define kindUint32 10
#define kindFunc 19
_Static_assert(sizeof(struct _type) == 36, "gccgo 386 type descriptor");
_Static_assert(sizeof(struct functype) == 64, "gccgo 386 function descriptor");
