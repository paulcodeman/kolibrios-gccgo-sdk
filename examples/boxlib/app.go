package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	boxLibButtonExit  kos.ButtonID = 1
	boxLibButtonReset kos.ButtonID = 2

	boxLibWindowTitle  = "KolibriOS BOX_LIB Demo"
	boxLibWindowX      = 200
	boxLibWindowY      = 118
	boxLibWindowWidth  = 900
	boxLibWindowHeight = 360
)

type App struct {
	summary    string
	editLine   string
	scrollLine string
	noteLine   string
	ok         bool
	resetBtn   *ui.Element
	lib        kos.BoxLib
	libLoaded  bool
	edit       *kos.EditBox
	scroll     *kos.ScrollBar
	progress   *kos.ProgressBar
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	reset := elements.ButtonAt(boxLibButtonReset, "Reset", 28, 304)
	reset.SetWidth(110)
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse)

	app := App{
		resetBtn: reset,
	}
	app.resetState()
	app.ensureWidgets()
	return app
}

func (app *App) Run() {
	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventKey:
			app.handleKey()
			app.Redraw()
		case kos.EventMouse:
			app.handleMouse()
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
	case boxLibButtonReset:
		app.resetWidgets()
		app.Redraw()
	case boxLibButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) handleKey() {
	var event kos.KeyEvent

	if !app.ensureWidgets() {
		return
	}

	event = kos.ReadKey()
	if event.Empty || event.Hotkey {
		return
	}

	app.lib.HandleEditBoxKey(app.edit, uint32(event.Raw))
	app.syncState("keyboard")
}

func (app *App) handleMouse() {
	if !app.ensureWidgets() {
		return
	}

	app.lib.HandleEditBoxMouse(app.edit)
	app.lib.HandleVerticalScrollBarMouse(app.scroll)
	app.progress.SetValue(app.scroll.Position())
	app.syncState("mouse")
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(boxLibButtonExit, "Exit", 156, 304)
	exit.SetWidth(92)

	kos.BeginRedraw()
	kos.OpenWindow(boxLibWindowX, boxLibWindowY, boxLibWindowWidth, boxLibWindowHeight, boxLibWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "BOX_LIB.OBJ edit_box, scrollbar, and progressbar via typed kos wrappers")
	kos.DrawText(28, 92, ui.Aqua, app.editLine)
	kos.DrawText(28, 114, ui.Lime, app.scrollLine)
	kos.DrawText(28, 136, ui.Yellow, app.noteLine)
	kos.DrawText(28, 164, ui.Black, "Type into the edit box, click inside it to refocus, and drag the scrollbar to change the progress value")
	kos.DrawText(28, 186, ui.Black, "This sample keeps BOX_LIB lifecycle explicit: one DLL load, caller-owned packed structs, redraw through library draw exports")
	_ = app.lib.DrawEditBox(app.edit)
	_ = app.lib.DrawVerticalScrollBar(app.scroll)
	_ = app.lib.DrawProgressBar(app.progress)
	app.resetBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.ok = false
	app.summary = "boxlib wrapper ready / loading BOX_LIB.OBJ"
	app.editLine = "Edit: waiting for widget initialization"
	app.scrollLine = "Scroll: waiting for widget initialization"
	app.noteLine = "Info: this sample uses BOX_LIB for widget draw/input instead of manual controls"
}

func (app *App) ensureWidgets() bool {
	if app.libLoaded {
		return true
	}

	lib, ok := kos.LoadBoxLib()
	if !ok {
		app.fail("box_lib.obj unavailable", "Info: failed to load "+kos.BoxLibDLLPath)
		return false
	}

	app.lib = lib
	app.libLoaded = true
	app.edit = kos.NewEditBox(28, 220, 290, 48, "go boxlib")
	app.scroll = kos.NewVerticalScrollBar(342, 208, 18, 74, 100, 18, 35)
	app.progress = kos.NewProgressBar(382, 224, 220, 18, 0, 100, app.scroll.Position())
	app.syncState("startup")
	return true
}

func (app *App) resetWidgets() {
	if !app.ensureWidgets() {
		return
	}

	app.lib.SetEditBoxText(app.edit, "go boxlib")
	app.edit.SetFlags(kos.EditBoxFlagAlwaysFocus | kos.EditBoxFlagFocus)
	app.scroll.SetPosition(35)
	app.progress.SetValue(app.scroll.Position())
	app.syncState("reset")
}

func (app *App) syncState(source string) {
	app.ok = true
	app.summary = "boxlib widgets active / edit, scrollbar, and progress paths loaded"
	app.editLine = fmt.Sprintf("Edit: %q / size=%d / cursor=%d / version 0x%x", app.edit.Text(), app.edit.Size(), app.edit.Position(), app.lib.EditVersion())
	app.scrollLine = fmt.Sprintf("Scroll: position=%d / progress=%d / version 0x%x / table 0x%x", app.scroll.Position(), app.progress.Value(), app.lib.ScrollBarVersion(), uint32(app.lib.ExportTable()))
	app.noteLine = "Info: last update came from " + source + " / progress tracks current scrollbar position"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "boxlib probe failed / " + detail
	app.noteLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
