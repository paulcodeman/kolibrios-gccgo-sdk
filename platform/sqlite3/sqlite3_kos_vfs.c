#include <stddef.h>
#include <stdint.h>
#include <string.h>

#include "upstream/sqlite3.h"

#define SQLITE_KOS_ENCODING_UTF8 3u
#define SQLITE_KOS_MAX_PATH 1024
#define SQLITE_KOS_JULIAN_UNIX_EPOCH_MS 210866760000000LL

#define SQLITE_KOS_FS_READ_FILE 0u
#define SQLITE_KOS_FS_CREATE_FILE 2u
#define SQLITE_KOS_FS_WRITE_FILE 3u
#define SQLITE_KOS_FS_SET_END 4u
#define SQLITE_KOS_FS_GET_INFO 5u
#define SQLITE_KOS_FS_DELETE 8u

#define SQLITE_KOS_FS_OK 0
#define SQLITE_KOS_FS_NOT_FOUND 5
#define SQLITE_KOS_FS_EOF 6
#define SQLITE_KOS_FS_DISK_FULL 8
#define SQLITE_KOS_FS_ACCESS_DENIED 10
#define SQLITE_KOS_FS_NEEDS_MORE_MEMORY 12

#define SQLITE_KOS_ATTR_READ_ONLY 1u
#define SQLITE_KOS_ATTR_DIRECTORY 16u

typedef struct sqlite3_kos_request sqlite3_kos_request;
typedef struct sqlite3_kos_file sqlite3_kos_file;

struct sqlite3_kos_request {
  uint32_t subfunction;
  uint32_t offset;
  uint32_t offset_high_or_flags;
  uint32_t size;
  uint32_t data;
  uint32_t encoding;
  uint32_t path;
};

struct sqlite3_kos_file {
  sqlite3_file base;
  char path[SQLITE_KOS_MAX_PATH];
  int flags;
  int delete_on_close;
};

typedef struct sqlite3_kos_file_info sqlite3_kos_file_info;
struct sqlite3_kos_file_info {
  unsigned char raw[40];
};

extern int32_t sqlite3_kos_file_system_encoded(sqlite3_kos_request* request, uint32_t* secondary)
  __asm__("go_0kos.FileSystemEncoded");
extern int32_t sqlite3_kos_get_current_folder_raw(unsigned char* buffer, uint32_t size, uint32_t encoding)
  __asm__("go_0kos.GetCurrentFolderRaw");
extern uint32_t sqlite3_kos_get_time_counter(void) __asm__("go_0kos.GetTimeCounter");
extern void sqlite3_kos_sleep(uint32_t centiseconds) __asm__("go_0kos.Sleep");
extern sqlite3_int64 sqlite3_kos_current_unix_millis(void);

static int sqlite3_kos_vfs_registered = 0;
static uint32_t sqlite3_kos_temp_counter = 0;

static int sqlite3_kos_close(sqlite3_file* file);
static int sqlite3_kos_read(sqlite3_file* file, void* buffer, int amount, sqlite3_int64 offset);
static int sqlite3_kos_write(sqlite3_file* file, const void* buffer, int amount, sqlite3_int64 offset);
static int sqlite3_kos_truncate(sqlite3_file* file, sqlite3_int64 size);
static int sqlite3_kos_sync(sqlite3_file* file, int flags);
static int sqlite3_kos_file_size(sqlite3_file* file, sqlite3_int64* size);
static int sqlite3_kos_lock(sqlite3_file* file, int lock);
static int sqlite3_kos_unlock(sqlite3_file* file, int lock);
static int sqlite3_kos_check_reserved_lock(sqlite3_file* file, int* result);
static int sqlite3_kos_file_control(sqlite3_file* file, int op, void* arg);
static int sqlite3_kos_sector_size(sqlite3_file* file);
static int sqlite3_kos_device_characteristics(sqlite3_file* file);

static sqlite3_io_methods sqlite3_kos_io_methods = {
  1,
  sqlite3_kos_close,
  sqlite3_kos_read,
  sqlite3_kos_write,
  sqlite3_kos_truncate,
  sqlite3_kos_sync,
  sqlite3_kos_file_size,
  sqlite3_kos_lock,
  sqlite3_kos_unlock,
  sqlite3_kos_check_reserved_lock,
  sqlite3_kos_file_control,
  sqlite3_kos_sector_size,
  sqlite3_kos_device_characteristics
};

