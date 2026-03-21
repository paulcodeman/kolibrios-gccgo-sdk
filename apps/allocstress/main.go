package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"kos"
)

const (
	defaultBaseIterations = 12000
	minBaseIterations     = 1
	consoleTitle          = "Allocator Stress"
)

var (
	sinkInt    int
	sinkString string
	sinkBytes  []byte
	sinkIface  interface{}
)

type counters struct {
	heapAllocCount   uint32
	heapAllocBytes   uint32
	heapFreeCount    uint32
	heapReallocCount uint32
	heapReallocBytes uint32
	gcAllocCount     uint32
	gcAllocBytes     uint32
	gcCollections    uint32
	gcLiveBytes      uint32
	gcThreshold      uint32
	gcPollRetry      uint32
}

type phaseSpec struct {
	name        string
	area        string
	iterations  int
	sampleEvery int
	fn          func(int, *phaseSampler)
}

type phaseResult struct {
	name              string
	area              string
	iterations        int
	elapsedNS         uint64
	pollGCNS          uint64
	forceGCNS         uint64
	startFreeKB       uint32
	minFreeKB         uint32
	endFreeKB         uint32
	afterGCFreeKB     uint32
	startLiveBytes    uint32
	peakLiveBytes     uint32
	endLiveBytes      uint32
	afterGCLiveBytes  uint32
	startLiveObjects  uint32
	peakLiveObjects   uint32
	endLiveObjects    uint32
	afterGCLiveObjects uint32
	delta             counters
}

type phaseSampler struct {
	sampleEvery     int
	minFreeKB       uint32
	peakLiveBytes   uint32
	peakLiveObjects uint32
}

type pairKey struct {
	a int
	b string
}

type smallNode struct {
	left  int
	right int
	next  *smallNode
}

func forceGC() __asm__("runtime_force_gc")
func gcLiveObjectCount() uint32 __asm__("runtime_gc_live_object_count")

func main() {
	console, ok := kos.OpenConsole(consoleTitle)
	if !ok {
		kos.DebugString("allocstress: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / ready")
	}

	baseIterations := defaultBaseIterations
	if len(os.Args) > 1 {
		if parsed, err := strconv.Atoi(os.Args[1]); err == nil && parsed >= minBaseIterations {
			baseIterations = parsed
		}
	}

	fmt.Printf("allocator stress suite, base=%d\n", baseIterations)
	fmt.Printf("runtime_threads=%d runtime_m=%d total_ram_kb=%d free_ram_kb=%d\n",
		kos.GetRuntimeThreadsRaw(),
		kos.GetRuntimeMCountRaw(),
		kos.TotalRAMKB(),
		kos.FreeRAMKB(),
	)
	fmt.Println("time excludes the final forced GC; memory shows start/peak/end/after_gc.")
	fmt.Println("")

	warmup(baseIterations)
	results := runSuite(baseIterations, console)
	printSummary(results)
	printSuspects(results)
	waitForExit(console)

	_ = sinkInt
	_ = sinkString
	_ = sinkBytes
	_ = sinkIface
}

func warmup(baseIterations int) {
	forceGC()
	workloadAllocSmall(maxInt(baseIterations/16, 32), newPhaseSampler(32))
	workloadMapString(maxInt(baseIterations/32, 16), newPhaseSampler(8))
	workloadMixedRuntime(maxInt(baseIterations/64, 8), newPhaseSampler(4))
	kos.PollRuntimeGCRaw()
	forceGC()
}

