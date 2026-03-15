package main

import (
	"os"
	"path/filepath"
	"ui/elements"

	"kos"
	"ui"
)

const (
	filepathButtonExit    kos.ButtonID = 1
	filepathButtonRefresh kos.ButtonID = 2

	filepathWindowTitle  = "KolibriOS Filepath Demo"
	filepathWindowX      = 246
	filepathWindowY      = 162
	filepathWindowWidth  = 780
	filepathWindowHeight = 298

	filepathRawProbe = "\\sys\\.\\skins\\..\\default.skn"
)

type App struct {
	summary    string
	cleanLine  string
	joinLine   string
	splitLine  string
	absLine    string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(filepathButtonRefresh, "Refresh", 28, 242)
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
	case filepathButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case filepathButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(filepathButtonExit, "Exit", 170, 242)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(filepathWindowX, filepathWindowY, filepathWindowWidth, filepathWindowHeight, filepathWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary path/filepath package")
	kos.DrawText(28, 92, ui.Aqua, app.cleanLine)
	kos.DrawText(28, 114, ui.Lime, app.joinLine)
	kos.DrawText(28, 136, ui.Yellow, app.splitLine)
	kos.DrawText(28, 158, ui.White, app.absLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	const expectedPath = "/sys/default.skn"

	cleaned := filepath.Clean(filepathRawProbe)
	joined := filepath.Join("/sys", ".", "skins", "..", "default.skn")
	dir, file := filepath.Split(cleaned)
	base := filepath.Base(cleaned)
	ext := filepath.Ext(cleaned)
	isAbs := filepath.IsAbs(cleaned)
	slashed := filepath.ToSlash(filepathRawProbe)
	restored := filepath.FromSlash(expectedPath)
	volume := filepath.VolumeName(cleaned)
	relativeAbs, err := filepath.Abs("default.skn")
	if err != nil {
		app.fail("abs failed")
		app.infoLine = "Info: " + err.Error()
		return
	}

	app.cleanLine = "Clean/ToSlash: " + filepathRawProbe + " -> " + cleaned + " / slash " + slashed
	app.joinLine = "Join: " + joined + " / sep " + string([]byte{filepath.Separator}) + " / list " + string([]byte{filepath.ListSeparator})
	app.splitLine = "Split: dir " + dir + " / file " + file + " / base " + base + " / ext " + ext + " / abs " + formatBool(isAbs) + " / volume " + volume
	app.absLine = "Abs: default.skn -> " + relativeAbs + " / fromslash " + restored

	if cleaned != expectedPath {
		app.fail("clean mismatch")
		return
	}
	if joined != expectedPath {
		app.fail("join mismatch")
		return
	}
	if dir != "/sys/" || file != "default.skn" || base != "default.skn" || ext != ".skn" || !isAbs {
		app.fail("split/base/ext mismatch")
		return
	}
	if filepath.Clean(slashed) != expectedPath || restored != expectedPath || volume != "" {
		app.fail("slash or volume mismatch")
		return
	}
	if !filepath.IsAbs(relativeAbs) || filepath.Base(relativeAbs) != "default.skn" {
		app.fail("abs semantics mismatch")
		return
	}

	info, err := os.Stat(cleaned)
	if err != nil {
		app.ok = false
		app.summary = "filepath probe failed / file info unavailable"
		app.infoLine = "Info: " + cleaned + " / " + err.Error()
		return
	}
	rawInfo, ok := info.Sys().(kos.FileInfo)
	if !ok {
		app.fail("stat sys payload mismatch")
		return
	}

	app.ok = true
	app.summary = "filepath probe ok / ordinary import path/filepath resolved"
	app.infoLine = "Info: size " + formatHex64(uint64(info.Size())) + " bytes / attrs " + formatHex32(uint32(rawInfo.Attributes))
}

func (app *App) fail(detail string) {
	app.ok = false
	app.summary = "filepath probe failed / " + detail
	app.infoLine = "Info: unavailable"
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func main() {
	app := NewApp()
	app.Run()
}
