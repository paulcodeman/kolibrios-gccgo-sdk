package main

import (
	"fmt"

	"kos"
)

const demoTitle = "KolibriOS Goroutines Demo"
const exitKey = 27

func main() {
	console, ok := kos.OpenConsole(demoTitle)
	if !ok {
		kos.DebugString("goroutines demo: failed to open /sys/lib/console.obj")
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(demoTitle)
	}

	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		for i := 0; i < 3; i++ {
			ch1 <- i
		}
		close(ch1)
	}()

	go func() {
		for i := 10; i < 13; i++ {
			ch2 <- i
		}
		close(ch2)
	}()

	closed := 0
	for closed < 2 {
		select {
		case v, ok := <-ch1:
			if !ok {
				ch1 = nil
				closed++
				fmt.Println("ch1 closed")
				continue
			}
			fmt.Printf("ch1 -> %d\n", v)
		case v, ok := <-ch2:
			if !ok {
				ch2 = nil
				closed++
				fmt.Println("ch2 closed")
				continue
			}
			fmt.Printf("ch2 -> %d\n", v)
		}
	}

	fmt.Println("done")
	if console.SupportsInput() {
		fmt.Println("Press Esc to exit.")
		for {
			if console.Getch() == exitKey {
				return
			}
		}
	}
}