func runSuite(baseIterations int, console kos.Console) []phaseResult {
	specs := []phaseSpec{
		{
			name:        "plain-loop",
			area:        "control",
			iterations:  maxInt(baseIterations*256, 1000000),
			sampleEvery: 0,
			fn:          workloadPlainLoop,
		},
		{
			name:        "alloc-small",
			area:        "allocator",
			iterations:  maxInt(baseIterations, 1),
			sampleEvery: 128,
			fn:          workloadAllocSmall,
		},
		{
			name:        "map-int",
			area:        "map/int",
			iterations:  maxInt(baseIterations/4, 128),
			sampleEvery: 32,
			fn:          workloadMapInt,
		},
		{
			name:        "map-string",
			area:        "map/string",
			iterations:  maxInt(baseIterations/6, 96),
			sampleEvery: 16,
			fn:          workloadMapString,
		},
		{
			name:        "slice-grow",
			area:        "slice/realloc",
			iterations:  maxInt(baseIterations/3, 128),
			sampleEvery: 32,
			fn:          workloadSliceGrow,
		},
		{
			name:        "string-churn",
			area:        "string",
			iterations:  maxInt(baseIterations/3, 128),
			sampleEvery: 32,
			fn:          workloadStringChurn,
		},
		{
			name:        "iface-box",
			area:        "interface",
			iterations:  maxInt(baseIterations, 1),
			sampleEvery: 128,
			fn:          workloadInterfaceBox,
		},
		{
			name:        "mixed-runtime",
			area:        "mixed",
			iterations:  maxInt(baseIterations/10, 64),
			sampleEvery: 8,
			fn:          workloadMixedRuntime,
		},
	}

	results := make([]phaseResult, 0, len(specs))
	for _, spec := range specs {
		if console.SupportsTitle() {
			console.SetTitle(consoleTitle + " / " + spec.name)
		}
		fmt.Printf("[run] %s area=%s iter=%d\n", spec.name, spec.area, spec.iterations)
		results = append(results, runPhase(spec))
		last := results[len(results)-1]
		fmt.Printf("[ok]  %s time_ms=%d ns_iter=%d live_peak=%dKB free_drop=%dKB gc=%d\n",
			last.name,
			nsToMilliseconds(last.elapsedNS),
			last.nsPerIter(),
			bytesToKB(last.peakLiveBytes),
			last.sysDropKB(),
			last.delta.gcCollections,
		)
	}
	if console.SupportsTitle() {
		console.SetTitle(consoleTitle + " / done")
	}
	return results
}

func runPhase(spec phaseSpec) phaseResult {
	forceGC()
	before := readCounters()
	startFreeKB := kos.FreeRAMKB()
	startLiveBytes := kos.GCLiveBytesRaw()
	startLiveObjects := gcLiveObjectCount()

	sampler := newPhaseSampler(spec.sampleEvery)
	start := kos.UptimeNanoseconds()
	spec.fn(spec.iterations, sampler)
	elapsedNS := kos.UptimeNanoseconds() - start
	sampler.sample()

	endFreeKB := kos.FreeRAMKB()
	endLiveBytes := kos.GCLiveBytesRaw()
	endLiveObjects := gcLiveObjectCount()

	pollStart := kos.UptimeNanoseconds()
	kos.PollRuntimeGCRaw()
	pollGCNS := kos.UptimeNanoseconds() - pollStart
	forceStart := kos.UptimeNanoseconds()
	forceGC()
	forceGCNS := kos.UptimeNanoseconds() - forceStart

	after := readCounters()
	afterGCFreeKB := kos.FreeRAMKB()
	afterGCLiveBytes := kos.GCLiveBytesRaw()
	afterGCLiveObjects := gcLiveObjectCount()

	return phaseResult{
		name:               spec.name,
		area:               spec.area,
		iterations:         spec.iterations,
		elapsedNS:          elapsedNS,
		pollGCNS:           pollGCNS,
		forceGCNS:          forceGCNS,
		startFreeKB:        startFreeKB,
		minFreeKB:          sampler.minFreeKB,
		endFreeKB:          endFreeKB,
		afterGCFreeKB:      afterGCFreeKB,
		startLiveBytes:     startLiveBytes,
		peakLiveBytes:      sampler.peakLiveBytes,
		endLiveBytes:       endLiveBytes,
		afterGCLiveBytes:   afterGCLiveBytes,
		startLiveObjects:   startLiveObjects,
		peakLiveObjects:    sampler.peakLiveObjects,
		endLiveObjects:     endLiveObjects,
		afterGCLiveObjects: afterGCLiveObjects,
		delta:              diffCounters(after, before),
	}
}