static int sqlite3_kos_xopen(sqlite3_vfs* vfs, const char* name, sqlite3_file* file, int flags, int* out_flags);
static int sqlite3_kos_xdelete(sqlite3_vfs* vfs, const char* name, int sync_dir);
static int sqlite3_kos_xaccess(sqlite3_vfs* vfs, const char* name, int flags, int* result);
static int sqlite3_kos_xfull_pathname(sqlite3_vfs* vfs, const char* name, int out_size, char* out);
static void* sqlite3_kos_xdlopen(sqlite3_vfs* vfs, const char* filename);
static void sqlite3_kos_xdlerror(sqlite3_vfs* vfs, int out_size, char* out);
static void (*sqlite3_kos_xdlsym(sqlite3_vfs* vfs, void* handle, const char* symbol))(void);
static void sqlite3_kos_xdlclose(sqlite3_vfs* vfs, void* handle);
static int sqlite3_kos_xrandomness(sqlite3_vfs* vfs, int out_size, char* out);
static int sqlite3_kos_xsleep(sqlite3_vfs* vfs, int microseconds);
static int sqlite3_kos_xcurrent_time(sqlite3_vfs* vfs, double* result);
static int sqlite3_kos_xget_last_error(sqlite3_vfs* vfs, int out_size, char* out);
static int sqlite3_kos_xcurrent_time_int64(sqlite3_vfs* vfs, sqlite3_int64* result);

static sqlite3_vfs sqlite3_kos_vfs = {
  2,
  sizeof(sqlite3_kos_file),
  SQLITE_KOS_MAX_PATH,
  NULL,
  "kos",
  NULL,
  sqlite3_kos_xopen,
  sqlite3_kos_xdelete,
  sqlite3_kos_xaccess,
  sqlite3_kos_xfull_pathname,
  sqlite3_kos_xdlopen,
  sqlite3_kos_xdlerror,
  sqlite3_kos_xdlsym,
  sqlite3_kos_xdlclose,
  sqlite3_kos_xrandomness,
  sqlite3_kos_xsleep,
  sqlite3_kos_xcurrent_time,
  sqlite3_kos_xget_last_error,
  sqlite3_kos_xcurrent_time_int64,
  NULL,
  NULL,
  NULL
};

static int
sqlite3_kos_status_to_open_rc(int status)
{
  switch (status) {
    case SQLITE_KOS_FS_OK:
      return SQLITE_OK;
    case SQLITE_KOS_FS_DISK_FULL:
      return SQLITE_FULL;
    case SQLITE_KOS_FS_ACCESS_DENIED:
      return SQLITE_PERM;
    case SQLITE_KOS_FS_NEEDS_MORE_MEMORY:
      return SQLITE_NOMEM;
  }
  return SQLITE_CANTOPEN;
}

static int
sqlite3_kos_status_to_io_rc(int status, int io_code)
{
  switch (status) {
    case SQLITE_KOS_FS_OK:
      return SQLITE_OK;
    case SQLITE_KOS_FS_DISK_FULL:
      return SQLITE_FULL;
    case SQLITE_KOS_FS_ACCESS_DENIED:
      return SQLITE_PERM;
    case SQLITE_KOS_FS_NEEDS_MORE_MEMORY:
      return SQLITE_NOMEM;
  }
  return io_code;
}

static int
sqlite3_kos_get_current_folder(char* out, int out_size)
{
  int copied = sqlite3_kos_get_current_folder_raw((unsigned char*) out, (uint32_t) out_size, SQLITE_KOS_ENCODING_UTF8);

  if (copied <= 0 || out_size <= 0) {
    return SQLITE_CANTOPEN;
  }
  out[out_size - 1] = '\0';
  return SQLITE_OK;
}

