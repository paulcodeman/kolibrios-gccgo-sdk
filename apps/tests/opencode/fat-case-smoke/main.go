package main

import (
	"fmt"
	"kos"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	console, ok := kos.OpenConsole("Native FAT filename spelling")
	if !ok {
		panic("console")
	}
	defer console.Exit(false)
	entries, err := os.ReadDir("/hd0/1")
	if err != nil {
		panic(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Name() == "fixture.txt" {
			found = true
		}
	}
	if !found {
		for _, entry := range entries {
			fmt.Println(entry.Name())
		}
		panic("FAT NT lower-case spelling")
	}
	names := []string{"lower.go", "Mixed.Go", "UPPER.TXT", "camelCase.sh", "Привет.sh"}
	if err = os.Mkdir("/hd0/1/caseDir", 0700); err != nil {
		panic(err)
	}
	for _, name := range names {
		if err = os.WriteFile("/hd0/1/caseDir/"+name, []byte(name), 0600); err != nil {
			panic(err)
		}
	}
	entries, err = os.ReadDir("/hd0/1/caseDir")
	if err != nil {
		panic(err)
	}
	for _, name := range names {
		found = false
		for _, entry := range entries {
			if entry.Name() == name {
				found = true
			}
		}
		if !found {
			for _, entry := range entries {
				fmt.Println(entry.Name())
			}
			panic("native create LFN spelling")
		}
		content, err := os.ReadFile("/hd0/1/caseDir/" + strings.ToUpper(name))
		if err != nil || string(content) != name {
			panic("FAT case-insensitive lookup")
		}
	}
	if err = os.Rename("/hd0/1/caseDir/lower.go", "/hd0/1/caseDir/rename.go"); err != nil {
		panic(err)
	}
	matched, err := filepath.Glob("/hd0/1/caseDir/*.go")
	if err != nil || len(matched) != 1 || filepath.Base(matched[0]) != "rename.go" {
		fmt.Println(matched, err)
		panic("native case-preserved glob")
	}
	if err = os.RemoveAll("/hd0/1/caseDir"); err != nil {
		panic(err)
	}
	fmt.Println("FAT NT lower-case read, mixed/Unicode create, case-insensitive lookup, rename, glob and cleanup PASS")
	kos.DebugString("OPENCODE_FAT_CASE_NATIVE_PASS\n")
}