func printSummary(results []phaseResult) {
	fmt.Println("")
	fmt.Println("phases:")
	for _, result := range results {
		fmt.Printf("%-13s area=%-13s iter=%7d time_ms=%6d ns_iter=%6d\n",
			result.name,
			result.area,
			result.iterations,
			nsToMilliseconds(result.elapsedNS),
			result.nsPerIter(),
		)
		fmt.Printf("  gc_time   poll_us=%5d force_us=%5d collections=%3d threshold=%5dKB\n",
			nsToMicroseconds(result.pollGCNS),
			nsToMicroseconds(result.forceGCNS),
			result.delta.gcCollections,
			bytesToKB(result.delta.gcThreshold),
		)
		fmt.Printf("  live_kb   start=%5d peak=%5d end=%5d after_gc=%5d   objs start=%5d peak=%5d end=%5d after_gc=%5d\n",
			bytesToKB(result.startLiveBytes),
			bytesToKB(result.peakLiveBytes),
			bytesToKB(result.endLiveBytes),
			bytesToKB(result.afterGCLiveBytes),
			result.startLiveObjects,
			result.peakLiveObjects,
			result.endLiveObjects,
			result.afterGCLiveObjects,
		)
		fmt.Printf("  free_kb   start=%5d min=%5d end=%5d after_gc=%5d drop=%5d recover=%5d\n",
			result.startFreeKB,
			result.minFreeKB,
			result.endFreeKB,
			result.afterGCFreeKB,
			result.sysDropKB(),
			result.sysRecoverKB(),
		)
		fmt.Printf("  counters  heap_alloc=%6dKB(%5d) free=%5d realloc=%6dKB(%5d) gc_alloc=%6dKB(%5d) gc=%3d poll_retry=%3d\n",
			bytesToKB(result.delta.heapAllocBytes),
			result.delta.heapAllocCount,
			result.delta.heapFreeCount,
			bytesToKB(result.delta.heapReallocBytes),
			result.delta.heapReallocCount,
			bytesToKB(result.delta.gcAllocBytes),
			result.delta.gcAllocCount,
			result.delta.gcCollections,
			result.delta.gcPollRetry,
		)
	}
}

func printSuspects(results []phaseResult) {
	fmt.Println("")
	fmt.Println("suspects:")

	plain := findPhase(results, "plain-loop")
	mapInt := findPhase(results, "map-int")
	mapString := findPhase(results, "map-string")
	sliceGrow := findPhase(results, "slice-grow")
	ifaceBox := findPhase(results, "iface-box")
	allocSmall := findPhase(results, "alloc-small")
	mixed := findPhase(results, "mixed-runtime")

	printed := 0
	if mapString != nil && mapInt != nil && mapString.nsPerIter() > mapInt.nsPerIter()*3/2 {
		fmt.Printf("- map/string: %d ns/iter vs map/int %d ns/iter, string hash/compare path still dominates.\n",
			mapString.nsPerIter(),
			mapInt.nsPerIter(),
		)
		printed++
	}
	if sliceGrow != nil && sliceGrow.delta.heapReallocCount > uint32(maxInt(sliceGrow.iterations/4, 1)) {
		fmt.Printf("- slice/realloc: realloc count is high (%d over %d iterations), growth/memmove path is still expensive.\n",
			sliceGrow.delta.heapReallocCount,
			sliceGrow.iterations,
		)
		printed++
	}
	if ifaceBox != nil && plain != nil &&
		plain.nsPerIter() >= 100 &&
		ifaceBox.nsPerIter() > plain.nsPerIter()*8 &&
		(allocSmall == nil || ifaceBox.nsPerIter() > allocSmall.nsPerIter()/2) {
		fmt.Printf("- interface: %d ns/iter vs control %d ns/iter, boxing/type-assert/itab path is still heavy.\n",
			ifaceBox.nsPerIter(),
			plain.nsPerIter(),
		)
		printed++
	}
	if allocSmall != nil && allocSmall.delta.gcCollections > 0 && allocSmall.delta.gcAllocBytes > 512*1024 {
		fmt.Printf("- allocator: alloc-small triggered %d GC cycles and %d KB of GC alloc, churn is still visible.\n",
			allocSmall.delta.gcCollections,
			bytesToKB(allocSmall.delta.gcAllocBytes),
		)
		printed++
	}
	if mixed != nil && mixed.afterGCLiveBytes > mixed.startLiveBytes+128*1024 {
		fmt.Printf("- retention: mixed phase keeps %d KB more live bytes after GC, cache flush or root retention may still be off.\n",
			bytesToKB(mixed.afterGCLiveBytes-mixed.startLiveBytes),
		)
		printed++
	}
	if mixed != nil && mixed.sysDropKB() > 256 {
		fmt.Printf("- gc-wave: mixed phase dropped free RAM by %d KB before recovery, wave amplitude is still large.\n",
			mixed.sysDropKB(),
		)
		printed++
	}
	if printed == 0 {
		slowest := slowestPhase(results)
		if slowest != nil {
			fmt.Printf("- slowest phase right now is %s at %d ns/iter; use it as the next optimization target.\n",
				slowest.name,
				slowest.nsPerIter(),
			)
		}
	}
}

