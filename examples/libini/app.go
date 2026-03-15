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
	libiniButtonExit    kos.ButtonID = 1
	libiniButtonRefresh kos.ButtonID = 2

	libiniWindowTitle  = "KolibriOS LIBINI Demo"
	libiniWindowX      = 228
	libiniWindowY      = 148
	libiniWindowWidth  = 860
	libiniWindowHeight = 338

	libiniProbePath = "/FD/1/GOLIBINI.INI"
)

type App struct {
	summary    string
	valueLine  string
	fileLine   string
	dllLine    string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(libiniButtonRefresh, "Refresh", 28, 286)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.summary = "libini window ready / press Refresh to run the probe"
	app.valueLine = "Values: probe not started yet"
	app.fileLine = "File: probe not started yet"
	app.dllLine = "DLL: " + kos.INIDLLPath
	app.infoLine = "Info: startup stays non-blocking; the shared kos LIBINI probe now starts only when you press Refresh"
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
	case libiniButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case libiniButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(libiniButtonExit, "Exit", 170, 286)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(libiniWindowX, libiniWindowY, libiniWindowWidth, libiniWindowHeight, libiniWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample uses the shared kos LIBINI wrapper: kos.LoadINI()")
	kos.DrawText(28, 92, ui.Aqua, app.valueLine)
	kos.DrawText(28, 114, ui.Lime, app.fileLine)
	kos.DrawText(28, 136, ui.Yellow, app.dllLine)
	kos.DrawText(28, 158, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	deleteStatus := kos.DeletePath(libiniProbePath)
	if deleteStatus != kos.FileSystemOK && deleteStatus != kos.FileSystemNotFound {
		app.fail("probe cleanup failed", "Info: "+formatFileSystemStatus(deleteStatus))
		return
	}

	ini, ok := kos.LoadINI()
	if !ok {
		app.fail("libini.obj unavailable", "Info: failed to load "+kos.INIDLLPath)
		return
	}
	if !ini.Ready() {
		app.fail("libini init failed", "Info: lib_init returned an error for "+kos.INIDLLPath)
		return
	}
	if !ini.SetString(libiniProbePath, "demo", "name", "go-libini") {
		app.fail("SetString failed", "Info: unable to write a string key into "+libiniProbePath)
		return
	}
	if !ini.SetInt(libiniProbePath, "demo", "count", 42) {
		app.fail("SetInt failed", "Info: unable to write an integer key into "+libiniProbePath)
		return
	}

	name, found := ini.GetString(libiniProbePath, "demo", "name", "fallback")
	missing, missingFound := ini.GetString(libiniProbePath, "demo", "missing", "fallback")
	count := ini.GetInt(libiniProbePath, "demo", "count", -1)
	missingCount := ini.GetInt(libiniProbePath, "demo", "missing_count", -7)
	if !found || name != "go-libini" || missingFound || missing != "fallback" || count != 42 || missingCount != -7 {
		app.fail("LIBINI contract mismatch", "Info: read-back or default semantics differ from the upstream wrapper contract")
		return
	}

	data, status := kos.ReadAllFile(libiniProbePath)
	if status != kos.FileSystemOK && status != kos.FileSystemEOF {
		app.fail("ReadAllFile failed", "Info: "+formatFileSystemStatus(status))
		return
	}

	app.ok = true
	app.summary = "libini probe ok / shared wrapper resolved"
	app.valueLine = fmt.Sprintf("Values: name=%s / count=%d / missing=%s / found=%t", name, count, missing, found)
	app.fileLine = fmt.Sprintf("File: %s / %d bytes / %s", libiniProbePath, len(data), compactINIText(string(data)))
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x", kos.INIDLLPath, uint32(ini.ExportTable()))
	app.infoLine = "Info: refresh rewrites a small INI file on /FD/1 and validates shared kos LIBINI read/write semantics"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "libini probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func compactINIText(value string) string {
	text := strings.ReplaceAll(value, "\r\n", " / ")
	text = strings.ReplaceAll(text, "\n", " / ")
	if len(text) > 88 {
		return text[:88] + "..."
	}

	return text
}

func formatFileSystemStatus(status kos.FileSystemStatus) string {
	switch status {
	case kos.FileSystemOK:
		return "ok"
	case kos.FileSystemEOF:
		return "eof"
	case kos.FileSystemNotFound:
		return "not found"
	default:
		return fmt.Sprintf("status 0x%x", uint32(status))
	}
}
