package main

import (
	"kos"
	"runtime"
	"sync"
	"sync/atomic"
)

func check(ok bool, why string) {
	if !ok {
		kos.DebugString(why)
		panic(why)
	}
}

func main() {
	console, ok := kos.OpenConsole("OpenCode sync port test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_SYNC_START")
	runtime.GOMAXPROCS(2)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var once sync.Once
	var count, initialized int
	wg.Add(8)
	for i := 0; i < 8; i++ {
		go func() {
			defer wg.Done()
			once.Do(func() { initialized++; runtime.Gosched() })
			for j := 0; j < 2000; j++ {
				mu.Lock()
				count++
				if j%37 == 0 {
					runtime.Gosched()
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	check(count == 16000 && initialized == 1, "mutex, waitgroup and once")
	check(mu.TryLock() && !mu.TryLock(), "mutex TryLock")
	mu.Unlock()
	kos.DebugString("SYNC_MUTEX_OK")
	var rw sync.RWMutex
	var pair [2]int
	wg.Add(8)
	for i := 0; i < 8; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				if i%2 == 0 {
					rw.Lock()
					pair[0]++
					runtime.Gosched()
					pair[1]++
					rw.Unlock()
				} else {
					rw.RLock()
					check(pair[0] == pair[1], "RWMutex excludes incomplete writes")
					rw.RUnlock()
				}
			}
		}()
	}
	wg.Wait()
	check(pair[0] == 2000 && pair[1] == 2000, "RWMutex writes")
	check(rw.TryLock() && !rw.TryRLock(), "RWMutex TryLock")
	rw.Unlock()
	check(rw.TryRLock() && !rw.TryLock(), "RWMutex TryRLock")
	rw.RUnlock()
	kos.DebugString("SYNC_RW_MUTEX_OK")
	cond := sync.NewCond(&mu)
	ready, waiting := false, 0
	wg.Add(4)
	for i := 0; i < 4; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			waiting++
			for !ready {
				cond.Wait()
			}
			mu.Unlock()
		}()
	}
	for {
		mu.Lock()
		if waiting == 4 {
			break
		}
		mu.Unlock()
		runtime.Gosched()
	}
	ready = true
	cond.Broadcast()
	mu.Unlock()
	wg.Wait()
	kos.DebugString("SYNC_COND_OK")
	var m sync.Map
	wg.Add(8)
	for i := 0; i < 8; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := i*100 + j
				actual, loaded := m.LoadOrStore(key, key)
				check(!loaded && actual == key, "Map LoadOrStore")
				check(m.CompareAndSwap(key, key, key+1), "Map CompareAndSwap")
			}
		}()
	}
	wg.Wait()
	n := 0
	m.Range(func(key, value any) bool { check(value == key.(int)+1, "Map stored value"); n++; return true })
	check(n == 800, "Map concurrent contents")
	m.Clear()
	_, found := m.Load(0)
	check(!found, "Map Clear")
	var calls atomic.Int32
	onceValue := sync.OnceValue(func() int { calls.Add(1); return 42 })
	check(onceValue() == 42 && onceValue() == 42 && calls.Load() == 1, "generic OnceValue")
	panics := 0
	f := sync.OnceFunc(func() { panics++; panic("once panic") })
	for i := 0; i < 2; i++ {
		func() {
			defer func() { check(recover() == "once panic", "OnceFunc repeats original panic") }()
			f()
		}()
	}
	check(panics == 1, "panicking once function executes once")
	kos.DebugString("OPENCODE_SYNC_PASS")
	console.WriteString("OPENCODE_SYNC_PASS\n")
}