func readCounters() counters {
	return counters{
		heapAllocCount:   kos.HeapAllocCountRaw(),
		heapAllocBytes:   kos.HeapAllocBytesRaw(),
		heapFreeCount:    kos.HeapFreeCountRaw(),
		heapReallocCount: kos.HeapReallocCountRaw(),
		heapReallocBytes: kos.HeapReallocBytesRaw(),
		gcAllocCount:     kos.GCAllocCountRaw(),
		gcAllocBytes:     kos.GCAllocBytesRaw(),
		gcCollections:    kos.GCCollectionCountRaw(),
		gcLiveBytes:      kos.GCLiveBytesRaw(),
		gcThreshold:      kos.GCThresholdRaw(),
		gcPollRetry:      kos.GCPollRetryRaw(),
	}
}

func diffCounters(after counters, before counters) counters {
	return counters{
		heapAllocCount:   after.heapAllocCount - before.heapAllocCount,
		heapAllocBytes:   after.heapAllocBytes - before.heapAllocBytes,
		heapFreeCount:    after.heapFreeCount - before.heapFreeCount,
		heapReallocCount: after.heapReallocCount - before.heapReallocCount,
		heapReallocBytes: after.heapReallocBytes - before.heapReallocBytes,
		gcAllocCount:     after.gcAllocCount - before.gcAllocCount,
		gcAllocBytes:     after.gcAllocBytes - before.gcAllocBytes,
		gcCollections:    after.gcCollections - before.gcCollections,
		gcLiveBytes:      after.gcLiveBytes - before.gcLiveBytes,
		gcThreshold:      after.gcThreshold - before.gcThreshold,
		gcPollRetry:      after.gcPollRetry - before.gcPollRetry,
	}
}

func newPhaseSampler(sampleEvery int) *phaseSampler {
	freeKB := kos.FreeRAMKB()
	liveBytes := kos.GCLiveBytesRaw()
	liveObjects := gcLiveObjectCount()
	return &phaseSampler{
		sampleEvery:     sampleEvery,
		minFreeKB:       freeKB,
		peakLiveBytes:   liveBytes,
		peakLiveObjects: liveObjects,
	}
}

func (sampler *phaseSampler) maybe(index int) {
	if sampler == nil || sampler.sampleEvery <= 0 {
		return
	}
	if index%sampler.sampleEvery != 0 {
		return
	}
	kos.PollRuntimeGCRaw()
	sampler.sample()
}

