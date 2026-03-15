package main

import (
	"kos"
	"ui"
	"ui/elements"
)

const (
	windowWidth  = 640
	windowHeight = 400

	padding           = 10
	headerHeight      = 20
	bottomPanelHeight = 30
	lineHeight        = 18

	btnDeleteLast kos.ButtonID = 10
	btnDeleteAll  kos.ButtonID = 11
	btnUnlock     kos.ButtonID = 12
	btnPrev       kos.ButtonID = 13
	btnNext       kos.ButtonID = 14
	btnRefresh    kos.ButtonID = 15
)

const (
	mouseLeftPressed = 1 << 8
)

type ClipboardSlot struct {
	Index    int
	Size     uint32
	Type     kos.ClipboardType
	Encoding kos.ClipboardEncoding
	Preview  string
}

type App struct {
	x         int
	y         int
	clientX   int
	clientY   int
	clientW   int
	clientH   int
	listX     int
	listY     int
	listW     int
	listH     int
	previewY  int
	visible   int
	first     int
	selected  int
	lastCount int
	slots     []ClipboardSlot
	status    string

	btnLast    *ui.Element
	btnAll     *ui.Element
	btnUnlock  *ui.Element
	btnPrev    *ui.Element
	btnNext    *ui.Element
	btnRefresh *ui.Element
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
	listX := clientX + padding
	listY := clientY + headerHeight
	listW := clientW - padding*2
	listH := clientH - headerHeight - bottomPanelHeight - padding
	if listH < lineHeight {
		listH = lineHeight
	}
	visible := listH / lineHeight
	previewY := listY + listH + 4
	buttonY := clientY + clientH - bottomPanelHeight + 4

	last := elements.ButtonAt(btnDeleteLast, "Delete last", clientX+10, buttonY)
	last.SetSize(110, 22)
	last.SetBackground(ui.Silver)
	last.SetForeground(ui.Black)

	all := elements.ButtonAt(btnDeleteAll, "Delete all", clientX+130, buttonY)
	all.SetSize(110, 22)
	all.SetBackground(ui.Silver)
	all.SetForeground(ui.Black)

	unlock := elements.ButtonAt(btnUnlock, "Unlock", clientX+250, buttonY)
	unlock.SetSize(90, 22)
	unlock.SetBackground(ui.Silver)
	unlock.SetForeground(ui.Black)

	prev := elements.ButtonAt(btnPrev, "<", clientX+360, buttonY)
	prev.SetSize(30, 22)
	prev.SetBackground(ui.Silver)
	prev.SetForeground(ui.Black)

	next := elements.ButtonAt(btnNext, ">", clientX+400, buttonY)
	next.SetSize(30, 22)
	next.SetBackground(ui.Silver)
	next.SetForeground(ui.Black)

	refresh := elements.ButtonAt(btnRefresh, "Refresh", clientX+450, buttonY)
	refresh.SetSize(80, 22)
	refresh.SetBackground(ui.Silver)
	refresh.SetForeground(ui.Black)

	app := &App{
		x:          x,
		y:          y,
		clientX:    clientX,
		clientY:    clientY,
		clientW:    clientW,
		clientH:    clientH,
		listX:      listX,
		listY:      listY,
		listW:      listW,
		listH:      listH,
		previewY:   previewY,
		visible:    visible,
		selected:   -1,
		btnLast:    last,
		btnAll:     all,
		btnUnlock:  unlock,
		btnPrev:    prev,
		btnNext:    next,
		btnRefresh: refresh,
	}
	app.refreshSlots()
	return app
}

func (app *App) Run() {
	kos.InitHeapRaw()
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse)

	for {
		switch kos.WaitEventFor(20) {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		case kos.EventMouse:
			app.handleMouse()
		case kos.EventKey:
			if app.handleKey() {
				return
			}
		case kos.EventNone:
			app.pollClipboard()
		}
	}
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

func (app *App) handleMouse() {
	state := kos.GetMouseButtonEventState()
	if state&mouseLeftPressed == 0 {
		return
	}
	mx, my := decodePackedCoords(kos.GetMouseWindowPosition())
	if mx < app.listX || mx >= app.listX+app.listW {
		return
	}
	if my < app.listY || my >= app.listY+app.listH {
		return
	}
	row := (my - app.listY) / lineHeight
	index := app.first + row
	if index >= 0 && index < len(app.slots) {
		app.selected = index
		app.Redraw()
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case 1:
		kos.Exit()
		return true
	case btnDeleteLast:
		kos.ClipboardDeleteLast()
		app.refreshSlots()
		app.Redraw()
	case btnDeleteAll:
		app.deleteAll()
		app.refreshSlots()
		app.Redraw()
	case btnUnlock:
		kos.ClipboardUnlockBuffer()
		app.refreshSlots()
		app.Redraw()
	case btnPrev:
		app.page(-1)
	case btnNext:
		app.page(1)
	case btnRefresh:
		app.refreshSlots()
		app.Redraw()
	}

	return false
}

func (app *App) pollClipboard() {
	count, status := kos.ClipboardSlotCount()
	if status != kos.ClipboardOK {
		return
	}
	if count != app.lastCount {
		app.refreshSlots()
		app.Redraw()
	}
}

