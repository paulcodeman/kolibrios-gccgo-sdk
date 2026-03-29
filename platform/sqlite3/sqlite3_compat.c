#include <stddef.h>

int
strcmp(const char* left, const char* right)
{
  unsigned char a;
  unsigned char b;

  if (left == right) {
    return 0;
  }
  if (left == NULL) {
    return -1;
  }
  if (right == NULL) {
    return 1;
  }
  while (*left != '\0' && *right != '\0') {
    if (*left != *right) {
      break;
    }
    left++;
    right++;
  }
  a = (unsigned char) *left;
  b = (unsigned char) *right;
  return (int) a - (int) b;
}

int
strncmp(const char* left, const char* right, size_t count)
{
  size_t index;

  if (count == 0 || left == right) {
    return 0;
  }
  if (left == NULL) {
    return -1;
  }
  if (right == NULL) {
    return 1;
  }
  for (index = 0; index < count; ++index) {
    unsigned char a = (unsigned char) left[index];
    unsigned char b = (unsigned char) right[index];

    if (a != b) {
      return (int) a - (int) b;
    }
    if (a == '\0') {
      return 0;
    }
  }
  return 0;
}

void*
strchr(const char* value, int target)
{
  unsigned char ch = (unsigned char) target;

  if (value == NULL) {
    return NULL;
  }
  while (*value != '\0') {
    if ((unsigned char) *value == ch) {
      return (void*) value;
    }
    value++;
  }
  if (ch == '\0') {
    return (void*) value;
  }
  return NULL;
}

void*
strrchr(const char* value, int target)
{
  const char* last = NULL;
  unsigned char ch = (unsigned char) target;

  if (value == NULL) {
    return NULL;
  }
  while (*value != '\0') {
    if ((unsigned char) *value == ch) {
      last = value;
    }
    value++;
  }
  if (ch == '\0') {
    return (void*) value;
  }
  return (void*) last;
}

size_t
strspn(const char* value, const char* accept)
{
  size_t count = 0;

  if (value == NULL || accept == NULL) {
    return 0;
  }
  while (*value != '\0') {
    const char* cursor = accept;
    int found = 0;

    while (*cursor != '\0') {
      if (*cursor == *value) {
        found = 1;
        break;
      }
      cursor++;
    }
    if (!found) {
      break;
    }
    count++;
    value++;
  }
  return count;
}

size_t
strcspn(const char* value, const char* reject)
{
  size_t count = 0;

  if (value == NULL || reject == NULL) {
    return 0;
  }
  while (*value != '\0') {
    const char* cursor = reject;

    while (*cursor != '\0') {
      if (*cursor == *value) {
        return count;
      }
      cursor++;
    }
    count++;
    value++;
  }
  return count;
}

void*
memchr(const void* src, int target, size_t count)
{
  const unsigned char* bytes = (const unsigned char*) src;
  unsigned char ch = (unsigned char) target;
  size_t index;

  if (src == NULL) {
    return NULL;
  }
  for (index = 0; index < count; ++index) {
    if (bytes[index] == ch) {
      return (void*) (bytes + index);
    }
  }
  return NULL;
}

void*
__memcpy_chk(void* dst, const void* src, size_t len, size_t dstlen)
{
  unsigned char* out = (unsigned char*) dst;
  const unsigned char* in = (const unsigned char*) src;
  size_t index;

  if (len > dstlen) {
    len = dstlen;
  }
  for (index = 0; index < len; ++index) {
    out[index] = in[index];
  }
  return dst;
}