func (sampler *phaseSampler) sample() {
	if sampler == nil {
		return
	}
	freeKB := kos.FreeRAMKB()
	liveBytes := kos.GCLiveBytesRaw()
	liveObjects := gcLiveObjectCount()
	if freeKB < sampler.minFreeKB {
		sampler.minFreeKB = freeKB
	}
	if liveBytes > sampler.peakLiveBytes {
		sampler.peakLiveBytes = liveBytes
	}
	if liveObjects > sampler.peakLiveObjects {
		sampler.peakLiveObjects = liveObjects
	}
}

func workloadPlainLoop(iterations int, sampler *phaseSampler) {
	value := iterations
	for value > 0 {
		value--
	}
	if sampler != nil {
		sampler.sample()
	}
	sinkInt = value
}

func workloadAllocSmall(iterations int, sampler *phaseSampler) {
	sum := 0
	var last *smallNode
	for i := 0; i < iterations; i++ {
		var head *smallNode
		for j := 0; j < 12; j++ {
			head = &smallNode{
				left:  i + j,
				right: i + j + 1,
				next:  head,
			}
			sum += head.right
		}
		last = head
		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sinkInt = sum
	sinkIface = last
}

func workloadMapInt(iterations int, sampler *phaseSampler) {
	sum := 0
	for i := 0; i < iterations; i++ {
		m := make(map[int]int, 32)
		for j := 0; j < 24; j++ {
			m[j] = i + j + 1
		}
		for j := 0; j < 24; j++ {
			sum += m[j]
			m[j] = sum
		}
		for j := 0; j < 8; j++ {
			delete(m, j)
		}
		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sinkInt = sum
}

func workloadMapString(iterations int, sampler *phaseSampler) {
	keys := make([]string, 24)
	for i := 0; i < len(keys); i++ {
		keys[i] = "key/" + strconv.Itoa(i)
	}
	sum := 0
	for i := 0; i < iterations; i++ {
		m := make(map[string]int, len(keys))
		for j := 0; j < len(keys); j++ {
			key := keys[(i+j)%len(keys)]
			m[key] = i + j + 1
		}
		for j := 0; j < len(keys); j++ {
			key := keys[(i+j*3)%len(keys)]
			sum += m[key]
			m[key] = sum
		}
		for j := 0; j < 8; j++ {
			delete(m, keys[(i+j)%len(keys)])
		}
		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sinkInt = sum
	sinkString = keys[sum%len(keys)]
}

func workloadSliceGrow(iterations int, sampler *phaseSampler) {
	sum := 0
	lastByte := byte(0)
	for i := 0; i < iterations; i++ {
		ints := make([]int, 0, 4)
		for j := 0; j < 96; j++ {
			ints = append(ints, i+j)
			sum += ints[len(ints)-1] & 7
		}
		buf := make([]byte, 0, 8)
		for j := 0; j < 256; j++ {
			buf = append(buf, byte(i+j))
		}
		if len(buf) > 0 {
			lastByte = buf[len(buf)-1]
		}
		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sum ^= int(lastByte)
	sinkInt = sum
}

func workloadStringChurn(iterations int, sampler *phaseSampler) {
	words := [...]string{"alpha", "beta", "gamma", "delta", "omega", "lambda", "sigma", "theta"}
	sum := 0
	last := ""
	for i := 0; i < iterations; i++ {
		var builder strings.Builder
		for j := 0; j < 8; j++ {
			builder.WriteString(words[(i+j)&7])
			builder.WriteByte('-')
		}
		text := builder.String()
		parts := strings.Split(text, "-")
		joined := strings.Join(parts, "/")
		upper := strings.ToUpper(joined)
		last = strings.ReplaceAll(upper, "A", "@")
		sum += len(last)
		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sinkString = last
	sinkInt = sum
}

func workloadInterfaceBox(iterations int, sampler *phaseSampler) {
	sum := 0
	var value interface{}
	for i := 0; i < iterations; i++ {
		switch i & 3 {
		case 0:
			value = i
		case 1:
			value = "v/" + strconv.Itoa(i&63)
		case 2:
			value = float64(i) + 0.5
		default:
			value = pairKey{a: i, b: "k" + strconv.Itoa(i&31)}
		}

		switch current := value.(type) {
		case int:
			sum += current
		case string:
			sum += len(current)
		case float64:
			sum += int(current)
		case pairKey:
			sum += current.a + len(current.b)
		}

		if sampler != nil {
			sampler.maybe(i)
		}
	}
	sinkInt = sum
	sinkIface = value
}

func workloadMixedRuntime(iterations int, sampler *phaseSampler) {
	for i := 0; i < iterations; i++ {
		mixedIteration(i)
		if sampler != nil {
			sampler.maybe(i)
		}
	}
}

func mixedIteration(seed int) {
	var builder strings.Builder
	for i := 0; i < 8; i++ {
		builder.WriteString("chunk")
		builder.WriteByte(byte('a' + byte((seed+i)%26)))
	}
	built := builder.String()
	upper := strings.ToUpper(built)
	parts := strings.Split(upper, "A")
	joined := strings.Join(parts, "-")
	replaced := strings.ReplaceAll(joined, "B", "bb")

	m := make(map[string][]byte, 24)
	for i := 0; i < 24; i++ {
		key := "k" + strconv.Itoa(seed) + "/" + strconv.Itoa(i)
		size := 24 + (i % 32)
		buf := make([]byte, size)
		for j := 0; j < len(buf); j += 7 {
			buf[j] = byte(seed + i + j)
		}
		m[key] = buf
	}
	for i := 0; i < 8; i++ {
		delete(m, "k"+strconv.Itoa(seed)+"/"+strconv.Itoa(i))
	}

	m2 := make(map[int]string, 16)
	for i := 0; i < 16; i++ {
		m2[i] = replaced
	}

	mi := make(map[interface{}]int, 8)
	mi["alpha"] = seed
	mi[uint32(seed)] = seed * 2
	mi[float64(seed)+0.5] = seed * 3
	mi[pairKey{a: seed, b: built}] = seed * 4

	ints := make([]int, 96+(seed%32))
	for i := range ints {
		ints[i] = i + seed
	}

	big := make([]byte, 768+(seed%256))
	for i := 0; i < len(big); i += 64 {
		big[i] = byte(seed + i)
	}

	sinkInt ^= len(m) + len(m2) + len(mi) + len(ints) + len(big)
	if len(replaced) > 0 {
		sinkString = replaced[:1]
	}
	if len(big) > 0 {
		sinkInt ^= int(big[0])
	}
	sinkIface = pairKey{a: seed, b: built}
}

func findPhase(results []phaseResult, name string) *phaseResult {
	for i := range results {
		if results[i].name == name {
			return &results[i]
		}
	}
	return nil
}

func slowestPhase(results []phaseResult) *phaseResult {
	if len(results) == 0 {
		return nil
	}
	slowest := &results[0]
	for i := 1; i < len(results); i++ {
		if results[i].nsPerIter() > slowest.nsPerIter() {
			slowest = &results[i]
		}
	}
	return slowest
}

func (result phaseResult) nsPerIter() uint64 {
	if result.iterations <= 0 {
		return 0
	}
	return result.elapsedNS / uint64(result.iterations)
}

func (result phaseResult) sysDropKB() uint32 {
	if result.startFreeKB <= result.minFreeKB {
		return 0
	}
	return result.startFreeKB - result.minFreeKB
}

func (result phaseResult) sysRecoverKB() uint32 {
	if result.afterGCFreeKB <= result.minFreeKB {
		return 0
	}
	return result.afterGCFreeKB - result.minFreeKB
}

func nsToMilliseconds(value uint64) uint64 {
	return value / 1000000
}

func nsToMicroseconds(value uint64) uint64 {
	return value / 1000
}

func bytesToKB(value uint32) uint32 {
	return (value + 1023) / 1024
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func waitForExit(console kos.Console) {
	if !console.SupportsInput() {
		return
	}
	fmt.Println("")
	fmt.Println("Press any key to close.")
	console.Getch()
}
