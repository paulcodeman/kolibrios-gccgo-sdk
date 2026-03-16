package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	iconvButtonExit    kos.ButtonID = 1
	iconvButtonRefresh kos.ButtonID = 2

	iconvWindowTitle  = "KolibriOS ICONV Demo"
	iconvWindowX      = 226
	iconvWindowY      = 146
	iconvWindowWidth  = 900
	iconvWindowHeight = 320
)

type App struct {
	summary    string
	sourceLine string
	roundLine  string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(iconvButtonRefresh, "Refresh", 28, 264)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.refreshProbe()
	return app
}

func (app *App) Run() {
	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case iconvButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case iconvButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(iconvButtonExit, "Exit", 164, 264)
	exit.SetWidth(92)

	kos.BeginRedraw()
	kos.OpenWindow(iconvWindowX, iconvWindowY, iconvWindowWidth, iconvWindowHeight, iconvWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "ICONV.OBJ UTF-8 <-> CP866 roundtrip through the typed kos wrapper")
	kos.DrawText(28, 92, ui.Aqua, app.sourceLine)
	kos.DrawText(28, 114, ui.Lime, app.roundLine)
	kos.DrawText(28, 136, ui.Yellow, app.infoLine)
	kos.DrawText(28, 162, ui.Black, "The sample keeps the library contract explicit: open(from,to), caller-owned input/output buffers, roundtrip validation")
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	var lib kos.Iconv
	var ok bool
	var source string
	var cp866 string
	var roundtrip string
	var status int32

	lib, ok = kos.LoadIconv()
	if !ok {
		app.fail("iconv.obj unavailable", "Info: failed to load "+kos.IconvDLLPath)
		return
	}

	source = sampleUTF8Text()
	cp866, status, ok = lib.ConvertString(kos.IconvCharsetUTF8, kos.IconvCharsetCP866, source)
	if !ok {
		app.fail("utf8->cp866 failed", "Info: status="+fmt.Sprintf("%d", status))
		return
	}
	roundtrip, status, ok = lib.ConvertString(kos.IconvCharsetCP866, kos.IconvCharsetUTF8, cp866)
	if !ok {
		app.fail("cp866->utf8 failed", "Info: status="+fmt.Sprintf("%d", status))
		return
	}
	if roundtrip != source {
		app.fail("roundtrip mismatch", "Info: decoded text did not match the UTF-8 source")
		return
	}

	app.ok = true
	app.summary = "iconv probe ok / utf8 cp866 roundtrip stable"
	app.sourceLine = fmt.Sprintf("Source: %q / utf8 bytes=%d / cp866 bytes=%d", source, len(source), len(cp866))
	app.roundLine = fmt.Sprintf("Roundtrip: %q", roundtrip)
	app.infoLine = fmt.Sprintf("Info: %s / version 0x%x / table 0x%x", kos.IconvDLLPath, lib.Version(), uint32(lib.ExportTable()))
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "iconv probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func sampleUTF8Text() string {
	return string([]byte{0xD0, 0x9F, 0xD1, 0x80, 0xD0, 0xB8, 0xD0, 0xB2, 0xD0, 0xB5, 0xD1, 0x82})
}
