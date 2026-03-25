package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"golang.org/x/net/publicsuffix"
	"io"
	"kos"
	nethttp "net/http"
	netcookiejar "net/http/cookiejar"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"ui"
)

const (
	defaultWindowWidth  = 780
	defaultWindowHeight = 560
	defaultPageHeight   = 420
	rootInset           = 0
	shellGap            = 0
	minPageHeight       = 180
	maxContent          = 512 * 1024
	defaultURL          = "about:tagix"
	aboutFormsURL       = "about:forms"
	aboutHomeAsset      = "assets/about_tagix.html"
	aboutFormsAsset     = "assets/about_forms.html"
	localCABundleAsset  = "assets/ca-bundle.pem"
)

const defaultAboutHomeHTML = `<html><head><title>Tagix Browser</title></head><body><h1>Tagix Browser</h1><p>Browser chrome now loads from <code>assets/shell.html</code>, and page content is hosted in a dedicated iframe-like frame below the toolbar.</p><p><a href="about:forms">Open the built-in forms demo</a> or visit <a href="https://example.com">example.com</a>.</p><h2>What to test</h2><ul><li>Semantic shell tags: header, nav, section, h1, small, iframe</li><li>Shell styling through inline HTML5 <code>style</code> attributes</li><li>Inline links, lists, headings and forms in rendered pages</li><li>Document focus, hover and scroll inside the hosted page view</li></ul><details><summary>Pipeline</summary><p><code>shell.html -&gt; Parse -&gt; ui.DocumentNode -&gt; DocumentView</code></p></details></body></html>`

const defaultAboutFormsHTML = `<html><head><title>Tagix Forms</title></head><body><h1>Tagix Forms</h1><p>This page exists to test browser-side HTML controls that are currently mapped onto the shared UI pipeline.</p><p><a href="about:tagix">Back to the built-in home page</a></p><p>Submit keeps you on built-in pages by targeting <code>about:tagix</code>; after submit, the address bar should include the serialized query string.</p><form action="about:tagix" method="get"><input type="hidden" name="source" value="about:forms"><h2>Text controls</h2><p><input type="text" name="url" value="https://kolibrios.org" placeholder="Type a URL"></p><p><input type="search" name="query" placeholder="Search demo"></p><p>Textarea currently submits its initial content; multiline editing is still pending in the shared DocumentView host.</p><textarea name="notes" rows="4">Textarea fallback content.
Second line.
Third line.</textarea><h2>Choice controls</h2><p><input type="checkbox" name="remember" checked value="1"></p><p><input type="checkbox" name="compact" value="1"></p><p><input type="radio" name="theme" checked value="ocean"></p><p><input type="radio" name="theme" value="sunset"></p><p><select name="mode"><option selected value="first">First option</option><option value="second">Second option</option><option value="third">Third option</option></select></p><h2>Range and progress</h2><p><input type="range" name="level" min="0" max="10" step="2" value="4"></p><p><progress value="42" max="100"></progress></p><h2>Buttons</h2><p><button type="submit" name="submitter" value="button">Submit form</button></p><p><input type="reset" value="Reset form"></p></form></body></html>`

type App struct {
	window *ui.Window

	httpClient    *nethttp.Client
	caBundlePath  string
	caBundleError string

	stylesheetCache  map[string]string
	imageCache       map[string]*ui.DocumentImage
	imageErrors      map[string]string
	resourceCacheDir string

	shellDocument *ui.Document
	shellView     *ui.DocumentView
	pageFrame     *ui.Element
	pageDocument  *ui.Document
	pageView      *ui.DocumentView

	shellNodesByID     map[string]*ui.DocumentNode
	shellNodesByRole   map[string][]*ui.DocumentNode
	shellNodesByAction map[string][]*ui.DocumentNode
	shellNodeDisplay   map[*ui.DocumentNode]ui.DisplayMode

	shellTitleNode   *ui.DocumentNode
	shellStatusNode  *ui.DocumentNode
	shellBackNode    *ui.DocumentNode
	shellForwardNode *ui.DocumentNode
	shellReloadNode  *ui.DocumentNode
	shellHomeNode    *ui.DocumentNode
	shellAddressNode *ui.DocumentNode

	currentURL    string
	addressText   string
	pageTitle     string
	statusBase    string
	renderDoc     *Document
	messageTitle  string
	messageDetail string
	renderWidth   int
	renderHeight  int
	history       []string
	historyIndex  int
	startupURL    string
	pageMinHeight int
	webViewMode   bool
}

