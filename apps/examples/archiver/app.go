package main

import (
	"fmt"
	"os"
	"strings"
	"ui/elements"

	"kos"
	"ui"
)

const (
	archiverButtonExit    kos.ButtonID = 1
	archiverButtonRefresh kos.ButtonID = 2

	archiverWindowTitle  = "KolibriOS ARCHIVER Demo"
	archiverWindowX      = 226
	archiverWindowY      = 146
	archiverWindowWidth  = 920
	archiverWindowHeight = 320
)

var archiverCompressedPayload = []byte{
	0xcb, 0xce, 0xcf, 0xc9, 0x4c, 0x2a, 0xca, 0x54, 0x48, 0xcf, 0x57, 0x48, 0x2c,
	0x4a, 0xce, 0xc8, 0x2c, 0x4b, 0x2d, 0xe2, 0x02, 0x00,
}

const archiverExpectedText = "kolibri go archiver\n"

type App struct {
	summary    string
	dataLine   string
	resultLine string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(archiverButtonRefresh, "Refresh", 28, 264)
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
	case archiverButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case archiverButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(archiverButtonExit, "Exit", 164, 264)
	exit.SetWidth(92)

	kos.BeginRedraw()
	kos.OpenWindow(archiverWindowX, archiverWindowY, archiverWindowWidth, archiverWindowHeight, archiverWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "ARCHIVER.OBJ deflate_unpack through the typed kos wrapper")
	kos.DrawText(28, 92, ui.Aqua, app.dataLine)
	kos.DrawText(28, 114, ui.Lime, app.resultLine)
	kos.DrawText(28, 136, ui.Yellow, app.infoLine)
	kos.DrawText(28, 162, ui.Black, "The payload is a raw DEFLATE byte slice embedded in the sample and unpacked with archiver.deflate_unpack")
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	var lib kos.Archiver
	var ok bool
	var unpacked []byte

	lib, ok = kos.LoadArchiver()
	if !ok {
		app.fail("archiver.obj unavailable", "Info: failed to load "+kos.ArchiverDLLPath)
		return
	}

	unpacked, ok = lib.DeflateUnpack(archiverCompressedPayload)
	if !ok {
		app.fail("deflate unpack failed", "Info: the embedded raw-deflate payload did not unpack")
		return
	}
	if string(unpacked) != archiverExpectedText {
		app.fail("payload mismatch", "Info: unpacked text did not match the embedded reference")
		return
	}

	app.ok = true
	app.summary = "archiver probe ok / deflate_unpack stable"
	app.dataLine = fmt.Sprintf("Data: packed=%d bytes / unpacked=%d bytes", len(archiverCompressedPayload), len(unpacked))
	app.resultLine = fmt.Sprintf("Result: %q", strings.TrimSpace(string(unpacked)))
	app.infoLine = fmt.Sprintf("Info: %s / version %d / table 0x%x", kos.ArchiverDLLPath, lib.Version(), uint32(lib.ExportTable()))
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "archiver probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
