/* OS boundary for libvterm diagnostics; formatting stays in upstream printf. */
#include <stdio.h>
#include <stdarg.h>
#include <stddef.h>
#include "format_print.h"

extern void sdk_console_debug_byte(int byte);
extern void sdk_console_terminate(void) __attribute__((noreturn));
static unsigned int stderr_tag;
FILE *sdk_console_stderr = (FILE *)&stderr_tag;

static void diagnostic_byte(char byte, void *buffer, size_t index, size_t capacity)
{
    (void)buffer; (void)index; (void)capacity;
    if (byte) sdk_console_debug_byte((unsigned char)byte);
}

int sdk_console_fprintf(FILE *stream, const char *format, ...)
{
    if (stream != sdk_console_stderr) return -1;
    va_list args;
    va_start(args, format);
    int written = _vsnprintf(diagnostic_byte, NULL, (size_t)-1, format, args);
    va_end(args);
    return written;
}

void sdk_console_exit(int code)
{
    sdk_console_fprintf(sdk_console_stderr, "CONSOLE_VTERM_FATAL exit %d\n", code);
    sdk_console_terminate();
}

void sdk_console_abort(void)
{
    sdk_console_exit(1);
}