func NewApp() *App {
	startupURL, webViewMode := resolveStartupTarget(os.Args[1:])
	httpClient, caBundlePath, caBundleError := newBrowserHTTPClient()
	app := &App{
		httpClient:       httpClient,
		caBundlePath:     caBundlePath,
		caBundleError:    caBundleError,
		stylesheetCache:  map[string]string{},
		imageCache:       map[string]*ui.DocumentImage{},
		imageErrors:      map[string]string{},
		resourceCacheDir: initResourceCacheDir(),
		statusBase:       "Ready",
		historyIndex:     -1,
		addressText:      defaultURL,
		startupURL:       startupURL,
		pageMinHeight:    minPageHeight,
		webViewMode:      webViewMode,
	}
	if strings.TrimSpace(caBundleError) != "" {
		app.debugf("tls root bundle: %s (path=%s)", strings.TrimSpace(caBundleError), strings.TrimSpace(caBundlePath))
	}
	app.buildUI()
	if !app.webViewMode {
		app.showMessageDocument(
			"Tagix Browser",
			"Browser shell now renders in its own DocumentView, and the page below is hosted in a separate frame-like DocumentView. The shell toolbar itself now comes from HTML, including the editable address field.",
		)
	}
	app.syncShell()
	return app
}

func (app *App) buildUI() {
	window := ui.NewWindowDefault()
	windowTitle := "Tagix Browser"
	windowBackground := kos.Color(0xE7EBF0)
	rootBackground := kos.Color(0xF1F3F4)
	pageFrameMarginTop := 10
	pageFrameBorder := 1
	pageFrameRadius := 16
	if app != nil && app.webViewMode {
		windowTitle = "Tagix WebView"
		windowBackground = ui.White
		rootBackground = ui.White
		pageFrameMarginTop = 0
		pageFrameBorder = 0
		pageFrameRadius = 0
	}
	window.SetTitle(windowTitle)
	window.UpdateStyle(func(style *ui.Style) {
		style.SetWidth(defaultWindowWidth)
		style.SetHeight(defaultWindowHeight)
		style.SetOverflow(ui.OverflowAuto)
		style.SetBackground(windowBackground)
	})
	window.CenterOnScreen()
	app.window = window

	root := ui.CreateBox()
	root.UpdateStyle(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(rootInset)
		style.SetBackground(rootBackground)
	})

	app.pageFrame = ui.CreateBox()
	app.pageFrame.UpdateStyle(func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(pageFrameMarginTop, 0, 0, 0)
		style.SetPadding(0)
		style.SetBorder(pageFrameBorder, 0xD7DEE7)
		style.SetBorderRadius(pageFrameRadius)
		style.SetBackground(ui.White)
		style.SetContain(ui.ContainPaint)
	})
	app.pageDocument = ui.NewDocument(nil)
	app.pageView = ui.CreateDocumentView(app.pageDocument)
	// The browser page currently repaints more reliably without DocumentView scroll-blit.
	app.pageView.DisableScrollBlit = true
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
	app.pageFrame.Append(app.pageView)

	app.shellDocument = ui.NewDocument(renderShellRoot(app))
	app.shellView = ui.CreateDocumentView(app.shellDocument)
	app.shellView.Style = styled(func(style *ui.Style) {
		if app != nil && app.webViewMode {
			style.SetDisplay(ui.DisplayNone)
		} else {
			style.SetDisplay(ui.DisplayBlock)
		}
		style.SetMargin(0, 0, shellGap, 0)
		style.SetPadding(0)
		style.SetBorder(0, rootBackground)
		style.SetBackground(rootBackground)
		style.SetOverflow(ui.OverflowHidden)
		style.SetContain(ui.ContainPaint)
	})

	root.Append(app.shellView)
	root.Append(app.pageFrame)
	window.Append(root)
	window.OnResize = app.handleResize
	app.handleResize(window.ClientRect())
}

