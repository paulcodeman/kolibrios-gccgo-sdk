package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	inputBoxButtonExit      kos.ButtonID = 1
	inputBoxButtonAskString kos.ButtonID = 2
	inputBoxButtonAskNumber kos.ButtonID = 3
	inputBoxButtonReset     kos.ButtonID = 4
	inputBoxWindowTitle                  = "KolibriOS INPUTBOX Demo"
	inputBoxWindowX                      = 220
	inputBoxWindowY                      = 146
	inputBoxWindowWidth                  = 900
	inputBoxWindowHeight                 = 336
)

type App struct {
	summary      string
	stringLine   string
	numberLine   string
	dllLine      string
	noteLine     string
	ok           bool
	askStringBtn *ui.Element
	askNumberBtn *ui.Element
	resetBtn     *ui.Element
}

func NewApp() App {
	askString := elements.ButtonAt(inputBoxButtonAskString, "Ask string", 28, 282)
	askString.SetWidth(126)

	askNumber := elements.ButtonAt(inputBoxButtonAskNumber, "Ask number", 174, 282)
	askNumber.SetWidth(132)

	reset := elements.ButtonAt(inputBoxButtonReset, "Reset", 326, 282)
	reset.SetWidth(96)

	app := App{
		askStringBtn: askString,
		askNumberBtn: askNumber,
		resetBtn:     reset,
	}
	app.resetState()
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
	case inputBoxButtonAskString:
		app.runStringPrompt()
		app.Redraw()
	case inputBoxButtonAskNumber:
		app.runNumberPrompt()
		app.Redraw()
	case inputBoxButtonReset:
		app.resetState()
		app.Redraw()
	case inputBoxButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(inputBoxButtonExit, "Exit", 778, 282)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(inputBoxWindowX, inputBoxWindowY, inputBoxWindowWidth, inputBoxWindowHeight, inputBoxWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample uses kos.LoadInputBox() with the exported InputBox(...) modal helper")
	kos.DrawText(28, 92, ui.Aqua, app.stringLine)
	kos.DrawText(28, 114, ui.Lime, app.numberLine)
	kos.DrawText(28, 136, ui.Yellow, app.dllLine)
	kos.DrawText(28, 158, ui.Black, app.noteLine)
	kos.DrawText(28, 184, ui.Black, "String prompt uses screen-relative positioning; number prompt uses parent-relative positioning")
	kos.DrawText(28, 206, ui.Black, "The current Go wrapper intentionally passes no redraw callback and relies on the library's own modal loop")
	app.askStringBtn.Draw()
	app.askNumberBtn.Draw()
	app.resetBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.ok = false
	app.summary = "inputbox wrapper ready / press Ask string or Ask number"
	app.stringLine = "String: no prompt result yet"
	app.numberLine = "Number: no prompt result yet"
	app.dllLine = "DLL: " + kos.InputBoxDLLPath
	app.noteLine = "Info: cancel returns the provided default value; status still reports any parsing/truncation error"
}

func (app *App) runStringPrompt() {
	inputBox, ok := kos.LoadInputBox()
	if !ok {
		app.fail("inputbox.obj unavailable", "Info: failed to load "+kos.InputBoxDLLPath)
		return
	}

	value, status, ok := inputBox.PromptString(
		"Go InputBox",
		"Enter a label for the sample window",
		"go-inputbox",
		kos.InputBoxScreenRelative,
		80,
	)
	if !ok {
		app.fail("string prompt failed", "Info: InputBox export did not accept the string prompt call")
		return
	}

	app.ok = status == kos.InputBoxNoError
	app.summary = "inputbox string prompt returned"
	app.stringLine = fmt.Sprintf("String: %q / status=%s", value, inputBoxErrorText(status))
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x", kos.InputBoxDLLPath, uint32(inputBox.ExportTable()))
	app.noteLine = "Info: cancel restores the default string; the wrapper keeps the raw library semantics explicit"
}

func (app *App) runNumberPrompt() {
	inputBox, ok := kos.LoadInputBox()
	if !ok {
		app.fail("inputbox.obj unavailable", "Info: failed to load "+kos.InputBoxDLLPath)
		return
	}

	value, status, ok := inputBox.PromptNumber(
		"Go InputBox",
		"Enter a refresh count",
		"42",
		kos.InputBoxParentRelative,
	)
	if !ok {
		app.fail("number prompt failed", "Info: InputBox export did not accept the numeric prompt call")
		return
	}

	app.ok = status == kos.InputBoxNoError
	app.summary = "inputbox number prompt returned"
	app.numberLine = fmt.Sprintf("Number: %d / status=%s", value, inputBoxErrorText(status))
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x", kos.InputBoxDLLPath, uint32(inputBox.ExportTable()))
	app.noteLine = "Info: numeric mode returns a uint64 buffer; cancel falls back to the default decimal string"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "inputbox probe failed / " + detail
	app.noteLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func inputBoxErrorText(status kos.InputBoxError) string {
	switch status {
	case kos.InputBoxNoError:
		return "ok"
	case kos.InputBoxNumberOverflow:
		return "number overflow"
	case kos.InputBoxResultTooLong:
		return "result too long"
	default:
		return fmt.Sprintf("status %d", uint32(status))
	}
}
