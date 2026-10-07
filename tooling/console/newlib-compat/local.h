#include <_ansi.h>
/* Upstream's inhibit-loop-to-libcall attribute, without unused locale APIs. */
#define __inhibit_loop_to_libcall \
  __attribute__((__optimize__("-fno-tree-loop-distribute-patterns")))
