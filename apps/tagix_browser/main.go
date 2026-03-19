package main

import (
	"dom"
	"kos"
	"strings"
	"ui"
)

const (
	defaultWindowWidth  = 780
	defaultWindowHeight = 560
	defaultShellHeight  = 172
	defaultPageHeight   = 328
	maxContent          = 512 * 1024
	defaultURL          = "https://example.com"
)

type App struct {
	window *ui.Window
	http   kos.HTTP

	shellDocument *ui.Document
	shellView     *ui.DocumentView
	pageDocument  *ui.Document
	pageView      *ui.DocumentView

	shellTitleNode   *ui.DocumentNode
	shellStatusNode  *ui.DocumentNode
	shellBackNode    *ui.DocumentNode
	shellForwardNode *ui.DocumentNode
	shellReloadNode  *ui.DocumentNode
	shellHomeNode    *ui.DocumentNode
	shellAddressNode *ui.DocumentNode

	currentURL   string
	addressText  string
	pageTitle    string
	statusBase   string
	history      []string
	historyIndex int
}

func NewApp() *App {
	http, _ := kos.LoadHTTP()
	app := &App{
		http:         http,
		statusBase:   "Ready",
		historyIndex: -1,
		addressText:  defaultURL,
	}
	app.buildUI()
	app.showMessageDocument(
		"Tagix Browser",
		"Browser shell now renders in its own DocumentView, and the page below is hosted in a separate frame-like DocumentView. The shell toolbar itself now comes from HTML, including the editable address field.",
	)
	app.syncShell()
	return app
}

func (app *App) buildUI() {
	window := ui.NewWindowDefault()
	window.SetTitle("Tagix Browser")
	window.UpdateStyle(func(style *ui.Style) {
		style.SetWidth(defaultWindowWidth)
		style.SetHeight(defaultWindowHeight)
		style.SetOverflow(ui.OverflowAuto)
		style.SetGradient(ui.Gradient{
			From:      ui.White,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		})
	})
	window.CenterOnScreen()
	app.window = window

	root := ui.CreateBox()
	root.UpdateStyle(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
	})

	app.shellDocument = ui.NewDocument(buildShellDocument(app))
	app.shellView = ui.CreateDocumentView(app.shellDocument)
	app.shellView.Style = styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(defaultShellHeight)
		style.SetMargin(0, 0, 10, 0)
		style.SetPadding(10)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(14)
		style.SetBackground(ui.White)
		style.SetOverflow(ui.OverflowHidden)
		style.SetContain(ui.ContainPaint)
	})

	app.pageDocument = ui.NewDocument(nil)
	app.pageView = ui.CreateDocumentView(app.pageDocument)
	app.pageView.Style = styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(defaultPageHeight)
		style.SetPadding(10)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(12)
		style.SetBackground(ui.White)
		style.SetOverflow(ui.OverflowAuto)
		style.SetScrollbarWidth(8)
		style.SetScrollbarTrack(ui.Silver)
		style.SetScrollbarThumb(ui.Gray)
		style.SetScrollbarRadius(4)
		style.SetScrollbarPadding(1)
		style.SetContain(ui.ContainPaint)
		style.SetWillChange(ui.WillChangeScrollPosition)
	})
	app.pageView.StyleFocus = styled(func(style *ui.Style) {
		style.SetBorderColor(ui.Blue)
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
	})

	root.Append(app.shellView)
	root.Append(app.pageView)
	window.Append(root)
}

func (app *App) Run() {
	app.openURL(defaultURL, true)
	app.window.Start()
}

func (app *App) submitAddress() {
	if app == nil {
		return
	}
	value := strings.TrimSpace(app.addressText)
	if value == "" {
		value = defaultURL
	}
	app.openURL(value, true)
}

func (app *App) reloadCurrent() {
	if app == nil {
		return
	}
	url := strings.TrimSpace(app.currentURL)
	if url == "" {
		url = defaultURL
	}
	app.openURL(url, false)
}

func (app *App) goHome() {
	if app == nil {
		return
	}
	app.openURL(defaultURL, true)
}

func (app *App) openURL(url string, push bool) {
	if app == nil {
		return
	}
	url = normalizeURL(url)
	app.currentURL = url
	app.addressText = url
	if push {
		app.pushHistory(url)
	}
	app.statusBase = "Loading"
	app.syncShell()
	app.loadURL(url)
	app.syncShell()
	if app.shellView != nil {
		app.shellView.MarkDirty()
	}
	if app.pageView != nil {
		app.pageView.MarkDirty()
	}
}

