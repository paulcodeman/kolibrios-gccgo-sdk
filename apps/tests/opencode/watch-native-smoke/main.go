package main

import (
	"errors"
	"fmt"
	"kos"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	console, ok := kos.OpenConsole("Upstream native filesystem watcher")
	if !ok {
		panic("console")
	}
	defer console.Exit(false)
	const directory = "/hd0/1/watch"
	const filename = directory + "/Привет.txt"
	const existing = directory + "/existing.txt"
	if err := os.Mkdir(directory, 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(existing, []byte("keep this existing file"), 0600); err != nil {
		panic(err)
	}
	watcher, err := fsnotify.NewBufferedWatcher(16)
	if err != nil {
		panic(err)
	}
	defer watcher.Close()
	if err = watcher.Add("/hd0/1/missing"); err == nil {
		panic("missing watch path accepted")
	}
	if err = watcher.Add(directory); err != nil {
		panic(err)
	}
	if err = watcher.Add(directory); err != nil {
		panic(err)
	}
	if paths := watcher.WatchList(); len(paths) != 1 || paths[0] != directory {
		panic("watch list")
	}
	await := func(name string, op fsnotify.Op) {
		deadline := time.After(8 * time.Second)
		for {
			select {
			case event := <-watcher.Events:
				if event.Name == existing && event.Has(fsnotify.Remove) {
					panic("concurrent watch registration falsely removed an existing file")
				}
				if event.Name == name && event.Has(op) {
					return
				}
			case err := <-watcher.Errors:
				panic(err)
			case <-deadline:
				panic("native filesystem event timeout: " + name + " " + op.String())
			}
		}
	}
	if err = os.WriteFile(filename, []byte("AAAA"), 0600); err != nil {
		panic(err)
	}
	await(filename, fsnotify.Create)
	// Preserve the old timestamp deliberately: metadata-only polling loses it.
	info, err := os.Stat(filename)
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(filename, []byte("BBBB"), 0600); err != nil {
		panic(err)
	}
	if err = os.Chtimes(filename, info.ModTime(), info.ModTime()); err != nil {
		panic(err)
	}
	await(filename, fsnotify.Write)
	if err = os.Remove(filename); err != nil {
		panic(err)
	}
	await(filename, fsnotify.Remove)
	if err = watcher.Remove(directory); err != nil {
		panic(err)
	}
	if err = watcher.Remove(directory); !errors.Is(err, fsnotify.ErrNonExistentWatch) {
		panic("missing watch remove")
	}
	if err = watcher.Close(); err != nil {
		panic(err)
	}
	if err = watcher.Close(); err != nil {
		panic(err)
	}
	if err = watcher.Add(directory); !errors.Is(err, fsnotify.ErrClosed) {
		panic("closed watcher Add")
	}
	if _, open := <-watcher.Events; open {
		panic("closed event channel")
	}
	fmt.Println("Concurrent registration, existing file, Unicode create, same-size/same-timestamp write, remove, lifecycle PASS")
	kos.DebugString("OPENCODE_FSNOTIFY_NATIVE_PASS\n")
}
