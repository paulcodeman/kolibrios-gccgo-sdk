package main

import (
	"kos"
	_ "reflect"
	"runtime"
	"time"
)

type object struct {
	value   int
	next    *object
	padding [64]byte
}

func (o *object) Value() int { return o.value }

type valued interface{ Value() int }

var events = make(chan int, 20)
var resurrected *object

func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}

//go:noinline
func install() {
	a := &object{value: 1}
	runtime.SetFinalizer(a, func(o *object) { events <- o.value })
	b := &object{value: 2}
	runtime.SetFinalizer(b, func(o any) { events <- o.(*object).value })
	c := &object{value: 3}
	runtime.SetFinalizer(c, func(o valued) { events <- o.Value() })
	d := &object{value: 4}
	runtime.SetFinalizer(d, func(o *object) { events <- o.value })
	runtime.SetFinalizer(d, nil)
	e := &object{value: 5}
	runtime.SetFinalizer(e, func(o *object) int { events <- o.value; return 10 })
	f := &object{value: 6}
	runtime.SetFinalizer(f, func(o *object) { resurrected = o; events <- o.value })
	g := &object{value: 9}
	captured := &object{value: 42}
	runtime.SetFinalizer(g, func(o *object) { check(captured.value == 42, "closure capture retained"); events <- o.value })
}

//go:noinline
func keepAlive() {
	o := &object{value: 10}
	runtime.SetFinalizer(o, func(o *object) { events <- o.value })
	collect()
	select {
	case <-events:
		panic("live object finalized")
	default:
	}
	runtime.KeepAlive(o)
}

//go:noinline
func dependencies() {
	b := &object{value: 8}
	a := &object{value: 7, next: b}
	runtime.SetFinalizer(a, func(o *object) { check(o.next.value == 8, "dependency alive"); events <- o.value })
	runtime.SetFinalizer(b, func(o *object) { events <- o.value })
}

//go:noinline
func concurrent() {
	for i := 20; i < 26; i++ {
		o := &object{value: i}
		runtime.SetFinalizer(o, func(o *object) { events <- o.value })
	}
}

func collect() {
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
}

func wait(count int, seen map[int]bool) {
	for i := 0; i < 100 && len(seen) < count; i++ {
		collect()
		for {
			select {
			case value := <-events:
				check(!seen[value], "callback repeated")
				check(value != 4, "removed callback")
				if value == 8 {
					check(seen[7], "dependency order")
				}
				seen[value] = true
			default:
				goto drained
			}
		}
	drained:
	}
	check(len(seen) == count, "callbacks did not execute")
}

func main() {
	console, ok := kos.OpenConsole("OpenCode finalizer port test")
	check(ok, "console")
	defer console.Close()
	kos.DebugString("OPENCODE_FINALIZER_START")
	install()
	seen := make(map[int]bool)
	wait(6, seen)
	check(resurrected != nil && resurrected.value == 6, "resurrection")
	for i := 0; i < 3; i++ {
		collect()
	}
	check(resurrected.value == 6, "resurrection survived GC")
	dependencies()
	wait(8, seen)
	for i := 0; i < 3; i++ {
		collect()
	}
	select {
	case <-events:
		panic("extra callback")
	default:
	}
	keepAlive()
	wait(9, seen)
	runtime.GOMAXPROCS(2)
	done := make(chan bool)
	go func() { concurrent(); done <- true }()
	<-done
	wait(15, seen)
	kos.DebugString("OPENCODE_FINALIZER_PASS")
	console.WriteString("OPENCODE_FINALIZER_PASS\n")
}
