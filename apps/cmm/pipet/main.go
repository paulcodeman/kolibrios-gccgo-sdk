package main

import (
	"kos"
	"ui"
	"ui/elements"
)

const (
	windowWidth  = 240
	windowHeight = 110

	buttonHeight    = 20
	buttonGap       = 10
	buttonTextPadX  = 4
	buttonFontWidth = 8
	fontHeight      = 16
	textGap         = 4
	textTopMin      = 2

	btnPick    kos.ButtonID = 2
	btnCopyHex kos.ButtonID = 3
	btnCopyRGB kos.ButtonID = 4
)

const (
	mouseLeftPressed = 1 << 8
)

type App struct {
	x          int
	y          int
	clientX    int
	clientY    int
	clientW    int
	clientH    int
	color      uint32
	hex        string
	rgb        string
	status     string
	pick       bool
	layerSet   bool
	pickBtn    *ui.Element
	copyHexBtn *ui.Element
	copyRgbBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() *App {
	width, height := kos.ScreenSize()
	x := (width - windowWidth) / 2
	y := (height - windowHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	clientX, clientY, clientW, clientH := windowClientRect(windowWidth, windowHeight)
	buttonY := clientY + clientH - buttonHeight - 6

	pickLabel := "Pick"
	copyHexLabel := "Copy HEX"
	copyRgbLabel := "Copy RGB"

	pickWidth := buttonWidthForLabel(pickLabel)
	copyHexWidth := buttonWidthForLabel(copyHexLabel)
	copyRgbWidth := buttonWidthForLabel(copyRgbLabel)
	gap := buttonGap
	total := pickWidth + copyHexWidth + copyRgbWidth + gap*4
	if total > clientW {
		gap = (clientW - (pickWidth + copyHexWidth + copyRgbWidth)) / 4
		if gap < 2 {
			gap = 2
		}
	}

	pickX := clientX + gap
	copyHexX := pickX + pickWidth + gap
	copyRgbX := copyHexX + copyHexWidth + gap

	pick := elements.ButtonAt(btnPick, pickLabel, pickX, buttonY)
	pick.SetSize(pickWidth, buttonHeight)
	pick.SetPadding(0, buttonTextPadX)
	pick.SetBackground(ui.Silver)
	pick.SetForeground(ui.Black)

	copyHex := elements.ButtonAt(btnCopyHex, copyHexLabel, copyHexX, buttonY)
	copyHex.SetSize(copyHexWidth, buttonHeight)
	copyHex.SetPadding(0, buttonTextPadX)
	copyHex.SetBackground(ui.Silver)
	copyHex.SetForeground(ui.Black)

	copyRgb := elements.ButtonAt(btnCopyRGB, copyRgbLabel, copyRgbX, buttonY)
	copyRgb.SetSize(copyRgbWidth, buttonHeight)
	copyRgb.SetPadding(0, buttonTextPadX)
	copyRgb.SetBackground(ui.Silver)
	copyRgb.SetForeground(ui.Black)

	app := &App{
		x:          x,
		y:          y,
		clientX:    clientX,
		clientY:    clientY,
		clientW:    clientW,
		clientH:    clientH,
		color:      0xFFFFFF,
		status:     "Pick mode",
		pick:       true,
		pickBtn:    pick,
		copyHexBtn: copyHex,
		copyRgbBtn: copyRgb,
	}
	app.updateStrings()
	return app
}

func buttonWidthForLabel(label string) int {
	return len(label)*buttonFontWidth + buttonTextPadX*2
}

func (app *App) Run() {
	kos.InitHeapRaw()
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse)

	for {
		switch kos.WaitEventFor(10) {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventMouse:
			app.handleMouse()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		case kos.EventKey:
			if app.handleKey() {
				return
			}
		case kos.EventNone:
			app.tick()
		}
	}
}

func (app *App) handleMouse() {
	if !app.pick {
		return
	}
	changed := app.updateColorFromMouse()
	state := kos.GetMouseButtonEventState()
	if state&mouseLeftPressed != 0 {
		app.pick = false
		app.status = "Picked"
		changed = true
	}
	if changed {
		app.Redraw()
	}
}

func (app *App) tick() {
	if !app.pick {
		return
	}
	changed := app.updateColorFromMouse()
	state := kos.GetMouseButtonEventState()
	if state&mouseLeftPressed != 0 {
		app.pick = false
		app.status = "Picked"
		changed = true
	}
	if changed {
		app.Redraw()
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case 1:
		kos.Exit()
		return true
	case btnPick:
		app.pick = true
		app.status = "Pick mode"
		app.updateColorFromMouse()
		app.Redraw()
	case btnCopyHex:
		kos.ClipboardCopyText(app.hex)
		app.status = "HEX copied"
		app.Redraw()
	case btnCopyRGB:
		kos.ClipboardCopyText(app.rgb)
		app.status = "RGB copied"
		app.Redraw()
	}

	return false
}

func (app *App) handleKey() bool {
	key := kos.ReadKey()
	if key.Empty {
		return false
	}
	if key.Code == 27 || key.ScanCode == 1 {
		kos.Exit()
		return true
	}
	return false
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(app.x, app.y, windowWidth, windowHeight, "Pipet")
	if !app.layerSet {
		kos.SetWindowLayerBehaviour(kos.WindowLayerAlwaysTop)
		app.layerSet = true
	}
	kos.FillRect(app.clientX, app.clientY, app.clientW, app.clientH, ui.White)

	buttonY := app.pickBtn.Bounds().Y
	line2Y := buttonY - fontHeight - textGap
	line1Y := line2Y - fontHeight
	line0Y := line1Y - fontHeight
	minTextY := app.clientY + textTopMin
	if line0Y < minTextY {
		line0Y = minTextY
		line1Y = line0Y + fontHeight
		line2Y = line1Y + fontHeight
	}

	swatchX := app.clientX + app.clientW - 10 - 40
	swatchY := line0Y
	kos.FillRect(swatchX, swatchY, 40, 40, kos.Color(app.color))

	kos.DrawText(app.clientX+10, line0Y, ui.Black, "HEX:")
	kos.DrawText(app.clientX+52, line0Y, ui.Black, app.hex)
	kos.DrawText(app.clientX+10, line1Y, ui.Black, "RGB:")
	kos.DrawText(app.clientX+52, line1Y, ui.Black, app.rgb)
	kos.DrawText(app.clientX+10, line2Y, ui.Gray, app.status)

	app.pickBtn.Draw()
	app.copyHexBtn.Draw()
	app.copyRgbBtn.Draw()

	kos.EndRedraw()
}

func (app *App) updateColorFromMouse() bool {
	packed := kos.GetMouseScreenPosition()
	x := int(packed >> 16)
	y := int(packed & 0xFFFF)
	next := kos.GetPixelColorFromScreen(x, y)
	if next == app.color {
		return false
	}
	app.color = next
	app.updateStrings()
	return true
}

func (app *App) updateStrings() {
	r := int((app.color >> 16) & 0xFF)
	g := int((app.color >> 8) & 0xFF)
	b := int(app.color & 0xFF)
	app.hex = formatHex(app.color)
	app.rgb = formatRGB(r, g, b)
}

func formatHex(value uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 5; i >= 0; i-- {
		buf[i] = digits[value&0xF]
		value >>= 4
	}
	return string(buf[:])
}

func formatRGB(r int, g int, b int) string {
	var buf [11]byte
	copy(buf[0:3], format3(r))
	buf[3] = ','
	copy(buf[4:7], format3(g))
	buf[7] = ','
	copy(buf[8:11], format3(b))
	return string(buf[:])
}

func format3(value int) []byte {
	if value < 0 {
		value = 0
	}
	if value > 999 {
		value = 999
	}
	return []byte{
		byte('0' + value/100),
		byte('0' + (value/10)%10),
		byte('0' + value%10),
	}
}

func windowClientRect(width int, height int) (int, int, int, int) {
	skin := kos.SkinHeight()
	x := 5
	y := skin
	w := width - 9
	h := height - skin - 4
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return x, y, w, h
}