static int
sqlite3_kos_normalize_absolute_path(const char* source, char* out, int out_size)
{
  int checkpoints[128];
  int checkpoint_count = 0;
  int out_pos = 0;
  const char* cursor;

  if (source == NULL || out == NULL || out_size <= 1 || source[0] != '/') {
    return SQLITE_CANTOPEN;
  }

  out[out_pos++] = '/';
  cursor = source + 1;

  for (;;) {
    const char* segment_start;
    int segment_len;
    int checkpoint;
    int i;

    while (*cursor == '/') {
      cursor++;
    }
    if (*cursor == '\0') {
      break;
    }

    segment_start = cursor;
    while (*cursor != '\0' && *cursor != '/') {
      cursor++;
    }
    segment_len = (int) (cursor - segment_start);

    if (segment_len == 1 && segment_start[0] == '.') {
      continue;
    }
    if (segment_len == 2 && segment_start[0] == '.' && segment_start[1] == '.') {
      if (checkpoint_count > 0) {
        checkpoint_count--;
        out_pos = checkpoints[checkpoint_count];
        out[out_pos] = '\0';
      }
      continue;
    }

    checkpoint = out_pos;
    if (out_pos > 1) {
      if (out_pos + 1 >= out_size) {
        return SQLITE_CANTOPEN;
      }
      out[out_pos++] = '/';
    }
    if (out_pos + segment_len >= out_size) {
      return SQLITE_CANTOPEN;
    }
    for (i = 0; i < segment_len; ++i) {
      out[out_pos++] = segment_start[i];
    }
    out[out_pos] = '\0';
    if (checkpoint_count < (int) (sizeof(checkpoints) / sizeof(checkpoints[0]))) {
      checkpoints[checkpoint_count++] = checkpoint;
    }
  }

  if (out_pos == 0) {
    out[0] = '/';
    out[1] = '\0';
  } else {
    out[out_pos] = '\0';
  }
  return SQLITE_OK;
}

static int
sqlite3_kos_full_path(const char* name, char* out, int out_size)
{
  char joined[SQLITE_KOS_MAX_PATH];
  size_t base;
  size_t extra;

  if (name == NULL || name[0] == '\0') {
    return sqlite3_kos_get_current_folder(out, out_size);
  }
  if (name[0] == '/') {
    return sqlite3_kos_normalize_absolute_path(name, out, out_size);
  }
  if (sqlite3_kos_get_current_folder(joined, (int) sizeof(joined)) != SQLITE_OK) {
    return SQLITE_CANTOPEN;
  }
  if ((int) strlen(joined) + 1 + (int) strlen(name) + 1 > (int) sizeof(joined)) {
    return SQLITE_CANTOPEN;
  }
  base = strlen(joined);
  extra = strlen(name);
  if (joined[0] != '\0' && joined[strlen(joined) - 1] == '/') {
    memcpy(joined + base, name, extra + 1);
  } else {
    joined[base++] = '/';
    memcpy(joined + base, name, extra + 1);
  }
  return sqlite3_kos_normalize_absolute_path(joined, out, out_size);
}

static int
sqlite3_kos_make_temp_path(char* out, int out_size)
{
  char cwd[SQLITE_KOS_MAX_PATH];
  uint32_t tick = sqlite3_kos_get_time_counter();
  uint32_t serial = ++sqlite3_kos_temp_counter;

  if (sqlite3_kos_get_current_folder(cwd, (int) sizeof(cwd)) != SQLITE_OK) {
    return SQLITE_CANTOPEN;
  }
  sqlite3_snprintf(out_size, out, "%s/.sqlite-tmp-%u-%u", cwd, tick, serial);
  return SQLITE_OK;
}

static int
sqlite3_kos_get_info(const char* path, sqlite3_kos_file_info* info)
{
  sqlite3_kos_request request;
  int status;

  memset(&request, 0, sizeof(request));
  memset(info, 0, sizeof(*info));
  request.subfunction = SQLITE_KOS_FS_GET_INFO;
  request.data = (uint32_t) (uintptr_t) info;
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) path;

  status = sqlite3_kos_file_system_encoded(&request, NULL);
  return status;
}

static uint32_t
sqlite3_kos_info_attributes(const sqlite3_kos_file_info* info)
{
  return ((uint32_t) info->raw[0]) |
         ((uint32_t) info->raw[1] << 8) |
         ((uint32_t) info->raw[2] << 16) |
         ((uint32_t) info->raw[3] << 24);
}

static sqlite3_int64
sqlite3_kos_info_size(const sqlite3_kos_file_info* info)
{
  sqlite3_int64 size = 0;
  int i;

  for (i = 0; i < 8; ++i) {
    size |= ((sqlite3_int64) info->raw[32 + i]) << (i * 8);
  }
  return size;
}

static int
sqlite3_kos_create_empty_file(const char* path)
{
  sqlite3_kos_request request;
  uint32_t secondary = 0;

  memset(&request, 0, sizeof(request));
  request.subfunction = SQLITE_KOS_FS_CREATE_FILE;
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) path;

  return sqlite3_kos_file_system_encoded(&request, &secondary);
}

