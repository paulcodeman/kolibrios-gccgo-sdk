#include <stddef.h>
#include <stdint.h>

extern int runtime_console_bridge_write(uint32_t data, uint32_t size);

static void*
compat_memcpy_impl(void* dst, const void* src, size_t n)
{
  unsigned char* out = (unsigned char*) dst;
  const unsigned char* in = (const unsigned char*) src;
  size_t i;

  for (i = 0; i < n; ++i)
    out[i] = in[i];
  return dst;
}

static void*
compat_memmove_impl(void* dst, const void* src, size_t n)
{
  unsigned char* out = (unsigned char*) dst;
  const unsigned char* in = (const unsigned char*) src;
  size_t i;

  if (out == in || n == 0)
    return dst;
  if (out < in) {
    for (i = 0; i < n; ++i)
      out[i] = in[i];
  } else {
    for (i = n; i > 0; --i)
      out[i - 1] = in[i - 1];
  }
  return dst;
}

static int
compat_memcmp_impl(const void* left, const void* right, size_t n)
{
  const unsigned char* a = (const unsigned char*) left;
  const unsigned char* b = (const unsigned char*) right;
  size_t i;

  for (i = 0; i < n; ++i) {
    if (a[i] != b[i])
      return (int) a[i] - (int) b[i];
  }
  return 0;
}

static void*
compat_memchr_impl(const void* src, int value, size_t n)
{
  const unsigned char* bytes = (const unsigned char*) src;
  unsigned char target = (unsigned char) value;
  size_t i;

  for (i = 0; i < n; ++i) {
    if (bytes[i] == target)
      return (void*) (bytes + i);
  }
  return NULL;
}

static size_t
compat_strlen_impl(const char* value)
{
  size_t len = 0;

  if (value == NULL)
    return 0;
  while (value[len] != '\0')
    ++len;
  return len;
}

static int
compat_strcmp_impl(const char* left, const char* right)
{
  unsigned char a;
  unsigned char b;

  if (left == right)
    return 0;
  if (left == NULL)
    return -1;
  if (right == NULL)
    return 1;
  while (*left != '\0' && *right != '\0') {
    if (*left != *right)
      break;
    ++left;
    ++right;
  }
  a = (unsigned char) *left;
  b = (unsigned char) *right;
  return (int) a - (int) b;
}

void*
memcpy(void* dst, const void* src, size_t n)
{
  return compat_memcpy_impl(dst, src, n);
}

void*
memmove(void* dst, const void* src, size_t n)
{
  return compat_memmove_impl(dst, src, n);
}

int
memcmp(const void* left, const void* right, size_t n)
{
  return compat_memcmp_impl(left, right, n);
}

void*
memchr(const void* src, int value, size_t n)
{
  return compat_memchr_impl(src, value, n);
}

size_t
strlen(const char* value)
{
  return compat_strlen_impl(value);
}

int
strcmp(const char* left, const char* right)
{
  return compat_strcmp_impl(left, right);
}

void*
__memset_chk(void* dst, int value, size_t len, size_t dstlen)
{
  unsigned char* out = (unsigned char*) dst;
  size_t i;

  if (len > dstlen)
    len = dstlen;
  for (i = 0; i < len; ++i)
    out[i] = (unsigned char) value;
  return dst;
}

static int compat_errno_value;

int*
__errno_location(void)
{
  return &compat_errno_value;
}

int
write(int fd, const void* data, size_t size)
{
  if ((fd == 1 || fd == 2) && data != NULL && size != 0) {
    if (runtime_console_bridge_write((uint32_t) (uintptr_t) data, (uint32_t) size))
      return (int) size;
  }
  compat_errno_value = 5;
  return -1;
}

struct dl_find_object;

int
_dl_find_object(void* address, struct dl_find_object* result)
{
  (void) address;
  (void) result;
  return -1;
}

__attribute__((noreturn)) void
abort(void)
{
  __asm__ __volatile__(
      "int $0x40"
      :
      : "a"(UINT32_MAX), "b"(1u)
      : "ecx", "edx", "esi", "edi", "memory", "cc");
  __builtin_unreachable();
}
