package main

import (
	"strings"

	"kos"
)

const (
	memStressIterations      = 120
	memStressWarmupIterations = 10
)

type StressResult struct {
	Iterations  int
	StartKB     uint32
	MinKB       uint32
	AfterPollKB uint32
	AfterGCKB   uint32
	LiveStart   uint32
	LiveEnd     uint32
	LiveBytesStart uint32
	LiveBytesEnd   uint32
}

type pairKey struct {
	a int
	b string
}

var sinkInt int

func forceGC() __asm__("runtime_force_gc")
func pollGC() __asm__("runtime_gc_poll")
func gcLiveObjectCount() uint32 __asm__("runtime_gc_live_object_count")
func gcLiveBytes() uint32 __asm__("runtime_gc_live_bytes_count")

func runStress(iterations int) StressResult {
	if iterations < 1 {
		iterations = 1
	}

	if memStressWarmupIterations > 0 {
		for i := 0; i < memStressWarmupIterations; i++ {
			stressIteration(i)
		}
		pollGC()
		forceGC()
	}

	forceGC()
	start := kos.FreeRAMKB()
	liveStart := gcLiveObjectCount()
	liveBytesStart := gcLiveBytes()
	minFree := start

	for i := 0; i < iterations; i++ {
		stressIteration(i)
		if i%8 == 0 {
			pollGC()
		}
		free := kos.FreeRAMKB()
		if free < minFree {
			minFree = free
		}
	}

	pollGC()
	afterPoll := kos.FreeRAMKB()
	if afterPoll < minFree {
		minFree = afterPoll
	}
	forceGC()
	afterGC := kos.FreeRAMKB()
	if afterGC < minFree {
		minFree = afterGC
	}

	return StressResult{
		Iterations:  iterations,
		StartKB:     start,
		MinKB:       minFree,
		AfterPollKB: afterPoll,
		AfterGCKB:   afterGC,
		LiveStart:   liveStart,
		LiveEnd:     gcLiveObjectCount(),
		LiveBytesStart: liveBytesStart,
		LiveBytesEnd:   gcLiveBytes(),
	}
}

func stressIteration(seed int) {
	var builder strings.Builder
	for i := 0; i < 8; i++ {
		builder.WriteString("chunk")
		builder.WriteByte(byte('a' + byte((seed+i)%26)))
	}
	built := builder.String()
	upper := strings.ToUpper(built)
	parts := strings.Split(upper, "A")
	joined := strings.Join(parts, "-")
	_ = strings.ReplaceAll(joined, "B", "bb")

	m := make(map[string][]byte, 96)
	for i := 0; i < 96; i++ {
		key := "k" + formatInt(seed) + "/" + formatInt(i)
		size := 32 + (i % 64)
		buf := make([]byte, size)
		for j := 0; j < len(buf); j += 7 {
			buf[j] = byte(seed + i + j)
		}
		m[key] = buf
	}
	for i := 0; i < 24; i++ {
		delete(m, "k"+formatInt(seed)+"/"+formatInt(i))
	}

	m2 := make(map[int]string, 64)
	for i := 0; i < 64; i++ {
		m2[i] = joined
	}

	mi := make(map[interface{}]int, 8)
	mi["alpha"] = seed
	mi[uint32(seed)] = seed * 2
	mi[float64(seed)+0.5] = seed * 3
	mi[pairKey{a: seed, b: built}] = seed * 4

	ints := make([]int, 256+(seed%128))
	for i := range ints {
		ints[i] = i + seed
	}

	big := make([]byte, 2048+(seed%1024))
	for i := 0; i < len(big); i += 64 {
		big[i] = byte(seed + i)
	}
	s := string(big[:64])
	if len(s) > 0 {
		sinkInt ^= int(s[0])
	}

	sinkInt ^= len(m) + len(m2) + len(mi) + len(ints)
	if sinkInt == 0x5a5a5a5a {
		sinkInt = 0
	}

	// Clear references to reduce conservative stack retention.
	m = nil
	m2 = nil
	mi = nil
	ints = nil
	big = nil
	parts = nil
	built = ""
	upper = ""
	joined = ""
	s = ""
}