func (app *App) Run() {
	startURL := strings.TrimSpace(app.startupURL)
	if startURL == "" {
		startURL = defaultURL
	}
	app.openURL(startURL, true)
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
	if app.loadLocalPage(url) {
		return
	}
	if app.httpClient == nil {
		app.debugf("http unavailable for %s", displayURL(url))
		app.pageTitle = "HTTP unavailable"
		app.statusBase = "HTTP client unavailable"
		app.showMessageDocument("HTTP unavailable", "The network client is not available, so Tagix Browser cannot download the page.")
		return
	}

	request, err := nethttp.NewRequest(nethttp.MethodGet, url, nil)
	if err != nil {
		app.debugError("http request build "+displayURL(url), err)
		app.pageTitle = "Load failed"
		app.statusBase = "Invalid URL"
		app.showMessageDocument("Load failed", "Failed to prepare the request: "+err.Error())
		return
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.5")
	request.Header.Set("User-Agent", "TagixBrowser/0.1")

	response, err := app.httpClient.Do(request)
	if err != nil {
		app.debugError("http get "+displayURL(url), err)
		app.pageTitle = "Load failed"
		app.statusBase = "Network error"
		app.showMessageDocument("Load failed", app.networkErrorDetail(url, err))
		return
	}
	defer response.Body.Close()

	finalURL := url
	if response.Request != nil && response.Request.URL != nil {
		if resolved := strings.TrimSpace(response.Request.URL.String()); resolved != "" {
			finalURL = normalizeURL(resolved)
		}
	}
	redirected := finalURL != "" && finalURL != url
	if finalURL != "" {
		app.currentURL = finalURL
		app.addressText = finalURL
		app.replaceCurrentHistory(finalURL)
	}

	if response.StatusCode >= 400 {
		app.debugf("http status %s for %s", response.Status, displayURL(finalURL))
		app.pageTitle = response.Status
		app.statusBase = response.Status
		app.showMessageDocument("HTTP error", app.httpErrorDetail(response.Status, finalURL, redirected))
		return
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxContent+1))
	if err != nil {
		app.debugError("http read body "+displayURL(finalURL), err)
		app.pageTitle = "Load failed"
		app.statusBase = "Read error"
		app.showMessageDocument("Load failed", "Failed while reading the response body: "+err.Error())
		return
	}

	app.updateContent(response.Header.Get("Content-Type"), body, redirected)
}

func newBrowserHTTPClient() (*nethttp.Client, string, string) {
	caBundlePath, _ := configureLocalCABundle()
	rootCAs, err := loadRootPool(caBundlePath)

	baseTransport, _ := nethttp.DefaultTransport.(*nethttp.Transport)
	transport := &nethttp.Transport{}
	if baseTransport != nil {
		*transport = *baseTransport
		if baseTransport.TLSClientConfig != nil {
			transport.TLSClientConfig = baseTransport.TLSClientConfig.Clone()
		}
	}
	if rootCAs != nil {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		} else {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	jar, jarErr := netcookiejar.New(&netcookiejar.Options{PublicSuffixList: publicsuffix.List})

	client := &nethttp.Client{
		Transport: transport,
	}
	if jarErr == nil {
		client.Jar = jar
	}
	if err != nil {
		return client, caBundlePath, err.Error()
	}
	return client, caBundlePath, ""
}

func configureLocalCABundle() (string, bool) {
	if value := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); value != "" {
		return value, true
	}
	if _, err := os.Stat(localCABundleAsset); err != nil {
		tagixDebugf("SSL_CERT_FILE stat %s: %v", localCABundleAsset, err)
		return "", false
	}
	if err := os.Setenv("SSL_CERT_FILE", localCABundleAsset); err != nil {
		tagixDebugf("SSL_CERT_FILE set %s: %v", localCABundleAsset, err)
		return "", false
	}
	return localCABundleAsset, true
}

