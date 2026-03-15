package main

import (
	"dom"
	"kos"
	"strings"
	"ui"
	"ui/elements"
)

const (
	defaultWindowWidth  = 640
	defaultWindowHeight = 480

	padding    = 6
	toolbarH   = 28
	statusH    = 16
	scrollW    = 14
	buttonH    = 20
	buttonW    = 28
	goButtonW  = 40
	fontW      = 8
	fontH      = 16
	textGap    = 4
	maxContent = 512 * 1024
	defaultURL = "https://example.com"
)

const (
	btnBack kos.ButtonID = 2
	btnFwd  kos.ButtonID = 3
	btnGo   kos.ButtonID = 4
)

type App struct {
	x      int
	y      int
	width  int
	height int

	clientX int
	clientY int
	clientW int
	clientH int

	contentX int
	contentY int
	contentW int
	contentH int
	textCols int

	boxlib       kos.BoxLib
	edit         *kos.EditBox
	scroll       *kos.ScrollBar
	http         kos.HTTP
	loading      bool
	transfer     kos.HTTPTransfer
	lastReceived uint32

	backBtn *ui.Element
	fwdBtn  *ui.Element
	goBtn   *ui.Element

	statusBase   string
	pageTitle    string
	url          string
	history      []string
	historyIndex int

	lines       []RenderLine
	runs        []Run
	links       []string
	firstLine   int
	hoverLink   int
	pageBuf     *pageBuffer
	windowReady bool
}

func NewApp() *App {
	width, height := kos.ScreenSize()
	x := (width - defaultWindowWidth) / 2
	y := (height - defaultWindowHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	boxlib, ok := kos.LoadBoxLib()
	if !ok {
		kos.Exit()
	}

	http, _ := kos.LoadHTTP()

	app := &App{
		x:          x,
		y:          y,
		width:      defaultWindowWidth,
		height:     defaultWindowHeight,
		boxlib:     boxlib,
		http:       http,
		statusBase: "Ready",
		pageTitle:  "",
		hoverLink:  -1,
	}
	app.layout()
	app.setURL(defaultURL)
	app.runs = []Run{{Text: "Type a URL and press Enter or Go.", Link: -1}}
	app.lines = wrapRuns(app.runs, app.textCols)
	if len(app.lines) == 0 {
		app.lines = []RenderLine{{Text: ""}}
	}
	app.buildPageBuffer()
	return app
}

func (app *App) Run() {
	kos.InitHeapRaw()
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskNetwork)

	for {
		var event kos.EventType
		if app.loading {
			event = kos.WaitEventFor(10)
		} else {
			event = kos.WaitEvent()
		}
		switch event {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventKey:
			app.handleKey()
		case kos.EventMouse:
			app.handleMouse()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		case kos.EventNetwork, kos.EventNone:
			if app.loading {
				app.pollHTTP()
			}
		}
	}
}

func (app *App) layout() {
	clientX, clientY, clientW, clientH := windowClientRect(app.width, app.height)
	app.clientX = clientX
	app.clientY = clientY
	app.clientW = clientW
	app.clientH = clientH

	btnY := clientY + padding
	backX := clientX + padding
	fwdX := backX + buttonW + padding
	goX := clientX + clientW - padding - goButtonW
	editX := fwdX + buttonW + padding
	editW := goX - padding - editX
	if editW < 80 {
		editW = 80
	}

	app.backBtn = elements.ButtonAt(btnBack, "<", backX, btnY)
	app.backBtn.SetSize(buttonW, buttonH)
	app.backBtn.SetPadding(0, 8)
	app.backBtn.SetBackground(ui.Silver)
	app.backBtn.SetForeground(ui.Black)

	app.fwdBtn = elements.ButtonAt(btnFwd, ">", fwdX, btnY)
	app.fwdBtn.SetSize(buttonW, buttonH)
	app.fwdBtn.SetPadding(0, 8)
	app.fwdBtn.SetBackground(ui.Silver)
	app.fwdBtn.SetForeground(ui.Black)

	app.goBtn = elements.ButtonAt(btnGo, "Go", goX, btnY)
	app.goBtn.SetSize(goButtonW, buttonH)
	app.goBtn.SetPadding(0, 6)
	app.goBtn.SetBackground(ui.Silver)
	app.goBtn.SetForeground(ui.Black)

	editY := btnY + 2
	app.edit = kos.NewEditBox(editX, editY, editW, 256, app.url)
	app.edit.SetFlags(kos.EditBoxFlagAlwaysFocus | kos.EditBoxFlagFocus)

	app.contentX = clientX + padding
	app.contentY = clientY + toolbarH + padding
	app.contentW = clientW - padding*2 - scrollW
	app.contentH = clientH - toolbarH - statusH - padding*2
	if app.contentH < fontH {
		app.contentH = fontH
	}
	if app.contentW < fontW {
		app.contentW = fontW
	}

	scrollX := app.contentX + app.contentW
	app.textCols = (app.contentW - textGap) / fontW
	if app.textCols < 10 {
		app.textCols = 10
	}

	app.scroll = kos.NewVerticalScrollBar(scrollX, app.contentY, scrollW, app.contentH, 0, 0, 0)
}

