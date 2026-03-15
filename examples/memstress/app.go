package main

import (
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	memStressButtonExit kos.ButtonID = 1
	memStressButtonRun  kos.ButtonID = 2

	memStressWindowTitle  = "KolibriOS GC Stress"
	memStressWindowX      = 240
	memStressWindowY      = 160
	memStressWindowWidth  = 760
	memStressWindowHeight = 320
)

type App struct {
	summary       string
	statusOK      bool
	infoLine      string
	startLine     string
	minLine       string
	afterPollLine string
	afterGCLine   string
	deltaLine     string
	liveLine      string
	liveBytesLine string
	runBtn        *ui.Element
}

func NewApp() App {
	runBtn := elements.ButtonAt(memStressButtonRun, "Run", 28, 272)
	runBtn.SetWidth(96)

	app := App{
		runBtn:  runBtn,
		summary: "ready",
	}
	app.refresh(nil)
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
	case memStressButtonRun:
		app.summary = "running..."
		app.statusOK = true
		app.Redraw()
		result := runStress(memStressIterations)
		app.refresh(&result)
		app.Redraw()
	case memStressButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(memStressButtonExit, "Exit", 140, 272)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(memStressWindowX, memStressWindowY, memStressWindowWidth, memStressWindowHeight, memStressWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, app.infoLine)
	kos.DrawText(28, 92, ui.Aqua, app.startLine)
	kos.DrawText(28, 112, ui.White, app.minLine)
	kos.DrawText(28, 132, ui.Yellow, app.afterPollLine)
	kos.DrawText(28, 152, ui.Lime, app.afterGCLine)
	kos.DrawText(28, 172, ui.Silver, app.deltaLine)
	kos.DrawText(28, 192, ui.Navy, app.liveLine)
	kos.DrawText(28, 212, ui.Aqua, app.liveBytesLine)
	app.runBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refresh(result *StressResult) {
	if result == nil {
		app.infoLine = "Press Run to allocate and free lots of memory (maps, slices, strings)."
		app.startLine = "Start KB: -"
		app.minLine = "Min KB: -"
		app.afterPollLine = "After poll KB: -"
		app.afterGCLine = "After GC KB: -"
		app.deltaLine = "Delta KB: -"
		app.liveLine = "Live objects: -"
		app.liveBytesLine = "Live bytes: -"
		app.statusOK = true
		return
	}

	app.infoLine = "Iterations: " + formatInt(result.Iterations) + " / warmup: " + formatInt(memStressWarmupIterations)
	app.startLine = "Start KB: " + formatUint32(result.StartKB)
	app.minLine = "Min KB: " + formatUint32(result.MinKB)
	app.afterPollLine = "After poll KB: " + formatUint32(result.AfterPollKB)
	app.afterGCLine = "After GC KB: " + formatUint32(result.AfterGCKB)
	app.deltaLine = "Delta KB: " + formatSignedInt(int(result.AfterGCKB)-int(result.StartKB))
	app.liveLine = "Live objects: " + formatUint32(result.LiveStart) + " -> " + formatUint32(result.LiveEnd)
	app.liveBytesLine = "Live bytes: " + formatUint32(result.LiveBytesStart) + " -> " + formatUint32(result.LiveBytesEnd)
	app.statusOK = result.LiveBytesEnd <= result.LiveBytesStart+65536
	if app.statusOK {
		app.summary = "gc stress done (live bytes within 64 KB)"
		return
	}
	app.summary = "gc stress done (live bytes grew > 64 KB)"
}

func (app *App) summaryColor() kos.Color {
	if app.statusOK {
		return ui.Lime
	}

	return ui.Red
}
