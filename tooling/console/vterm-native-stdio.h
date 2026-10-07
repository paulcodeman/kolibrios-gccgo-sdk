/* Native diagnostics use the kernel debug board rather than host stdio. */
#include <stdio.h>
#undef stderr
#define stderr sdk_console_stderr
extern FILE *sdk_console_stderr;
