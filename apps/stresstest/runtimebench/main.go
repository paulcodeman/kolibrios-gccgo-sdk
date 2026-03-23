package main

import (
	"fmt"
	"os"
	"strconv"

	"kos"
)

var (
	sinkInt    int
	sinkString string
	sinkIface  interface{}
)

type benchResult struct {
	name            string
	iterations      int
	elapsedNS       uint64
	gcAllocBytes    uint32
	gcCollections   uint32
	heapAllocBytes  uint32
}

func main() {
	iterations := 100000
	if len(os.Args) > 1 {
		if parsed, err := strconv.Atoi(os.Args[1]); err == nil && parsed > 0 {
			iterations = parsed
		}
	}

	fmt.Printf("runtime bench, iterations=%d\n", iterations)
	fmt.Println("columns: total_ns ns_per_iter gc_alloc_delta gc_collections heap_alloc_delta")
	runAndPrint("plain-loop", iterations, benchPlainLoop)
	runAndPrint("map-int", iterations, benchMapInt)
	runAndPrint("map-string", iterations, benchMapString)
	runAndPrint("alloc-pair", iterations, benchAllocPair)
	runAndPrint("iface-box", iterations, benchInterfaceBox)
	_ = sinkInt
	_ = sinkString
	_ = sinkIface
}

func runAndPrint(name string, iterations int, fn func(int)) {
	result := measure(name, iterations, fn)
	nsPerIter := float64(result.elapsedNS) / float64(result.iterations)
	fmt.Printf("%-12s %10d %10.2f %14d %14d %15d\n",
		result.name,
		result.elapsedNS,
		nsPerIter,
		result.gcAllocBytes,
		result.gcCollections,
		result.heapAllocBytes)
}

func measure(name string, iterations int, fn func(int)) benchResult {
	beforeAlloc := kos.GCAllocBytesRaw()
	beforeCollections := kos.GCCollectionCountRaw()
	beforeHeap := kos.HeapAllocBytesRaw()
	start := kos.UptimeNanoseconds()
	fn(iterations)
	end := kos.UptimeNanoseconds()
	afterAlloc := kos.GCAllocBytesRaw()
	afterCollections := kos.GCCollectionCountRaw()
	afterHeap := kos.HeapAllocBytesRaw()
	return benchResult{
		name:           name,
		iterations:     iterations,
		elapsedNS:      end - start,
		gcAllocBytes:   afterAlloc - beforeAlloc,
		gcCollections:  afterCollections - beforeCollections,
		heapAllocBytes: afterHeap - beforeHeap,
	}
}

func benchPlainLoop(iterations int) {
	value := iterations
	for value > 0 {
		value--
	}
	sinkInt = value
}

func benchMapInt(iterations int) {
	m := make(map[int]int, 64)
	for i := 0; i < 64; i++ {
		m[i] = i + 1
	}
	sum := 0
	for i := 0; i < iterations; i++ {
		key := i & 63
		sum += m[key]
		m[key] = sum
	}
	sinkInt = sum
}

func benchMapString(iterations int) {
	keys := make([]string, 64)
	m := make(map[string]int, len(keys))
	for i := 0; i < len(keys); i++ {
		keys[i] = "k" + strconv.Itoa(i)
		m[keys[i]] = i + 1
	}
	sum := 0
	for i := 0; i < iterations; i++ {
		key := keys[i&63]
		sum += m[key]
		m[key] = sum
	}
	sinkString = keys[sum&63]
	sinkInt = sum
}

func benchAllocPair(iterations int) {
	type pair struct {
		left  int
		right int
		next  *pair
	}

	var head *pair
	sum := 0
	for i := 0; i < iterations; i++ {
		node := &pair{left: i, right: i + 1, next: head}
		sum += node.right
		head = node
	}
	sinkInt = sum
	sinkIface = head
}

func benchInterfaceBox(iterations int) {
	sum := 0
	var value interface{}
	for i := 0; i < iterations; i++ {
		value = i
		if current, ok := value.(int); ok {
			sum += current
		}
	}
	sinkInt = sum
	sinkIface = value
}