func (app *App) page(delta int) {
	if len(app.slots) == 0 {
		return
	}
	app.first += delta * app.visible
	if app.first < 0 {
		app.first = 0
	}
	maxFirst := len(app.slots) - app.visible
	if maxFirst < 0 {
		maxFirst = 0
	}
	if app.first > maxFirst {
		app.first = maxFirst
	}
	app.Redraw()
}

func (app *App) deleteAll() {
	for {
		count, status := kos.ClipboardSlotCount()
		if status != kos.ClipboardOK || count <= 0 {
			return
		}
		if kos.ClipboardDeleteLast() != kos.ClipboardOK {
			return
		}
	}
}

func (app *App) refreshSlots() {
	count, status := kos.ClipboardSlotCount()
	if status != kos.ClipboardOK {
		app.slots = nil
		app.status = "Clipboard unavailable"
		app.lastCount = 0
		app.selected = -1
		return
	}

	slots := make([]ClipboardSlot, 0, count)
	for i := 0; i < count; i++ {
		slots = append(slots, readSlot(i))
	}
	app.slots = slots
	app.lastCount = count
	if count == 0 {
		app.selected = -1
		app.first = 0
		app.status = "Clipboard empty"
		return
	}
	app.status = "Slots: " + itoa(count)
	if app.selected < 0 || app.selected >= count {
		app.selected = 0
	}
	if app.first > app.selected {
		app.first = app.selected
	}
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(app.x, app.y, windowWidth, windowHeight, "Clipboard Viewer")
	kos.FillRect(app.clientX, app.clientY, app.clientW, app.clientH, ui.White)

	kos.DrawText(app.listX, app.clientY+4, ui.Black, "#   Size   Type   Preview")
	app.drawList()

	app.btnLast.Draw()
	app.btnAll.Draw()
	app.btnUnlock.Draw()
	app.btnPrev.Draw()
	app.btnNext.Draw()
	app.btnRefresh.Draw()

	app.drawPreview()
	kos.EndRedraw()
}

func (app *App) drawList() {
	y := app.listY
	for row := 0; row < app.visible; row++ {
		index := app.first + row
		if index >= len(app.slots) {
			break
		}
		slot := app.slots[index]
		if index == app.selected {
			kos.FillRect(app.listX, y, app.listW, lineHeight, ui.Silver)
		}

		kos.DrawText(app.listX+4, y+2, ui.Black, itoa(slot.Index))
		kos.DrawText(app.listX+40, y+2, ui.Black, itoa(int(slot.Size)))
		kos.DrawText(app.listX+120, y+2, ui.Black, slotTypeName(slot.Type))
		kos.DrawText(app.listX+200, y+2, ui.Black, slot.Preview)
		y += lineHeight
	}
}

func (app *App) drawPreview() {
	if app.selected < 0 || app.selected >= len(app.slots) {
		kos.DrawText(app.listX, app.previewY, ui.Gray, app.status)
		return
	}
	slot := app.slots[app.selected]
	line := "Selected #" + itoa(slot.Index) + "  size " + itoa(int(slot.Size)) + "  type " + slotTypeName(slot.Type)
	kos.DrawText(app.listX, app.previewY, ui.Gray, line)
	if slot.Preview != "" {
		kos.DrawText(app.listX, app.previewY+18, ui.Black, slot.Preview)
	}
}

func readSlot(index int) ClipboardSlot {
	slot := ClipboardSlot{Index: index}
	ptr, status := kos.ClipboardSlotData(index)
	if status != kos.ClipboardOK {
		slot.Preview = "<error>"
		return slot
	}

	size := kos.ReadUint32Raw(ptr, 0)
	kind := kos.ReadUint32Raw(ptr, 4)
	encoding := kos.ReadUint32Raw(ptr, 8)

	slot.Size = size
	slot.Type = kos.ClipboardType(kind)
	slot.Encoding = kos.ClipboardEncoding(encoding)

	offset := uint32(8)
	if slot.Type == kos.ClipboardTypeText || slot.Type == kos.ClipboardTypeTextBlock {
		offset = 12
	}
	if size <= offset {
		return slot
	}
	dataLen := size - offset
	previewLen := dataLen
	if previewLen > 80 {
		previewLen = 80
	}
	data := kos.CopyBytesRaw(ptr+offset, previewLen)
	slot.Preview = sanitizePreview(data)
	return slot
}

func slotTypeName(kind kos.ClipboardType) string {
	switch kind {
	case kos.ClipboardTypeText:
		return "Text"
	case kos.ClipboardTypeTextBlock:
		return "TextBlk"
	case kos.ClipboardTypeImage:
		return "Image"
	case kos.ClipboardTypeRaw:
		return "Raw"
	default:
		return "Unknown"
	}
}

func sanitizePreview(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	length := len(data)
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		return ""
	}
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		b := data[i]
		if b < 32 || b > 126 {
			buf[i] = '.'
		} else {
			buf[i] = b
		}
	}
	return string(buf)
}

func decodePackedCoords(value uint32) (int, int) {
	x := int(int16(value >> 16))
	y := int(int16(value & 0xFFFF))
	return x, y
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	negative := false
	if value < 0 {
		negative = true
		value = -value
	}

	var buf [20]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}

	if negative {
		index--
		buf[index] = '-'
	}

	return string(buf[index:])
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
