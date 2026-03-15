package main

import (
	"fmt"
	"net/url"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	urlButtonExit    kos.ButtonID = 1
	urlButtonRefresh kos.ButtonID = 2

	urlWindowTitle  = "KolibriOS URL Demo"
	urlWindowX      = 214
	urlWindowY      = 134
	urlWindowWidth  = 880
	urlWindowHeight = 324
)

type App struct {
	summary    string
	parseLine  string
	queryLine  string
	pathLine   string
	valuesLine string
	noteLine   string
	ok         bool
	refreshBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(urlButtonRefresh, "Refresh", 28, 270)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.refreshProbe()
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
	case urlButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case urlButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(urlButtonExit, "Exit", 170, 270)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(urlWindowX, urlWindowY, urlWindowWidth, urlWindowHeight, urlWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary net/url package")
	kos.DrawText(28, 92, ui.Aqua, app.parseLine)
	kos.DrawText(28, 114, ui.Lime, app.queryLine)
	kos.DrawText(28, 136, ui.Yellow, app.pathLine)
	kos.DrawText(28, 158, ui.White, app.valuesLine)
	kos.DrawText(28, 180, ui.Black, app.noteLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	const rawURL = "https://127.0.0.1:8080/sys/default.skn?name=go+demo&ok=1#frag"
	const queryRaw = "hello world?x=1&y=2"
	const queryEscaped = "hello+world%3Fx%3D1%26y%3D2"
	const pathRaw = "/sys/default skin"
	const pathEscaped = "%2Fsys%2Fdefault%20skin"
	const encodedExpected = "file=%2Fsys%2Fdefault.skn&mode=demo&mode=mirror"

	parsed, err := url.Parse(rawURL)
	if err != nil {
		app.fail("parse failed", "Info: "+err.Error())
		return
	}
	mailto, err := url.Parse("mailto:go@example.com")
	if err != nil {
		app.fail("opaque parse failed", "Info: "+err.Error())
		return
	}

	queryValue := parsed.Query()
	encodedValues := make(url.Values)
	encodedValues.Set("mode", "demo")
	encodedValues.Add("mode", "mirror")
	encodedValues.Set("file", "/sys/default.skn")
	encoded := encodedValues.Encode()

	escapedQuery := url.QueryEscape(queryRaw)
	unescapedQuery, err := url.QueryUnescape(escapedQuery)
	if err != nil {
		app.fail("query unescape failed", "Info: "+err.Error())
		return
	}
	escapedPath := url.PathEscape(pathRaw)
	unescapedPath, err := url.PathUnescape(escapedPath)
	if err != nil {
		app.fail("path unescape failed", "Info: "+err.Error())
		return
	}

	app.parseLine = fmt.Sprintf("Parse: %s / scheme %s / host %s / path %s / frag %s", rawURL, parsed.Scheme, parsed.Host, parsed.Path, parsed.Fragment)
	app.queryLine = fmt.Sprintf("Query: %s -> %s -> %s / name %s / ok %s", queryRaw, escapedQuery, unescapedQuery, queryValue.Get("name"), queryValue.Get("ok"))
	app.pathLine = fmt.Sprintf("Path: %s -> %s -> %s / escaped-path %s", pathRaw, escapedPath, unescapedPath, parsed.EscapedPath())
	app.valuesLine = fmt.Sprintf("Values/String: %s / opaque %s / encoded %s", parsed.String(), mailto.Opaque, encoded)
	app.noteLine = "Info: current bootstrap url.Parse focuses on scheme/host/path/query/fragment/opaque and deterministic Values.Encode"

	if parsed.Scheme != "https" || parsed.Host != "127.0.0.1:8080" || parsed.Path != "/sys/default.skn" || parsed.RawQuery != "name=go+demo&ok=1" || parsed.Fragment != "frag" {
		app.fail("parsed URL mismatch", "Info: host/path/query fields do not match")
		return
	}
	if parsed.String() != rawURL || mailto.Opaque != "go@example.com" {
		app.fail("string or opaque mismatch", "Info: String()/opaque contract mismatch")
		return
	}
	if escapedQuery != queryEscaped || unescapedQuery != queryRaw {
		app.fail("query escape mismatch", "Info: QueryEscape/QueryUnescape round-trip mismatch")
		return
	}
	if escapedPath != pathEscaped || unescapedPath != pathRaw || parsed.EscapedPath() != "%2Fsys%2Fdefault.skn" {
		app.fail("path escape mismatch", "Info: PathEscape/PathUnescape mismatch")
		return
	}
	if queryValue.Get("name") != "go demo" || queryValue.Get("ok") != "1" || !queryValue.Has("ok") || encoded != encodedExpected {
		app.fail("query values mismatch", "Info: Query()/Values.Encode mismatch")
		return
	}

	app.ok = true
	app.summary = "url probe ok / ordinary import net/url resolved"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "url probe failed / " + detail
	app.noteLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
