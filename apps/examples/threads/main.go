package main

import (
	"fmt"
	"runtime"

	"kos"
)

const demoTitle = "KolibriOS Threads Demo"
const exitKey = 27

func main() {
	console, ok := kos.OpenConsole(demoTitle)
	if !ok {
		kos.DebugString("threads demo: failed to open /sys/lib/console.obj")
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(demoTitle)
	}

	runtime.GOMAXPROCS(3)
	threads := runtime.GOMAXPROCS(0)
	fmt.Printf("runtime threads: %d\n", threads)
	if slot, ok := kos.CurrentThreadSlotIndex(); ok {
		fmt.Printf("main slot: %d\n", slot)
	}

	samples := make(chan int, 512)
	done := make(chan struct{}, 1)
	workers := threads * 4
	if workers < 3 {
		workers = 3
	}
	for w := 0; w < workers; w++ {
		go func(id int) {
			seen := make(map[int]bool)
			for i := 0; i < 200000; i++ {
				if (i % 512) == 0 {
					if slot, ok := kos.CurrentThreadSlotIndex(); ok {
						samples <- slot
						seen[slot] = true
					}
					runtime.Gosched()
					if (i % 4096) == 0 {
						kos.Sleep(1)
					}
				}
			}
			done <- struct{}{}
		}(w)
	}

	counts := make(map[int]int)
	distinct := make(map[int]bool)
	completed := 0
	for completed < workers {
		select {
		case slot := <-samples:
			distinct[slot] = true
			counts[slot]++
		case <-done:
			completed++
		}
	}
	if len(distinct) > 0 {
		fmt.Printf("observed slots: ")
		first := true
		for slot := range distinct {
			if !first {
				fmt.Printf(", ")
			}
			first = false
			fmt.Printf("%d", slot)
		}
		fmt.Println()
		fmt.Println("samples per slot:")
		for slot, count := range counts {
			fmt.Printf("  slot %d -> %d\n", slot, count)
		}
	} else {
		fmt.Println("observed slots: none")
	}

	fmt.Println("done")
	if console.SupportsInputFull() {
		fmt.Println("Press Esc to exit.")
		for {
			key := console.Getch2()
			if key == 0 {
				kos.Sleep(1)
				continue
			}
			hi := byte(key >> 8)
			lo := byte(key & 0xff)
			if hi == exitKey || lo == exitKey || lo == 1 {
				return
			}
		}
	}
	if console.SupportsInput() {
		fmt.Println("Press Esc to exit.")
		for {
			key := console.Getch()
			if key == 0 {
				kos.Sleep(1)
				continue
			}
			if key == exitKey || key == 1 || (key&0xff) == exitKey || (key&0xff) == 1 {
				return
			}
		}
	}
}