func (app *App) Redraw() {
	sizeChanged := app.syncWindowInfo()
	oldContentW := app.contentW
	app.layout()
	if sizeChanged && app.contentW != oldContentW {
		app.reflow()
	}

	kos.BeginRedraw()
	kos.OpenWindow(app.x, app.y, app.width, app.height, app.windowTitle())

	kos.FillRect(app.clientX, app.clientY, app.clientW, app.clientH, ui.White)
	kos.FillRect(app.clientX, app.clientY, app.clientW, toolbarH, ui.Silver)

	app.backBtn.Draw()
	app.fwdBtn.Draw()
	app.goBtn.Draw()
	_ = app.boxlib.DrawEditBox(app.edit)

	kos.FillRect(app.contentX, app.contentY, app.contentW, app.contentH, ui.White)

	visible := app.visibleLines()
	app.clampFirstLine(visible)
	app.scroll.SetRange(len(app.lines), visible)
	app.scroll.SetPosition(app.firstLine)
	if app.pageBuf != nil {
		offsetY := app.firstLine * fontH
		app.pageBuf.show(app.contentX, app.contentY, offsetY, app.contentH)
		app.drawHoverOverlay(visible)
	} else {
		for i := 0; i < visible; i++ {
			index := app.firstLine + i
			if index >= len(app.lines) {
				break
			}
			app.drawLine(app.lines[index], app.contentX, app.contentY+i*fontH)
		}
	}

	_ = app.boxlib.DrawVerticalScrollBar(app.scroll)

	statusY := app.clientY + app.clientH - statusH
	kos.FillRect(app.clientX, statusY, app.clientW, statusH, ui.Silver)
	status := app.statusBase
	if app.hoverLink >= 0 && app.hoverLink < len(app.links) {
		status = app.links[app.hoverLink]
	}
	kos.DrawText(app.clientX+padding, statusY+1, ui.Black, status)

	kos.EndRedraw()
	app.windowReady = true
}

func (app *App) syncWindowInfo() bool {
	info, _, ok := kos.ReadCurrentThreadInfo()
	if !ok {
		return false
	}
	if info.WindowSize.X <= 0 || info.WindowSize.Y <= 0 {
		return false
	}
	if info.WindowSize.X == app.width && info.WindowSize.Y == app.height {
		return false
	}
	app.width = info.WindowSize.X
	app.height = info.WindowSize.Y
	return true
}

func (app *App) reflow() {
	if len(app.runs) == 0 {
		app.buildPageBuffer()
		return
	}
	offsetY := app.firstLine * fontH
	lines := wrapRuns(app.runs, app.textCols)
	if len(lines) == 0 {
		lines = []RenderLine{{Text: ""}}
	}
	app.lines = lines
	app.hoverLink = -1
	app.firstLine = offsetY / fontH
	app.clampFirstLine(app.visibleLines())
	app.buildPageBuffer()
}

