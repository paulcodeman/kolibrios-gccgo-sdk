//go:build kolibrios

package main

import (
	_ "embed"
	"kos"
	"os"
	"unsafe"
)

//go:embed console-unifont.kbf
var nativeUnicodeFont []byte

// kolibriMain connects the original CLI to the native console. main remains
// the unchanged upstream entrypoint; the SDK startup selects this wrapper.
func kolibriMain() {
	console, ok := kos.OpenConsole("OpenCode")
	if !ok {
		kos.DebugString("OpenCode: cannot initialize CONSOLE.OBJ")
		os.Exit(1)
	}
	// The unchanged CLI emits Unicode and xterm controls. Prefer the upstream
	// terminal backend when available; older SDK consoles support UTF-8 only.
	if terminal := console.ExportTable().Lookup("con_enable_vterm"); terminal.Valid() {
		if kos.CallStdcall0Raw(uint32(terminal)) != 1 {
			kos.DebugString("OpenCode: cannot initialize terminal backend")
			os.Exit(1)
		}
		if font := console.ExportTable().Lookup("con_set_unicode_font"); font.Valid() {
			if kos.CallStdcall2Raw(uint32(font), uint32(uintptr(unsafe.Pointer(&nativeUnicodeFont[0]))), uint32(len(nativeUnicodeFont))) != 1 {
				kos.DebugString("OpenCode: cannot initialize Unicode font")
				os.Exit(1)
			}
		}
	} else if mode := console.ExportTable().Lookup("con_set_output_mode"); mode.Valid() {
		kos.CallStdcall1Raw(uint32(mode), 1)
	}
	// Keep completed command output visible until the user closes the window.
	defer console.Exit(false)
	if err := loadZenPublicModels(); err != nil {
		kos.DebugString("OpenCode Zen: " + err.Error())
		console.WriteString("OpenCode Zen: " + err.Error() + "\n")
		return
	}
	main()
}