func loadRootPool(path string) (*x509.CertPool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tagixDebugf("ca bundle read %s: %v", path, err)
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		tagixDebugf("ca bundle parse %s: no certificates", path)
		return nil, fmt.Errorf("no certificates parsed from %s", path)
	}
	return pool, nil
}

func (app *App) replaceCurrentHistory(url string) {
	if app == nil {
		return
	}
	if app.historyIndex < 0 || app.historyIndex >= len(app.history) {
		return
	}
	app.history[app.historyIndex] = url
}

func (app *App) networkErrorDetail(url string, err error) string {
	detail := "Network error while downloading " + displayURL(url) + "."
	if err != nil {
		detail += " " + err.Error()
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(url)), "https://") {
		if app != nil && strings.TrimSpace(app.caBundleError) != "" {
			detail += " TLS root bundle error: " + app.caBundleError + "."
		} else if app != nil && strings.TrimSpace(app.caBundlePath) != "" {
			detail += " TLS roots: " + app.caBundlePath + "."
		}
	}
	return detail
}

func (app *App) httpErrorDetail(status string, finalURL string, redirected bool) string {
	status = strings.TrimSpace(status)
	if status == "" {
		status = "HTTP error"
	}
	if redirected && strings.TrimSpace(finalURL) != "" {
		return "The request was redirected to " + displayURL(finalURL) + ", then the server returned " + status + "."
	}
	return "Server returned " + status + "."
}

func (app *App) loadBuiltinPage(url string) bool {
	if app == nil {
		return false
	}
	pageTitle, status, html, ok := builtinPageSource(url)
	if !ok {
		return false
	}
	doc := Parse(html)
	app.inlineLinkedStylesheets(doc, url)
	if parsedTitle := documentTitle(doc); parsedTitle != "" {
		pageTitle = parsedTitle
	}
	app.pageTitle = pageTitle
	app.showRenderedDocument(doc)
	app.statusBase = status
	return true
}

func (app *App) loadLocalPage(url string) bool {
	if app == nil {
		return false
	}
	path, ok := localHTMLPathFromURL(url)
	if !ok {
		return false
	}
	body, err := os.ReadFile(path)
	if err != nil {
		app.debugError("local html read "+path, err)
		app.pageTitle = "Load failed"
		app.statusBase = "File error"
		app.showMessageDocument("Load failed", "Failed to read local HTML file: "+path+". "+err.Error())
		return true
	}
	canonicalURL := fileURLFromPath(path)
	doc := Parse(string(body))
	app.inlineLinkedStylesheets(doc, canonicalURL)
	app.currentURL = canonicalURL
	app.addressText = canonicalURL
	app.replaceCurrentHistory(canonicalURL)
	app.pageTitle = documentTitle(doc)
	if strings.TrimSpace(app.pageTitle) == "" {
		app.pageTitle = filepath.Base(path)
	}
	if strings.TrimSpace(app.pageTitle) == "" {
		app.pageTitle = "Local HTML"
	}
	app.showRenderedDocument(doc)
	app.statusBase = "Loaded (local file)"
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
		app.debugf("unsupported form method %s for %s", strings.ToUpper(method), displayURL(actionURL))
		app.pageTitle = "Unsupported form method"
		app.statusBase = strings.ToUpper(method) + " not supported"
		app.showMessageDocument("Unsupported form method", "This lite browser currently supports GET form submission only.")
		app.syncShell()
	}
}