static int
sqlite3_kos_xopen(sqlite3_vfs* vfs, const char* name, sqlite3_file* file, int flags, int* out_flags)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;
  sqlite3_kos_file_info info;
  int status;

  (void) vfs;

  memset(kos_file, 0, sizeof(*kos_file));
  kos_file->flags = flags;
  kos_file->delete_on_close = (flags & SQLITE_OPEN_DELETEONCLOSE) != 0;

  if (name == NULL || name[0] == '\0') {
    if (sqlite3_kos_make_temp_path(kos_file->path, (int) sizeof(kos_file->path)) != SQLITE_OK) {
      return SQLITE_CANTOPEN;
    }
  } else if (sqlite3_kos_full_path(name, kos_file->path, (int) sizeof(kos_file->path)) != SQLITE_OK) {
    return SQLITE_CANTOPEN;
  }

  status = sqlite3_kos_get_info(kos_file->path, &info);
  if (status == SQLITE_KOS_FS_OK) {
    if ((sqlite3_kos_info_attributes(&info) & SQLITE_KOS_ATTR_DIRECTORY) != 0) {
      return SQLITE_CANTOPEN;
    }
    if ((flags & SQLITE_OPEN_EXCLUSIVE) != 0 && (flags & SQLITE_OPEN_CREATE) != 0) {
      return SQLITE_CANTOPEN;
    }
    if ((flags & SQLITE_OPEN_READWRITE) != 0 &&
        (sqlite3_kos_info_attributes(&info) & SQLITE_KOS_ATTR_READ_ONLY) != 0) {
      return SQLITE_PERM;
    }
  } else if (status == SQLITE_KOS_FS_NOT_FOUND) {
    if ((flags & SQLITE_OPEN_CREATE) == 0) {
      return SQLITE_CANTOPEN;
    }
    status = sqlite3_kos_create_empty_file(kos_file->path);
    if (status != SQLITE_KOS_FS_OK) {
      return sqlite3_kos_status_to_open_rc(status);
    }
  } else {
    return sqlite3_kos_status_to_open_rc(status);
  }

  kos_file->base.pMethods = &sqlite3_kos_io_methods;
  if (out_flags != NULL) {
    *out_flags = (flags & SQLITE_OPEN_READONLY) != 0 ? SQLITE_OPEN_READONLY : SQLITE_OPEN_READWRITE;
  }
  return SQLITE_OK;
}

static int
sqlite3_kos_xdelete(sqlite3_vfs* vfs, const char* name, int sync_dir)
{
  sqlite3_kos_request request;
  char full_path[SQLITE_KOS_MAX_PATH];
  int status;

  (void) vfs;
  (void) sync_dir;

  if (sqlite3_kos_full_path(name, full_path, (int) sizeof(full_path)) != SQLITE_OK) {
    return SQLITE_IOERR_DELETE;
  }

  memset(&request, 0, sizeof(request));
  request.subfunction = SQLITE_KOS_FS_DELETE;
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) full_path;

  status = sqlite3_kos_file_system_encoded(&request, NULL);
  if (status == SQLITE_KOS_FS_OK || status == SQLITE_KOS_FS_NOT_FOUND) {
    return SQLITE_OK;
  }
  return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_DELETE);
}

static int
sqlite3_kos_xaccess(sqlite3_vfs* vfs, const char* name, int flags, int* result)
{
  sqlite3_kos_file_info info;
  char full_path[SQLITE_KOS_MAX_PATH];
  int status;

  (void) vfs;

  *result = 0;
  if (sqlite3_kos_full_path(name, full_path, (int) sizeof(full_path)) != SQLITE_OK) {
    return SQLITE_OK;
  }

  status = sqlite3_kos_get_info(full_path, &info);
  if (status == SQLITE_KOS_FS_NOT_FOUND) {
    return SQLITE_OK;
  }
  if (status != SQLITE_KOS_FS_OK) {
    return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_ACCESS);
  }

  switch (flags) {
    case SQLITE_ACCESS_EXISTS:
      *result = 1;
      break;
    case SQLITE_ACCESS_READWRITE:
      *result = (sqlite3_kos_info_attributes(&info) & SQLITE_KOS_ATTR_READ_ONLY) == 0;
      break;
    default:
      *result = 1;
      break;
  }
  return SQLITE_OK;
}