func (app *App) redrawContent() {
	if !app.windowReady || app.scroll == nil {
		return
	}

	visible := app.visibleLines()
	app.clampFirstLine(visible)
	app.scroll.SetRange(len(app.lines), visible)
	app.scroll.SetPosition(app.firstLine)

	if app.pageBuf != nil {
		offsetY := app.firstLine * fontH
		app.pageBuf.show(app.contentX, app.contentY, offsetY, app.contentH)
		app.drawHoverOverlay(visible)
	} else {
		kos.FillRect(app.contentX, app.contentY, app.contentW, app.contentH, ui.White)
		for i := 0; i < visible; i++ {
			index := app.firstLine + i
			if index >= len(app.lines) {
				break
			}
			app.drawLine(app.lines[index], app.contentX, app.contentY+i*fontH)
		}
	}

	_ = app.boxlib.DrawVerticalScrollBar(app.scroll)
	app.redrawStatus()
}

func (app *App) redrawStatus() {
	if !app.windowReady {
		return
	}
	statusY := app.clientY + app.clientH - statusH
	kos.FillRect(app.clientX, statusY, app.clientW, statusH, ui.Silver)
	status := app.statusBase
	if app.hoverLink >= 0 && app.hoverLink < len(app.links) {
		status = app.links[app.hoverLink]
	}
	kos.DrawText(app.clientX+padding, statusY+1, ui.Black, status)
}

func (app *App) redrawEdit() {
	if !app.windowReady {
		return
	}
	_ = app.boxlib.DrawEditBox(app.edit)
}

func (app *App) handleKey() {
	event := kos.ReadKey()
	if event.Empty || event.Hotkey {
		return
	}

	app.boxlib.HandleEditBoxKey(app.edit, uint32(event.Raw))
	app.redrawEdit()

	if event.Code == 13 {
		app.submitURL()
	}
	if event.Code == 27 || event.ScanCode == 1 {
		kos.Exit()
	}
}

