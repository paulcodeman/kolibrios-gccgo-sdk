#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct runtime_kolibrios_aligned_alloc_header {
  void *raw;
} runtime_kolibrios_aligned_alloc_header;

typedef struct runtime_kolibrios_reservation {
  void *raw_base;
  void *base;
  uintptr_t size;
  struct runtime_kolibrios_reservation *next;
} runtime_kolibrios_reservation;

static runtime_kolibrios_reservation *runtime_kolibrios_reservations;

static uintptr_t
runtime_kolibrios_align_up_pow2(uintptr_t value, uintptr_t align) {
  return (value + align - 1u) & ~(align - 1u);
}

static void *
runtime_kolibrios_alloc_aligned_zeroed(uintptr_t size, uintptr_t align) {
  uintptr_t total;
  void *raw;
  uintptr_t start;
  runtime_kolibrios_aligned_alloc_header *header;

  if (size == 0u) {
    return NULL;
  }
  if (align < sizeof(void *)) {
    align = sizeof(void *);
  }
  if ((align & (align - 1u)) != 0u) {
    return NULL;
  }
  if (size > (uintptr_t)-1 - align - sizeof(*header)) {
    return NULL;
  }

  total = size + align + sizeof(*header);
  raw = malloc((size_t)total);
  if (raw == NULL) {
    return NULL;
  }
  memset(raw, 0, (size_t)total);

  start = runtime_kolibrios_align_up_pow2((uintptr_t)raw + sizeof(*header), align);
  header = (runtime_kolibrios_aligned_alloc_header *)(start - sizeof(*header));
  header->raw = raw;
  return (void *)start;
}

static void
runtime_kolibrios_free_aligned(void *ptr) {
  runtime_kolibrios_aligned_alloc_header *header;

  if (ptr == NULL) {
    return;
  }
  header = (runtime_kolibrios_aligned_alloc_header *)((uintptr_t)ptr - sizeof(*header));
  free(header->raw);
}

void *
runtime_kolibrios_libgo_reserve(uintptr_t size) {
  runtime_kolibrios_reservation *reservation;
  void *base;

  base = runtime_kolibrios_alloc_aligned_zeroed(size, 4096u);
  if (base == NULL) {
    return NULL;
  }

  reservation = (runtime_kolibrios_reservation *)malloc(sizeof(*reservation));
  if (reservation == NULL) {
    runtime_kolibrios_free_aligned(base);
    return NULL;
  }

  reservation->raw_base = base;
  reservation->base = base;
  reservation->size = size;
  reservation->next = runtime_kolibrios_reservations;
  runtime_kolibrios_reservations = reservation;
  return base;
}

void
runtime_kolibrios_libgo_release(void *ptr, uintptr_t size) {
  runtime_kolibrios_reservation *prev;
  runtime_kolibrios_reservation *reservation;
  uintptr_t start;
  uintptr_t end;
  uintptr_t reservation_start;
  uintptr_t reservation_end;

  if (ptr == NULL || size == 0u) {
    return;
  }

  start = (uintptr_t)ptr;
  end = start + size;

  prev = NULL;
  reservation = runtime_kolibrios_reservations;
  while (reservation != NULL) {
    reservation_start = (uintptr_t)reservation->base;
    reservation_end = reservation_start + reservation->size;
    if (start >= reservation_start && end <= reservation_end) {
      break;
    }
    prev = reservation;
    reservation = reservation->next;
  }
  if (reservation == NULL) {
    return;
  }

  if (start == reservation_start && end == reservation_end) {
    if (prev != NULL) {
      prev->next = reservation->next;
    } else {
      runtime_kolibrios_reservations = reservation->next;
    }
    runtime_kolibrios_free_aligned(reservation->raw_base);
    free(reservation);
    return;
  }

  if (start == reservation_start) {
    reservation->base = (void *)end;
    reservation->size = reservation_end - end;
    return;
  }

  if (end == reservation_end) {
    reservation->size = start - reservation_start;
    return;
  }

  /* Middle splits are not needed by the current arena bootstrap path. */
}