func (app *App) pushHistory(url string) {
	if app == nil || url == "" {
		return
	}
	if app.historyIndex+1 < len(app.history) {
		app.history = app.history[:app.historyIndex+1]
	}
	app.history = append(app.history, url)
	app.historyIndex = len(app.history) - 1
}

func (app *App) goBack() {
	if app == nil || app.historyIndex <= 0 {
		return
	}
	app.historyIndex--
	app.openURL(app.history[app.historyIndex], false)
}

func (app *App) goForward() {
	if app == nil || app.historyIndex+1 >= len(app.history) {
		return
	}
	app.historyIndex++
	app.openURL(app.history[app.historyIndex], false)
}

func (app *App) loadURL(url string) {
	if app == nil {
		return
	}
	if !app.http.Ready() {
		app.pageTitle = "HTTP unavailable"
		app.statusBase = "Missing /sys/lib/http.obj"
		app.showMessageDocument("HTTP unavailable", "The HTTP library is not available, so Tagix Browser cannot download the page.")
		return
	}

	transfer, ok := app.http.Get(url, 0, kos.HTTPFlagHTTP11, "")
	if !ok || !transfer.Valid() {
		app.pageTitle = "Load failed"
		app.statusBase = "HTTP start failed"
		app.showMessageDocument("Load failed", "Failed to start HTTP GET.")
		return
	}

	for {
		_ = app.http.Receive(transfer)
		flags := transfer.Flags()
		if flags.Has(kos.HTTPFlagGotAllData) ||
			flags.Has(kos.HTTPFlagTransferFailed) ||
			flags.Has(kos.HTTPFlagTimeoutError) ||
			flags.Has(kos.HTTPFlagSocketError) {
			break
		}
		kos.SleepMilliseconds(10)
	}

	flags := transfer.Flags()
	status := transfer.Status()
	header := transfer.HeaderString()
	body := transfer.ContentBytes()
	app.http.Free(transfer)

	if flags.Has(kos.HTTPFlagTransferFailed) || flags.Has(kos.HTTPFlagTimeoutError) || flags.Has(kos.HTTPFlagSocketError) {
		app.pageTitle = "Load failed"
		app.statusBase = "Network error"
		app.showMessageDocument("Load failed", "Network error while downloading the page.")
		return
	}

	if status >= 400 {
		app.pageTitle = "HTTP " + formatUint(status)
		app.statusBase = "HTTP " + formatUint(status)
		app.showMessageDocument("HTTP error", "Server returned status "+formatUint(status)+".")
		return
	}

	app.updateContent(header, body)
}

func (app *App) updateContent(header string, body []byte) {
	if app == nil {
		return
	}
	if !isTextContent(header) {
		app.pageTitle = "Unsupported content"
		app.statusBase = "Content is not text"
		app.showMessageDocument("Unsupported content", "This lite browser currently renders only text-like responses.")
		return
	}

	truncated := false
	if len(body) > maxContent {
		body = body[:maxContent]
		truncated = true
	}

	doc := dom.Parse(string(body))
	app.pageTitle = documentTitle(doc)
	app.pageDocument.SetRoot(buildRenderedDocument(app.pageTitle, app.currentURL, doc, func(target string) {
		app.openURL(target, true)
	}))
	if truncated {
		app.statusBase = "Loaded (truncated)"
	} else {
		app.statusBase = "Loaded"
	}
}

func (app *App) showMessageDocument(title string, detail string) {
	if app == nil || app.pageDocument == nil {
		return
	}
	app.pageDocument.SetRoot(buildMessageDocument(title, detail))
	if app.pageView != nil {
		app.pageView.MarkDirty()
	}
}

func (app *App) syncShell() {
	if app == nil || app.shellDocument == nil {
		return
	}
	title := strings.TrimSpace(app.pageTitle)
	if title == "" {
		title = "Tagix Browser"
	}
	status := strings.TrimSpace(app.statusBase)
	if status == "" {
		status = "Ready"
	}
	syncShellDocument(app, title, status)
	windowTitle := "Tagix Browser"
	if app.pageTitle != "" {
		windowTitle += " - " + app.pageTitle
	}
	app.window.SetTitle(windowTitle)
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

func documentTitle(doc *dom.Document) string {
	if doc == nil {
		return ""
	}
	nodes := doc.GetElementsByTagName("title")
	if len(nodes) == 0 {
		return ""
	}
	return strings.TrimSpace(collectText(nodes[0]))
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
