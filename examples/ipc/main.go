package main

import (
	"os"

	"kos"
	"ui"
)

const (
	ipcButtonExit  kos.ButtonID = 1
	ipcButtonSend  kos.ButtonID = 2
	ipcButtonClear kos.ButtonID = 3

	ipcWindowX      = 320
	ipcWindowY      = 170
	ipcWindowWidth  = 600
	ipcWindowHeight = 260
	ipcWindowTitle  = "KolibriOS IPC Demo"

	ipcBufferSize = 512
)

var ipcPayload = [...]byte{'s', 'e', 'l', 'f', ' ', 'i', 'p', 'c'}

type App struct {
	pid           uint32
	buffer        [ipcBufferSize]byte
	sent          uint32
	received      uint32
	lastBatch     uint32
	lastRegister  kos.IPCStatus
	lastStatus    kos.IPCStatus
	lastSender    uint32
	lastLength    uint32
	lastFirstByte byte
	lastHasData   bool
	send          *ui.Element
	clear         *ui.Element
}

func (app *App) Init() {
	app.send.ID = ipcButtonSend
	app.send.Label = "Send self IPC"
	app.send.SetLeft(28)
	app.send.SetTop(202)
	app.send.SetWidth(152)
	app.send.SetHeight(30)
	app.send.SetBackground(ui.Blue)
	app.send.SetForeground(ui.White)
	app.send.SetPadding(8, 10)

	app.clear.ID = ipcButtonClear
	app.clear.Label = "Clear"
	app.clear.SetLeft(200)
	app.clear.SetTop(202)
	app.clear.SetWidth(96)
	app.clear.SetHeight(30)
	app.clear.SetBackground(ui.Blue)
	app.clear.SetForeground(ui.White)
	app.clear.SetPadding(8, 10)

	id, ok := kos.CurrentThreadID()
	if ok {
		app.pid = id
	}

	app.lastRegister = kos.RegisterIPCBuffer(app.buffer[:])
	app.lastStatus = ipcStatusUnknown
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC)
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
		case kos.EventIPC:
			app.handleIPC()
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case ipcButtonSend:
		app.sendSelf()
		app.Redraw()
	case ipcButtonClear:
		app.resetState()
		kos.ResetIPCBuffer(app.buffer[:])
		app.Redraw()
	case ipcButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) handleIPC() {
	summary := kos.InspectIPCBuffer(app.buffer[:])
	if summary.MessageCount == 0 {
		return
	}

	app.lastBatch = summary.MessageCount
	app.received += summary.MessageCount
	app.lastSender = summary.LastSenderPID
	app.lastLength = summary.LastLength
	app.lastFirstByte = summary.LastFirstByte
	app.lastHasData = summary.LastHasData
	kos.ResetIPCBuffer(app.buffer[:])
	app.Redraw()
}

func (app *App) sendSelf() {
	app.lastStatus = kos.SendIPCRaw(app.pid, &ipcPayload[0], uint32(len(ipcPayload)))
	if app.lastStatus == kos.IPCOK {
		app.sent++
	}
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(ipcWindowX, ipcWindowY, ipcWindowWidth, ipcWindowHeight, ipcWindowTitle)
	kos.DrawText(28, 44, ui.White, "Function 60 registers an IPC buffer and delivers event 7")
	kos.DrawText(28, 62, ui.Silver, "This sample sends IPC to its own PID/TID")
	kos.DrawText(28, 88, ui.Aqua, "Self PID: "+formatUint32(app.pid))
	kos.DrawText(28, 106, ui.Yellow, "Register status: "+app.statusString(app.lastRegister))
	kos.DrawText(28, 124, ui.Lime, "Last send status: "+app.statusString(app.lastStatus))
	kos.DrawText(28, 142, ui.White, "Buffer size: "+formatUint32(uint32(len(app.buffer)))+" bytes")
	kos.DrawText(28, 160, ui.Silver, "Messages sent/received: "+formatUint32(app.sent)+" / "+formatUint32(app.received))
	kos.DrawText(28, 178, ui.Aqua, "Last IPC batch: "+formatUint32(app.lastBatch))
	kos.DrawText(320, 106, ui.Yellow, "Last sender: "+formatUint32(app.lastSender))
	kos.DrawText(320, 124, ui.Lime, "Last payload len: "+formatUint32(app.lastLength))
	kos.DrawText(320, 142, ui.White, "Last first byte: "+app.firstByteString())
	kos.DrawText(320, 160, ui.Silver, "Buffer locked: "+app.lockedString())
	app.send.Draw()
	app.clear.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.sent = 0
	app.received = 0
	app.lastBatch = 0
	app.lastStatus = ipcStatusUnknown
	app.lastSender = 0
	app.lastLength = 0
	app.lastFirstByte = 0
	app.lastHasData = false
}

func (app *App) firstByteString() string {
	if !app.lastHasData {
		return "-"
	}

	return formatUint32(uint32(app.lastFirstByte))
}

func (app *App) lockedString() string {
	if kos.IPCBufferIsLocked(app.buffer[:]) {
		return "yes"
	}

	return "no"
}

func (app *App) statusString(status kos.IPCStatus) string {
	switch status {
	case ipcStatusUnknown:
		return "none"
	case kos.IPCOK:
		return "ok"
	case kos.IPCReceiverMissing:
		return "receiver has no IPC buffer"
	case kos.IPCBufferLocked:
		return "receiver buffer locked"
	case kos.IPCBufferFull:
		return "receiver buffer full"
	case kos.IPCReceiverGone:
		return "receiver not found"
	}

	return formatUint32(uint32(status))
}

const ipcStatusUnknown kos.IPCStatus = 0xFFFFFFFF

func main() {
	var app App
	app.Init()
	app.Run()
}
