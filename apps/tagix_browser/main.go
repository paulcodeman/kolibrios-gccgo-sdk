package main

import (
	"dom"
	"kos"
	neturl "net/url"
	"os"
	"strings"
	"ui"
)

const (
	defaultWindowWidth  = 780
	defaultWindowHeight = 560
	defaultShellHeight  = 86
	defaultPageHeight   = 420
	rootInset           = 0
	shellGap            = 0
	minPageHeight       = 180
	maxContent          = 512 * 1024
	defaultURL          = "about:tagix"
	aboutFormsURL       = "about:forms"
	aboutHomeAsset      = "assets/about_tagix.html"
	aboutFormsAsset     = "assets/about_forms.html"
)

const defaultAboutHomeHTML = `<html><head><title>Tagix Browser</title></head><body><h1>Tagix Browser</h1><p>This built-in page is rendered through the same HTML5 parser and DocumentView pipeline as remote content.</p><p><a href="about:forms">Open the built-in forms demo</a> or visit <a href="https://example.com">example.com</a>.</p><h2>What to test</h2><ul><li>Inline links inside paragraphs</li><li>Preformatted code blocks</li><li>Lists and headings</li><li>Document focus, hover and scroll</li></ul><pre><code>&lt;html&gt; -&gt; dom.Parse -&gt; ui.DocumentNode -&gt; DocumentView</code></pre></body></html>`

const defaultAboutFormsHTML = `<html><head><title>Tagix Forms</title></head><body><h1>Tagix Forms</h1><p>This page exists to test browser-side HTML controls that are currently mapped onto the shared UI pipeline.</p><p><a href="about:tagix">Back to the built-in home page</a></p><p>Submit keeps you on built-in pages by targeting <code>about:tagix</code>; after submit, the address bar should include the serialized query string.</p><form action="about:tagix" method="get"><input type="hidden" name="source" value="about:forms"><h2>Text controls</h2><p><input type="text" name="url" value="https://kolibrios.org" placeholder="Type a URL"></p><p><input type="search" name="query" placeholder="Search demo"></p><p>Textarea currently submits its initial content; multiline editing is still pending in the shared DocumentView host.</p><textarea name="notes" rows="4">Textarea fallback content.
Second line.
Third line.</textarea><h2>Choice controls</h2><p><input type="checkbox" name="remember" checked value="1"></p><p><input type="checkbox" name="compact" value="1"></p><p><input type="radio" name="theme" checked value="ocean"></p><p><input type="radio" name="theme" value="sunset"></p><p><select name="mode"><option selected value="first">First option</option><option value="second">Second option</option><option value="third">Third option</option></select></p><h2>Range and progress</h2><p><input type="range" name="level" min="0" max="10" step="2" value="4"></p><p><progress value="42" max="100"></progress></p><h2>Buttons</h2><p><button type="submit" name="submitter" value="button">Submit form</button></p><p><input type="reset" value="Reset form"></p></form></body></html>`

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
		style.SetBackground(0xE7EBF0)
	})
	window.CenterOnScreen()
	app.window = window

	root := ui.CreateBox()
	root.UpdateStyle(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(rootInset)
		style.SetBackground(0xF1F3F4)
	})

	app.shellDocument = ui.NewDocument(renderShellRoot(app))
	app.shellView = ui.CreateDocumentView(app.shellDocument)
	app.shellView.Style = styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(defaultShellHeight)
		style.SetMargin(0, 0, shellGap, 0)
		style.SetPadding(8, 12)
		style.SetBorder(0, 0xF1F3F4)
		style.SetBorderBottom(1, 0xD2D7DD)
		style.SetBackground(0xF1F3F4)
		style.SetOverflow(ui.OverflowHidden)
		style.SetContain(ui.ContainPaint)
	})

	app.pageDocument = ui.NewDocument(nil)
	app.pageView = ui.CreateDocumentView(app.pageDocument)
	app.pageView.Style = styled(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(defaultPageHeight)
		style.SetPadding(0)
		style.SetBorder(0, ui.White)
		style.SetBorderRadius(0)
		style.SetBackground(ui.White)
		style.SetOverflow(ui.OverflowAuto)
		style.SetScrollbarWidth(8)
		style.SetScrollbarTrack(0xEDF0F2)
		style.SetScrollbarThumb(0xAAB2BC)
		style.SetScrollbarRadius(4)
		style.SetScrollbarPadding(1)
		style.SetContain(ui.ContainPaint)
		style.SetWillChange(ui.WillChangeScrollPosition)
	})
	app.pageView.StyleFocus = styled(func(style *ui.Style) {
		style.SetOutline(2, 0x1A73E8)
		style.SetOutlineOffset(1)
	})

	root.Append(app.shellView)
	root.Append(app.pageView)
	window.Append(root)
	window.OnResize = app.handleResize
	app.handleResize(ui.Rect{Height: defaultWindowHeight})
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
	if app.loadBuiltinPage(url) {
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

func (app *App) loadBuiltinPage(url string) bool {
	if app == nil {
		return false
	}
	pageTitle, status, html, ok := builtinPageSource(url)
	if !ok {
		return false
	}
	doc := dom.Parse(html)
	if parsedTitle := documentTitle(doc); parsedTitle != "" {
		pageTitle = parsedTitle
	}
	app.pageTitle = pageTitle
	app.pageDocument.SetRoot(buildRenderedDocument(app.pageTitle, app.currentURL, doc, func(target string) {
		app.openURL(target, true)
	}, func(actionURL string, method string, values neturl.Values) {
		app.submitForm(actionURL, method, values)
	}, func() {
		app.pageDocument.MarkLayoutDirty()
	}, func() {
		app.pageDocument.MarkDirty()
	}))
	app.statusBase = status
	return true
}

func builtinPageSource(url string) (string, string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(stripFragment(stripQuery(url)))) {
	case "about:tagix":
		return "Tagix Browser", "Built-in page", loadBuiltinAsset(aboutHomeAsset, defaultAboutHomeHTML), true
	case strings.ToLower(aboutFormsURL):
		return "Tagix Forms", "Built-in page", loadBuiltinAsset(aboutFormsAsset, defaultAboutFormsHTML), true
	default:
		return "", "", "", false
	}
}

