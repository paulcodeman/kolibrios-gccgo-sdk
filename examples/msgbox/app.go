package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	msgBoxButtonExit  kos.ButtonID = 1
	msgBoxButtonShow  kos.ButtonID = 2
	msgBoxButtonReset kos.ButtonID = 3

	msgBoxWindowTitle  = "KolibriOS MSGBOX Demo"
	msgBoxWindowX      = 230
	msgBoxWindowY      = 154
	msgBoxWindowWidth  = 860
	msgBoxWindowHeight = 316
)

type App struct {
	summary         string
	resultLine      string
	dllLine         string
	noteLine        string
	ok              bool
	showBtn         *ui.Element
	resetBtn        *ui.Element
	parentSlot      int
	parentSlotKnown bool
	dialog          kos.MsgBox
	box             kos.MsgBoxData
	modalActive     bool
	modalSeen       bool
	launchCount     uint32
}

func NewApp() App {
	show := elements.ButtonAt(msgBoxButtonShow, "Show MSGBOX", 28, 264)
	show.SetWidth(146)

	reset := elements.ButtonAt(msgBoxButtonReset, "Reset", 194, 264)
	reset.SetWidth(96)

	parentSlot, ok := kos.CurrentThreadSlotIndex()
	app := App{
		showBtn:         show,
		resetBtn:        reset,
		parentSlot:      parentSlot,
		parentSlotKnown: ok,
	}
	app.resetState()
	return app
}

func (app *App) Run() {
	for {
		event := kos.WaitEvent()
		changed := app.pollDialog()

		switch event {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		default:
			if changed {
				app.Redraw()
			}
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case msgBoxButtonShow:
		app.showDialog()
		app.Redraw()
	case msgBoxButtonReset:
		app.resetState()
		app.Redraw()
	case msgBoxButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(msgBoxButtonExit, "Exit", 726, 264)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(msgBoxWindowX, msgBoxWindowY, msgBoxWindowWidth, msgBoxWindowHeight, msgBoxWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample uses kos.LoadMsgBox() and keeps the MSGBOX.OBJ modal thread contract explicit")
	kos.DrawText(28, 92, ui.Aqua, app.resultLine)
	kos.DrawText(28, 114, ui.Lime, app.dllLine)
	kos.DrawText(28, 136, ui.Yellow, app.noteLine)
	kos.DrawText(28, 162, ui.Black, "Buttons: Show MSGBOX opens a separate modal thread with [Yes] and [Later]")
	kos.DrawText(28, 184, ui.Black, "The parent window only records the result after focus returns to its original slot")
	app.showBtn.Draw()
	app.resetBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.ok = false
	app.summary = "msgbox wrapper ready / press Show MSGBOX to launch a modal box"
	app.resultLine = "Result: no modal box started yet"
	app.dllLine = "DLL: " + kos.MsgBoxDLLPath
	if app.parentSlotKnown {
		app.noteLine = fmt.Sprintf("Info: current parent slot is %d", app.parentSlot)
	} else {
		app.noteLine = "Info: current thread slot lookup is unavailable on this runtime"
	}
	app.modalActive = false
	app.modalSeen = false
	app.launchCount = 0
}

func (app *App) showDialog() {
	dialog, ok := kos.LoadMsgBox()
	if !ok {
		app.fail("msgbox.obj unavailable", "Info: failed to load "+kos.MsgBoxDLLPath)
		return
	}

	box, built := kos.NewMsgBox(
		"KolibriOS MSGBOX",
		"Proceed with the shared Go wrapper?\rThis box runs in its own modal thread.",
		2,
		"Yes",
		"Later",
	)
	if !built {
		app.fail("message box buffer build failed", "Info: title/text/button payload exceeded the packed MSGBOX.OBJ buffer contract")
		return
	}
	app.dialog = dialog
	app.box = box
	if !app.dialog.Start(&app.box) {
		app.fail("msgbox start failed", "Info: mb_create did not accept the packed message-box buffer")
		return
	}

	app.ok = true
	app.modalActive = true
	app.modalSeen = false
	app.launchCount++
	app.summary = "msgbox launched / wait for focus to return after the modal thread closes"
	app.resultLine = fmt.Sprintf("Result: pending / default=%d / launches=%d", app.box.Result(), app.launchCount)
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x", kos.MsgBoxDLLPath, uint32(dialog.ExportTable()))
	if app.parentSlotKnown {
		app.noteLine = fmt.Sprintf("Info: watching active window slot until it returns to %d", app.parentSlot)
	} else {
		app.noteLine = "Info: slot detection unavailable, so only the launch status can be shown"
	}
}

func (app *App) pollDialog() bool {
	if !app.modalActive || !app.parentSlotKnown {
		return false
	}

	activeSlot := kos.ActiveWindowSlot()
	if !app.modalSeen {
		if activeSlot != app.parentSlot {
			app.modalSeen = true
		}
		return false
	}
	if activeSlot != app.parentSlot {
		return false
	}

	app.modalActive = false
	result := app.box.Result()
	app.summary = "msgbox closed / modal result captured"
	app.resultLine = fmt.Sprintf("Result: %s (%d) / launches=%d", app.resultText(result), result, app.launchCount)
	app.noteLine = "Info: MsgBoxCreate is async; the wrapper polls the shared result byte after focus returns"
	return true
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "msgbox probe failed / " + detail
	app.noteLine = info
}

func (app *App) resultText(result uint8) string {
	switch result {
	case 0:
		return "closed by [X]"
	case 1:
		return "button 1 / Yes"
	case 2:
		return "button 2 / Later"
	default:
		return fmt.Sprintf("button %d", result)
	}
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
