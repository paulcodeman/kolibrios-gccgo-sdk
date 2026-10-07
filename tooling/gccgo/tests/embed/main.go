package main

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed resources
var resources embed.FS

//go:embed "resources/space file.txt"
var text string

//go:embed resources/binary.bin
var binary []byte

//go:embed all:resources
var allResources embed.FS

func identity[T any](value T) T { return value }

func main() {
	if identity(text) != "quoted resource\n" || string(binary) != "a\x00b\xff" {
		panic("embedded scalar or binary contents changed")
	}
	files, err := fs.ReadDir(resources, "resources")
	if err != nil || len(files) != 3 {
		panic("embedded directory traversal failed")
	}
	if _, err := resources.ReadFile("resources/.hidden"); err == nil {
		panic("default directory pattern embedded a hidden file")
	}
	if data, err := allResources.ReadFile("resources/.hidden"); err != nil || string(data) != "hidden\n" {
		panic("all: pattern did not include hidden files")
	}
	if data, err := fs.ReadFile(resources, "resources/sub/a.txt"); err != nil || string(data) != "nested\n" {
		panic("embedded nested file read failed")
	}
	fmt.Println("PASS: original-resource embed.FS, string, binary and generics")
}
