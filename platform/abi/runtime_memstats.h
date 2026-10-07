/* Go/libgo MemStats ABI. Ordinary uint64 alignment is 4 on 386. */
typedef struct {
    uint64_t Alloc, TotalAlloc, Sys, Lookups, Mallocs, Frees;
    uint64_t HeapAlloc, HeapSys, HeapIdle, HeapInuse, HeapReleased, HeapObjects;
    uint64_t StackInuse, StackSys, MSpanInuse, MSpanSys, MCacheInuse, MCacheSys;
    uint64_t BuckHashSys, GCSys, OtherSys, NextGC, LastGC, PauseTotalNs;
    uint64_t PauseNs[256], PauseEnd[256];
    uint32_t NumGC, NumForcedGC;
    double GCCPUFraction;
    bool EnableGC, DebugGC;
    struct { uint32_t Size; uint64_t Mallocs, Frees; } BySize[61];
} runtime_kolibri_memstats;

void runtime_kolibri_read_memstats(runtime_kolibri_memstats* stats) {
    bool multiple_threads = runtime_atomic_load_u32(&runtime_m_count) > 1u;
    if (stats == NULL) runtime_panicmem();
    if (multiple_threads) runtime_lock_mutex(&runtime_gc_lock);
    runtime_stop_world();
    kos_memset(stats, 0, sizeof(*stats));
    stats->Alloc = stats->HeapAlloc = runtime_gc_live_bytes;
    stats->TotalAlloc = runtime_gc_alloc_bytes;
    stats->Mallocs = runtime_gc_alloc_count;
    stats->HeapObjects = runtime_gc_live_objects;
    stats->Frees = stats->Mallocs - stats->HeapObjects;
    stats->NextGC = runtime_gc_threshold;
    stats->NumGC = (uint32_t)runtime_gc_collection_count;
    stats->EnableGC = true;
    runtime_start_world();
    if (multiple_threads) runtime_unlock_mutex(&runtime_gc_lock);
}