func (app *App) updateContent(contentType string, body []byte, redirected bool) {
	if app == nil {
		return
	}
	if !isTextContentType(contentType) {
		app.debugf("unsupported content-type %q at %s", strings.TrimSpace(contentType), displayURL(app.currentURL))
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

	doc := Parse(string(body))
	app.inlineLinkedStylesheets(doc, app.currentURL)
	app.pageTitle = documentTitle(doc)
	app.showRenderedDocument(doc)
	app.statusBase = loadedStatus(truncated, redirected)
}

func (app *App) showMessageDocument(title string, detail string) {
	if app == nil || app.pageDocument == nil {
		return
	}
	app.renderDoc = nil
	app.messageTitle = title
	app.messageDetail = detail
	viewportWidth, viewportHeight := app.pageViewportSize()
	app.renderWidth = viewportWidth
	app.renderHeight = viewportHeight
	app.pageDocument.SetRoot(buildMessageDocument(title, detail))
	if app.pageFrame != nil {
		app.pageFrame.Style.SetBackground(ui.White)
	}
	if app.pageView != nil {
		app.pageView.Style.SetBackground(ui.White)
	}
	if app.pageView != nil {
		app.pageView.MarkDirty()
	}
}

func (app *App) showRenderedDocument(doc *Document) {
	if app == nil || app.pageDocument == nil || doc == nil {
		return
	}
	app.renderDoc = doc
	app.messageTitle = ""
	app.messageDetail = ""
	viewportWidth, viewportHeight := app.pageViewportSize()
	app.renderWidth = viewportWidth
	app.renderHeight = viewportHeight
	app.applyDocumentViewportStyle(doc, viewportWidth, viewportHeight)
	app.pageDocument.SetRoot(buildRenderedDocument(app.pageTitle, app.currentURL, doc, viewportWidth, viewportHeight, func(target string) {
		app.openURL(target, true)
	}, func(actionURL string, method string, values neturl.Values) {
		app.submitForm(actionURL, method, values)
	}, func(rawURL string) *ui.DocumentImage {
		return app.loadDocumentImage(rawURL)
	}, func(rawURL string) string {
		return app.imageErrors[strings.TrimSpace(rawURL)]
	}, func() {
		app.pageDocument.MarkLayoutDirty()
	}, func() {
		app.pageDocument.MarkDirty()
	}))
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
	if app.webViewMode {
		windowTitle = title
		if strings.TrimSpace(windowTitle) == "" {
			windowTitle = "Tagix WebView"
		}
	} else if app.pageTitle != "" && app.pageTitle != "Tagix Browser" {
		windowTitle += " - " + app.pageTitle
	}
	if app.window != nil {
		app.window.SetTitle(windowTitle)
	}
}

func (app *App) handleResize(client ui.Rect) {
	if app == nil || app.pageView == nil || app.shellView == nil {
		return
	}
	shellHeight := app.shellHeightForClient(client)
	pageHeight := client.Height - rootInset*2 - shellHeight - shellGap - app.pageFrameVerticalChrome()
	if minHeight := app.effectivePageMinHeight(); pageHeight < minHeight {
		pageHeight = minHeight
	}
	changed := false
	if current, ok := app.pageView.Style.GetHeight(); !ok || current != pageHeight {
		app.pageView.Style.SetHeight(pageHeight)
		changed = true
	}
	if changed {
		app.shellView.MarkLayoutDirty()
		app.pageView.MarkLayoutDirty()
	}
	app.rerenderForViewportChange()
}

func (app *App) shellHeightForClient(client ui.Rect) int {
	if app == nil || app.shellView == nil {
		return 0
	}
	if display, ok := app.shellView.Style.GetDisplay(); ok && display == ui.DisplayNone {
		return 0
	}
	width := client.Width - rootInset*2
	if width < 0 {
		width = 0
	}
	height := client.Height - rootInset*2
	if height < 0 {
		height = 0
	}
	app.shellView.LayoutWithContext(ui.DefaultLayoutContext(ui.Rect{
		X:      0,
		Y:      0,
		Width:  width,
		Height: height,
	}))
	if bounds := app.shellView.Bounds(); bounds.Height > 0 {
		return bounds.Height
	}
	fallback := 0
	if lineHeight, ok := app.shellView.Style.GetLineHeight(); ok {
		fallback += lineHeight
	} else {
		fallback += 18
	}
	if padding, ok := app.shellView.Style.GetPadding(); ok {
		fallback += padding.Top + padding.Bottom
	}
	if border, ok := app.shellView.Style.GetBorderTopWidth(); ok {
		fallback += border
	}
	if border, ok := app.shellView.Style.GetBorderBottomWidth(); ok {
		fallback += border
	}
	if fallback <= 0 {
		return 40
	}
	return fallback
}

func (app *App) effectivePageMinHeight() int {
	if app == nil || app.pageMinHeight < minPageHeight {
		return minPageHeight
	}
	return app.pageMinHeight
}

func (app *App) pageFrameVerticalChrome() int {
	if app == nil || app.pageFrame == nil {
		return 0
	}
	total := 0
	if margin, ok := app.pageFrame.Style.GetMargin(); ok {
		total += margin.Top + margin.Bottom
	}
	if padding, ok := app.pageFrame.Style.GetPadding(); ok {
		total += padding.Top + padding.Bottom
	}
	if border, ok := app.pageFrame.Style.GetBorderTopWidth(); ok {
		total += border
	}
	if border, ok := app.pageFrame.Style.GetBorderBottomWidth(); ok {
		total += border
	}
	return total
}

func (app *App) pageFrameHorizontalChrome() int {
	if app == nil || app.pageFrame == nil {
		return 0
	}
	total := 0
	if margin, ok := app.pageFrame.Style.GetMargin(); ok {
		total += margin.Left + margin.Right
	}
	if padding, ok := app.pageFrame.Style.GetPadding(); ok {
		total += padding.Left + padding.Right
	}
	if border, ok := app.pageFrame.Style.GetBorderLeftWidth(); ok {
		total += border
	}
	if border, ok := app.pageFrame.Style.GetBorderRightWidth(); ok {
		total += border
	}
	return total
}

func (app *App) pageViewportSize() (int, int) {
	width := defaultWindowWidth - rootInset*2 - 2
	height := defaultPageHeight
	if app != nil && app.window != nil {
		client := app.window.ClientRect()
		width = client.Width - rootInset*2 - app.pageFrameHorizontalChrome()
	}
	if app != nil && app.pageView != nil {
		if current, ok := app.pageView.Style.GetHeight(); ok && current > 0 {
			height = current
		}
	}
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return width, height
}

func (app *App) applyDocumentViewportStyle(doc *Document, viewportWidth int, viewportHeight int) {
	if app == nil {
		return
	}
	canvasStyle := documentCanvasStyle(doc, viewportWidth, viewportHeight)
	background := ui.White
	if color, ok := canvasStyle.GetBackground(); ok {
		background = color
	}
	if app.pageFrame != nil {
		app.pageFrame.Style.SetBackground(background)
	}
	if app.pageView != nil {
		app.pageView.Style.SetBackground(background)
	}
}

func (app *App) rerenderForViewportChange() {
	if app == nil || app.pageDocument == nil {
		return
	}
	viewportWidth, viewportHeight := app.pageViewportSize()
	if viewportWidth == app.renderWidth && viewportHeight == app.renderHeight {
		return
	}
	app.renderWidth = viewportWidth
	app.renderHeight = viewportHeight
	if app.renderDoc != nil {
		app.showRenderedDocument(app.renderDoc)
	} else if app.messageTitle != "" || app.messageDetail != "" {
		app.showMessageDocument(app.messageTitle, app.messageDetail)
	}
	if app.pageView != nil {
		app.pageView.MarkLayoutDirty()
		app.pageView.MarkDirty()
	}
}

func documentTitle(doc *Document) string {
	if doc == nil {
		return ""
	}
	nodes := doc.GetElementsByTagName("title")
	if len(nodes) == 0 {
		return ""
	}
	return strings.TrimSpace(collectText(nodes[0]))
}

func collectText(node *Node) string {
	if node == nil {
		return ""
	}
	if node.Type == TextNode {
		return node.Text
	}
	var builder strings.Builder
	for _, child := range node.Children {
		builder.WriteString(collectText(child))
	}
	return builder.String()
}

func isTextContentType(contentType string) bool {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return true
	}
	lower := toLowerASCII(contentType)
	return strings.Contains(lower, "text/") ||
		strings.Contains(lower, "json") ||
		strings.Contains(lower, "xml") ||
		strings.Contains(lower, "javascript")
}

func loadedStatus(truncated bool, redirected bool) string {
	switch {
	case truncated && redirected:
		return "Loaded (redirected, truncated)"
	case truncated:
		return "Loaded (truncated)"
	case redirected:
		return "Loaded (redirected)"
	default:
		return "Loaded"
	}
}

func main() {
	app := NewApp()
	app.Run()
}
