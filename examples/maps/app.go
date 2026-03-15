package main

import (
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	mapsButtonExit  kos.ButtonID = 1
	mapsButtonRerun kos.ButtonID = 2

	mapsWindowTitle  = "KolibriOS Map Test"
	mapsWindowX      = 240
	mapsWindowY      = 160
	mapsWindowWidth  = 760
	mapsWindowHeight = 320
)

type checkLine struct {
	ok   bool
	text string
}

type App struct {
	summary string
	checks  []checkLine
	rerun   *ui.Element
}

func NewApp() App {
	rerun := elements.ButtonAt(mapsButtonRerun, "Rerun", 28, 272)
	rerun.SetWidth(112)

	app := App{
		rerun: rerun,
	}
	app.runChecks()
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
	case mapsButtonRerun:
		app.runChecks()
		app.Redraw()
	case mapsButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(mapsButtonExit, "Exit", 160, 272)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(mapsWindowX, mapsWindowY, mapsWindowWidth, mapsWindowHeight, mapsWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "Map checks: insert/delete/rehash/range/key kinds")
	app.drawChecks()
	app.rerun.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) drawChecks() {
	y := 92
	for _, check := range app.checks {
		kos.DrawText(28, y, app.statusColor(check.ok), check.text)
		y += 20
	}
}

func (app *App) runChecks() {
	app.checks = app.checks[:0]

	ok, line := checkNilMapRead()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkStringMap()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkRehashInsert()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkTombstoneReuse()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkRangeSum()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkStructKey()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkInterfaceKey()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	ok, line = checkFloatKey()
	app.checks = append(app.checks, checkLine{ok: ok, text: line})

	if app.allOK() {
		app.summary = "maps ok"
		return
	}

	app.summary = "maps failure"
}

func (app *App) allOK() bool {
	for _, check := range app.checks {
		if !check.ok {
			return false
		}
	}

	return true
}

func (app *App) summaryColor() kos.Color {
	return app.statusColor(app.allOK())
}

func (app *App) statusColor(ok bool) kos.Color {
	if ok {
		return ui.Lime
	}

	return ui.Red
}