static int
sqlite3_kos_xfull_pathname(sqlite3_vfs* vfs, const char* name, int out_size, char* out)
{
  (void) vfs;
  return sqlite3_kos_full_path(name, out, out_size);
}

static void*
sqlite3_kos_xdlopen(sqlite3_vfs* vfs, const char* filename)
{
  (void) vfs;
  (void) filename;
  return NULL;
}

static void
sqlite3_kos_xdlerror(sqlite3_vfs* vfs, int out_size, char* out)
{
  (void) vfs;
  if (out != NULL && out_size > 0) {
    sqlite3_snprintf(out_size, out, "loadable extensions are not supported");
  }
}

static void
(*sqlite3_kos_xdlsym(sqlite3_vfs* vfs, void* handle, const char* symbol))(void)
{
  (void) vfs;
  (void) handle;
  (void) symbol;
  return NULL;
}

static void
sqlite3_kos_xdlclose(sqlite3_vfs* vfs, void* handle)
{
  (void) vfs;
  (void) handle;
}

static int
sqlite3_kos_xrandomness(sqlite3_vfs* vfs, int out_size, char* out)
{
  uint32_t seed = sqlite3_kos_get_time_counter() ^ ++sqlite3_kos_temp_counter;
  int i;

  (void) vfs;

  for (i = 0; i < out_size; ++i) {
    seed = seed * 1103515245u + 12345u;
    out[i] = (char) (seed >> 16);
  }
  return out_size;
}

static int
sqlite3_kos_xsleep(sqlite3_vfs* vfs, int microseconds)
{
  uint32_t centiseconds;

  (void) vfs;

  if (microseconds <= 0) {
    return 0;
  }
  centiseconds = (uint32_t) ((microseconds + 9999) / 10000);
  if (centiseconds == 0) {
    centiseconds = 1;
  }
  sqlite3_kos_sleep(centiseconds);
  return (int) centiseconds * 10000;
}

static int
sqlite3_kos_xcurrent_time(sqlite3_vfs* vfs, double* result)
{
  sqlite3_int64 unix_ms;

  (void) vfs;

  unix_ms = sqlite3_kos_current_unix_millis();
  *result = ((double) (unix_ms + SQLITE_KOS_JULIAN_UNIX_EPOCH_MS)) / 86400000.0;
  return SQLITE_OK;
}

static int
sqlite3_kos_xget_last_error(sqlite3_vfs* vfs, int out_size, char* out)
{
  (void) vfs;
  if (out != NULL && out_size > 0) {
    out[0] = '\0';
  }
  return 0;
}

static int
sqlite3_kos_xcurrent_time_int64(sqlite3_vfs* vfs, sqlite3_int64* result)
{
  sqlite3_int64 unix_ms;

  (void) vfs;

  unix_ms = sqlite3_kos_current_unix_millis();
  *result = unix_ms + SQLITE_KOS_JULIAN_UNIX_EPOCH_MS;
  return SQLITE_OK;
}

static int
sqlite3_kos_close(sqlite3_file* file)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;

  if (kos_file->delete_on_close && kos_file->path[0] != '\0') {
    sqlite3_kos_xdelete(&sqlite3_kos_vfs, kos_file->path, 0);
  }
  memset(kos_file, 0, sizeof(*kos_file));
  return SQLITE_OK;
}

static int
sqlite3_kos_read(sqlite3_file* file, void* buffer, int amount, sqlite3_int64 offset)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;
  sqlite3_kos_request request;
  uint32_t read_amount = 0;
  int status;

  memset(&request, 0, sizeof(request));
  request.subfunction = SQLITE_KOS_FS_READ_FILE;
  request.offset = (uint32_t) offset;
  request.offset_high_or_flags = (uint32_t) (((uint64_t) offset) >> 32);
  request.size = (uint32_t) amount;
  request.data = (uint32_t) (uintptr_t) buffer;
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) kos_file->path;

  status = sqlite3_kos_file_system_encoded(&request, &read_amount);
  if (status == SQLITE_KOS_FS_OK && read_amount == (uint32_t) amount) {
    return SQLITE_OK;
  }
  if ((status == SQLITE_KOS_FS_OK || status == SQLITE_KOS_FS_EOF) && read_amount <= (uint32_t) amount) {
    memset(((unsigned char*) buffer) + read_amount, 0, (size_t) amount - read_amount);
    return SQLITE_IOERR_SHORT_READ;
  }
  return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_READ);
}

