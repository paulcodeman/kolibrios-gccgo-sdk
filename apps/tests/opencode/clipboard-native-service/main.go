package main

import (
	"kos"
	"os"
	"time"

	"github.com/atotto/clipboard"
)

func main() {
	const text = "Привет 😀"
	if err := clipboard.WriteAll(text); err != nil {
		panic(err)
	}
	read, err := clipboard.ReadAll()
	if err != nil || read != text {
		panic("native UTF-8 clipboard round trip")
	}
	if err = os.WriteFile("/hd0/1/SERVICE.READY", []byte("ready"), 0600); err != nil {
		panic(err)
	}
	kos.DebugString("OPENCODE_CLIPBOARD_UTF8_READY\n")
	for {
		time.Sleep(time.Second)
	}
}