func (app *App) handleMouse() {
	app.boxlib.HandleEditBoxMouse(app.edit)
	app.redrawEdit()
	app.boxlib.HandleVerticalScrollBarMouse(app.scroll)

	newPos := app.scroll.Position()
	if newPos != app.firstLine {
		app.firstLine = newPos
		app.redrawContent()
	}

	delta := kos.MouseScrollDelta().Y
	if delta != 0 {
		app.scrollBy(-delta * 3)
	}

	pos := kos.MouseWindowPosition()
	link := app.linkAt(pos.X, pos.Y)
	if link != app.hoverLink {
		app.hoverLink = link
		app.redrawContent()
	}

	buttons := kos.MouseButtons()
	if buttons.LeftPressed {
		if link >= 0 && link < len(app.links) {
			app.openURL(app.links[link], true)
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case 1:
		kos.Exit()
		return true
	case btnBack:
		app.goBack()
	case btnFwd:
		app.goForward()
	case btnGo:
		app.submitURL()
	}
	return false
}

func (app *App) submitURL() {
	url := strings.TrimSpace(app.edit.Text())
	if url == "" {
		return
	}
	app.openURL(url, true)
}

func (app *App) openURL(url string, push bool) {
	url = normalizeURL(url)
	app.setURL(url)
	if push {
		app.pushHistory(url)
	}
	app.startLoad(url)
}

func (app *App) setURL(url string) {
	app.url = url
	if app.edit != nil {
		app.edit.SetText(url)
		app.redrawEdit()
	}
}

func (app *App) pushHistory(url string) {
	if app.historyIndex+1 < len(app.history) {
		app.history = app.history[:app.historyIndex+1]
	}
	app.history = append(app.history, url)
	app.historyIndex = len(app.history) - 1
}

func (app *App) goBack() {
	if app.historyIndex <= 0 {
		return
	}
	app.historyIndex--
	app.openURL(app.history[app.historyIndex], false)
}

func (app *App) goForward() {
	if app.historyIndex+1 >= len(app.history) {
		return
	}
	app.historyIndex++
	app.openURL(app.history[app.historyIndex], false)
}

func (app *App) startLoad(url string) {
	if !app.http.Ready() {
		app.statusBase = "HTTP unavailable"
		app.runs = []Run{{Text: "Missing /sys/lib/http.obj", Link: -1}}
		app.lines = wrapRuns(app.runs, app.textCols)
		if len(app.lines) == 0 {
			app.lines = []RenderLine{{Text: ""}}
		}
		app.links = nil
		app.buildPageBuffer()
		app.redrawContent()
		return
	}

	if app.loading {
		app.http.Disconnect(app.transfer)
		app.http.Free(app.transfer)
	}

	transfer, ok := app.http.Get(url, 0, kos.HTTPFlagHTTP11, "")
	if !ok || !transfer.Valid() {
		app.statusBase = "HTTP start failed"
		app.runs = []Run{{Text: "Failed to start HTTP GET.", Link: -1}}
		app.lines = wrapRuns(app.runs, app.textCols)
		if len(app.lines) == 0 {
			app.lines = []RenderLine{{Text: ""}}
		}
		app.links = nil
		app.loading = false
		app.buildPageBuffer()
		app.redrawContent()
		return
	}

	app.transfer = transfer
	app.loading = true
	app.lastReceived = 0
	app.statusBase = "Loading..."
	app.runs = []Run{{Text: "Loading...", Link: -1}}
	app.lines = wrapRuns(app.runs, app.textCols)
	if len(app.lines) == 0 {
		app.lines = []RenderLine{{Text: ""}}
	}
	app.links = nil
	app.firstLine = 0
	app.buildPageBuffer()
	app.redrawContent()
}

func (app *App) pollHTTP() {
	if !app.loading || !app.transfer.Valid() {
		return
	}

	_ = app.http.Receive(app.transfer)
	received := app.transfer.ContentReceived()
	total := app.transfer.ContentLength()
	if received != app.lastReceived {
		app.lastReceived = received
		app.statusBase = formatProgress(received, total)
		app.redrawStatus()
	}

	flags := app.transfer.Flags()
	if flags.Has(kos.HTTPFlagGotAllData) ||
		flags.Has(kos.HTTPFlagTransferFailed) ||
		flags.Has(kos.HTTPFlagTimeoutError) ||
		flags.Has(kos.HTTPFlagSocketError) {
		app.finishTransfer()
	}
}

func (app *App) finishTransfer() {
	flags := app.transfer.Flags()
	status := app.transfer.Status()
	header := app.transfer.HeaderString()
	body := app.transfer.ContentBytes()

	app.http.Free(app.transfer)
	app.transfer = 0
	app.loading = false

	if flags.Has(kos.HTTPFlagTransferFailed) || flags.Has(kos.HTTPFlagTimeoutError) || flags.Has(kos.HTTPFlagSocketError) {
		app.statusBase = "Load failed"
		app.runs = []Run{{Text: "Network error.", Link: -1}}
		app.lines = wrapRuns(app.runs, app.textCols)
		if len(app.lines) == 0 {
			app.lines = []RenderLine{{Text: ""}}
		}
		app.links = nil
		app.buildPageBuffer()
		app.redrawContent()
		return
	}

	if status >= 400 {
		app.statusBase = "HTTP " + formatUint(status)
		app.runs = []Run{{Text: "HTTP error " + formatUint(status), Link: -1}}
		app.lines = wrapRuns(app.runs, app.textCols)
		if len(app.lines) == 0 {
			app.lines = []RenderLine{{Text: ""}}
		}
		app.links = nil
		app.buildPageBuffer()
		app.redrawContent()
		return
	}

	app.updateContent(header, body)
	app.redrawContent()
}

func (app *App) updateContent(header string, body []byte) {
	if !isTextContent(header) {
		app.statusBase = "Unsupported content type"
		app.runs = []Run{{Text: "Content is not text.", Link: -1}}
		app.lines = wrapRuns(app.runs, app.textCols)
		if len(app.lines) == 0 {
			app.lines = []RenderLine{{Text: ""}}
		}
		app.links = nil
		app.buildPageBuffer()
		return
	}

	if len(body) > maxContent {
		body = body[:maxContent]
		app.statusBase = "Loaded (truncated)"
	} else {
		app.statusBase = "Loaded"
	}

	doc := dom.Parse(string(body))
	app.pageTitle = documentTitle(doc)
	kos.SetWindowTitle(app.windowTitle())
	runs, links := renderDocument(doc, app.url)
	app.runs = runs
	lines := wrapRuns(app.runs, app.textCols)
	if len(lines) == 0 {
		lines = []RenderLine{{Text: ""}}
	}
	app.lines = lines
	app.links = links
	app.firstLine = 0
	app.hoverLink = -1
	app.buildPageBuffer()
}

func (app *App) visibleLines() int {
	if app.contentH <= 0 {
		return 0
	}
	return app.contentH / fontH
}

func (app *App) clampFirstLine(visible int) {
	if visible < 0 {
		visible = 0
	}
	max := len(app.lines) - visible
	if max < 0 {
		max = 0
	}
	if app.firstLine < 0 {
		app.firstLine = 0
	}
	if app.firstLine > max {
		app.firstLine = max
	}
}

func (app *App) scrollBy(delta int) {
	if delta == 0 {
		return
	}
	visible := app.visibleLines()
	app.firstLine += delta
	app.clampFirstLine(visible)
	app.scroll.SetRange(len(app.lines), visible)
	app.scroll.SetPosition(app.firstLine)
	app.redrawContent()
}

func (app *App) drawLine(line RenderLine, x int, y int) {
	if len(line.Text) == 0 {
		return
	}

	if len(line.Spans) == 0 {
		kos.DrawText(x, y, ui.Black, line.Text)
		return
	}

	pos := 0
	for _, span := range line.Spans {
		if span.Start > pos {
			kos.DrawText(x+pos*fontW, y, ui.Black, line.Text[pos:span.Start])
		}
		color := ui.Blue
		if span.Link == app.hoverLink {
			color = ui.Red
		}
		kos.DrawText(x+span.Start*fontW, y, color, line.Text[span.Start:span.End])
		pos = span.End
	}
	if pos < len(line.Text) {
		kos.DrawText(x+pos*fontW, y, ui.Black, line.Text[pos:])
	}
}

func (app *App) linkAt(x int, y int) int {
	if x < app.contentX || y < app.contentY {
		return -1
	}
	if x >= app.contentX+app.contentW || y >= app.contentY+app.contentH {
		return -1
	}
	line := (y - app.contentY) / fontH
	index := app.firstLine + line
	if index < 0 || index >= len(app.lines) {
		return -1
	}
	col := (x - app.contentX) / fontW
	for _, span := range app.lines[index].Spans {
		if col >= span.Start && col < span.End {
			return span.Link
		}
	}
	return -1
}

func formatProgress(received uint32, total uint32) string {
	if total == 0 {
		return "Loading " + formatUint(received)
	}
	return "Loading " + formatUint(received) + "/" + formatUint(total)
}

func formatUint(value uint32) string {
	if value == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}

func (app *App) windowTitle() string {
	if app.pageTitle == "" {
		return "Go WebView (lite)"
	}
	return "Go WebView (lite) - " + app.pageTitle
}

func documentTitle(doc *dom.Document) string {
	if doc == nil {
		return ""
	}
	nodes := doc.GetElementsByTagName("title")
	if len(nodes) == 0 {
		return ""
	}
	text := collectText(nodes[0])
	return strings.TrimSpace(text)
}

func collectText(node *dom.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == dom.TextNode {
		return node.Text
	}
	var builder strings.Builder
	for _, child := range node.Children {
		builder.WriteString(collectText(child))
	}
	return builder.String()
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

func isTextContent(header string) bool {
	if header == "" {
		return true
	}
	lower := toLowerASCII(header)
	index := strings.Index(lower, "content-type:")
	if index < 0 {
		return true
	}
	line := lower[index:]
	if end := indexByte(line, '\n'); end >= 0 {
		line = line[:end]
	}
	return strings.Contains(line, "text/") ||
		strings.Contains(line, "json") ||
		strings.Contains(line, "xml") ||
		strings.Contains(line, "javascript")
}

func main() {
	app := NewApp()
	app.Run()
}