func loadBuiltinAsset(path string, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return fallback
	}
	return string(data)
}

func (app *App) submitForm(actionURL string, method string, values neturl.Values) {
	if app == nil {
		return
	}
	actionURL = strings.TrimSpace(actionURL)
	if actionURL == "" {
		actionURL = app.currentURL
	}
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "" {
		method = "get"
	}
	switch method {
	case "get":
		encoded := ""
		if values != nil {
			encoded = values.Encode()
		}
		app.openURL(appendURLQuery(actionURL, encoded), true)
	default:
		app.pageTitle = "Unsupported form method"
		app.statusBase = strings.ToUpper(method) + " not supported"
		app.showMessageDocument("Unsupported form method", "This lite browser currently supports GET form submission only.")
		app.syncShell()
	}
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
	}, func(actionURL string, method string, values neturl.Values) {
		app.submitForm(actionURL, method, values)
	}, func() {
		app.pageDocument.MarkLayoutDirty()
	}, func() {
		app.pageDocument.MarkDirty()
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
	if app.pageTitle != "" && app.pageTitle != "Tagix Browser" {
		windowTitle += " - " + app.pageTitle
	}
	app.window.SetTitle(windowTitle)
}

func (app *App) handleResize(client ui.Rect) {
	if app == nil || app.pageView == nil || app.shellView == nil {
		return
	}
	pageHeight := client.Height - rootInset*2 - defaultShellHeight - shellGap
	if pageHeight < minPageHeight {
		pageHeight = minPageHeight
	}
	changed := false
	if current, ok := app.pageView.Style.GetHeight(); !ok || current != pageHeight {
		app.pageView.Style.SetHeight(pageHeight)
		changed = true
	}
	if current, ok := app.shellView.Style.GetHeight(); !ok || current != defaultShellHeight {
		app.shellView.Style.SetHeight(defaultShellHeight)
		changed = true
	}
	if changed {
		app.shellView.MarkLayoutDirty()
		app.pageView.MarkLayoutDirty()
	}
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