static int
sqlite3_kos_write(sqlite3_file* file, const void* buffer, int amount, sqlite3_int64 offset)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;
  sqlite3_kos_request request;
  uint32_t written = 0;
  int status;

  memset(&request, 0, sizeof(request));
  request.subfunction = SQLITE_KOS_FS_WRITE_FILE;
  request.offset = (uint32_t) offset;
  request.offset_high_or_flags = (uint32_t) (((uint64_t) offset) >> 32);
  request.size = (uint32_t) amount;
  request.data = (uint32_t) (uintptr_t) buffer;
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) kos_file->path;

  status = sqlite3_kos_file_system_encoded(&request, &written);
  if (status == SQLITE_KOS_FS_OK && written == (uint32_t) amount) {
    return SQLITE_OK;
  }
  return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_WRITE);
}

static int
sqlite3_kos_truncate(sqlite3_file* file, sqlite3_int64 size)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;
  sqlite3_kos_request request;
  int status;

  memset(&request, 0, sizeof(request));
  request.subfunction = SQLITE_KOS_FS_SET_END;
  request.offset = (uint32_t) size;
  request.offset_high_or_flags = (uint32_t) (((uint64_t) size) >> 32);
  request.encoding = SQLITE_KOS_ENCODING_UTF8;
  request.path = (uint32_t) (uintptr_t) kos_file->path;

  status = sqlite3_kos_file_system_encoded(&request, NULL);
  return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_TRUNCATE);
}

static int
sqlite3_kos_sync(sqlite3_file* file, int flags)
{
  (void) file;
  (void) flags;
  return SQLITE_OK;
}

static int
sqlite3_kos_file_size(sqlite3_file* file, sqlite3_int64* size)
{
  sqlite3_kos_file* kos_file = (sqlite3_kos_file*) file;
  sqlite3_kos_file_info info;
  int status = sqlite3_kos_get_info(kos_file->path, &info);

  if (status != SQLITE_KOS_FS_OK) {
    return sqlite3_kos_status_to_io_rc(status, SQLITE_IOERR_FSTAT);
  }
  *size = sqlite3_kos_info_size(&info);
  return SQLITE_OK;
}

static int
sqlite3_kos_lock(sqlite3_file* file, int lock)
{
  (void) file;
  (void) lock;
  return SQLITE_OK;
}

static int
sqlite3_kos_unlock(sqlite3_file* file, int lock)
{
  (void) file;
  (void) lock;
  return SQLITE_OK;
}

static int
sqlite3_kos_check_reserved_lock(sqlite3_file* file, int* result)
{
  (void) file;
  *result = 0;
  return SQLITE_OK;
}

static int
sqlite3_kos_file_control(sqlite3_file* file, int op, void* arg)
{
  (void) file;
  (void) op;
  (void) arg;
  return SQLITE_NOTFOUND;
}

static int
sqlite3_kos_sector_size(sqlite3_file* file)
{
  (void) file;
  return 4096;
}

static int
sqlite3_kos_device_characteristics(sqlite3_file* file)
{
  (void) file;
  return 0;
}

int
sqlite3_os_init(void)
{
  if (!sqlite3_kos_vfs_registered) {
    sqlite3_vfs_register(&sqlite3_kos_vfs, 1);
    sqlite3_kos_vfs_registered = 1;
  }
  return SQLITE_OK;
}

int
sqlite3_os_end(void)
{
  return SQLITE_OK;
}

int
sqlite3_kos_initialize(void)
{
  return sqlite3_initialize();
}

int
sqlite3_kos_open(const char* filename, sqlite3** db, int flags)
{
  return sqlite3_open_v2(filename, db, flags, "kos");
}

int
sqlite3_kos_prepare(sqlite3* db, const char* sql, sqlite3_stmt** stmt)
{
  return sqlite3_prepare_v2(db, sql, -1, stmt, NULL);
}

int
sqlite3_kos_exec(sqlite3* db, const char* sql)
{
  return sqlite3_exec(db, sql, NULL, NULL, NULL);
}

int
sqlite3_kos_bind_text(sqlite3_stmt* stmt, int index, const char* text, int size)
{
  return sqlite3_bind_text(stmt, index, text, size, SQLITE_TRANSIENT);
}

int
sqlite3_kos_bind_blob(sqlite3_stmt* stmt, int index, const void* data, int size)
{
  return sqlite3_bind_blob(stmt, index, data, size, SQLITE_TRANSIENT);
}
