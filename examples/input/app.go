package main

import (
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	messageButtonExit         kos.ButtonID = 1
	messageButtonInjectButton kos.ButtonID = 2
	messageButtonInjectKey    kos.ButtonID = 3
	messageButtonReset        kos.ButtonID = 4
	messageButtonSynthetic    kos.ButtonID = 9

	messageWindowX      = 340
	messageWindowY      = 170
	messageWindowWidth  = 560
	messageWindowHeight = 250
	messageWindowTitle  = "KolibriOS Input Demo"
	messageButtonTitle  = "KolibriOS Входящее сообщение"
	messageKeyTitle     = "KolibriOS Входящая клавиша"
)

type App struct {
	sentButtons     uint32
	receivedButtons uint32
	sentKeys        uint32
	receivedKeys    uint32
	lastStatus      kos.MessageStatus
	lastButton      kos.ButtonID
	lastKey         kos.KeyEvent
	haveKey         bool
	injectButton    *ui.Element
	injectKey       *ui.Element
	reset           *ui.Element
}

func NewApp() App {
	injectButton := elements.ButtonAt(messageButtonInjectButton, "Inject button", 28, 192)
	injectButton.SetWidth(150)

	injectKey := elements.ButtonAt(messageButtonInjectKey, "Inject key", 198, 192)
	injectKey.SetWidth(132)

	reset := elements.ButtonAt(messageButtonReset, "Reset", 350, 192)
	reset.SetWidth(100)

	return App{
		lastStatus:   messageStatusUnknown,
		injectButton: injectButton,
		injectKey:    injectKey,
		reset:        reset,
	}
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
		case kos.EventKey:
			app.handleKeyEvent()
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case messageButtonInjectButton:
		app.sentButtons++
		app.lastStatus = kos.SendActiveWindowButton(messageButtonSynthetic)
		app.Redraw()
	case messageButtonInjectKey:
		app.sentKeys++
		app.lastStatus = kos.SendActiveWindowKey(uint32('R') << 8)
		app.Redraw()
	case messageButtonReset:
		app.resetState()
		kos.SetWindowTitle(messageWindowTitle)
		app.Redraw()
	case messageButtonSynthetic:
		app.receivedButtons++
		app.lastButton = id
		kos.SetWindowTitleWithEncodingPrefix(kos.EncodingUTF8, messageButtonTitle)
		app.Redraw()
	case messageButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) handleKeyEvent() {
	key := kos.ReadKey()
	if key.Empty {
		return
	}

	app.receivedKeys++
	app.lastKey = key
	app.haveKey = true
	kos.SetWindowTitleWithEncodingPrefix(kos.EncodingUTF8, messageKeyTitle)
	app.Redraw()
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(messageWindowX, messageWindowY, messageWindowWidth, messageWindowHeight, messageWindowTitle)
	kos.DrawText(28, 44, ui.White, "Function 72 sends events to the active window")
	kos.DrawText(28, 62, ui.Silver, "Inject button sends ButtonID 9")
	kos.DrawText(28, 80, ui.Silver, "Inject key sends ASCII 'R' in Function 2 format")
	kos.DrawText(28, 106, ui.Aqua, "Send status: "+app.statusString())
	kos.DrawText(28, 124, ui.Yellow, "Buttons sent/received: "+messageFormatUint32(app.sentButtons)+" / "+messageFormatUint32(app.receivedButtons))
	kos.DrawText(28, 142, ui.Lime, "Keys sent/received: "+messageFormatUint32(app.sentKeys)+" / "+messageFormatUint32(app.receivedKeys))
	kos.DrawText(28, 160, ui.White, "Last button id: "+messageFormatInt(int(app.lastButton)))
	kos.DrawText(300, 124, ui.Aqua, "Last key raw: "+app.keyRawString())
	kos.DrawText(300, 142, ui.Yellow, "Last key code: "+app.keyCodeString())
	kos.DrawText(300, 160, ui.Lime, "Last scancode: "+app.keyScanCodeString())
	app.injectButton.Draw()
	app.injectKey.Draw()
	app.reset.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.sentButtons = 0
	app.receivedButtons = 0
	app.sentKeys = 0
	app.receivedKeys = 0
	app.lastStatus = messageStatusUnknown
	app.lastButton = 0
	app.lastKey = kos.KeyEvent{}
	app.haveKey = false
}

func (app *App) statusString() string {
	switch app.lastStatus {
	case messageStatusUnknown:
		return "not sent"
	case kos.MessageOK:
		return "ok"
	case kos.MessageBufferFull:
		return "buffer full"
	}

	return messageFormatUint32(uint32(app.lastStatus))
}

func (app *App) keyRawString() string {
	if !app.haveKey {
		return "-"
	}

	return messageFormatHex32(uint32(app.lastKey.Raw))
}

func (app *App) keyCodeString() string {
	if !app.haveKey {
		return "-"
	}

	return messageFormatUint32(uint32(app.lastKey.Code))
}

func (app *App) keyScanCodeString() string {
	if !app.haveKey {
		return "-"
	}

	return messageFormatUint32(uint32(app.lastKey.ScanCode))
}

const messageStatusUnknown kos.MessageStatus = 0xFFFFFFFF
