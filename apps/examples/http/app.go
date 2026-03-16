package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"ui/elements"

	"kos"
	"ui"
)

const (
	httpButtonExit    kos.ButtonID = 1
	httpButtonRefresh kos.ButtonID = 2

	httpWindowTitle  = "KolibriOS HTTP Demo"
	httpWindowX      = 224
	httpWindowY      = 142
	httpWindowWidth  = 860
	httpWindowHeight = 320
)

const httpProbeURL = "http://127.0.0.1:8080/sys/default.skn?name=go+demo"
const httpProbeBody = "hello=world"

type App struct {
	summary       string
	requestLine   string
	headerLine    string
	transportLine string
	noteLine      string
	ok            bool
	refreshBtn    *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(httpButtonRefresh, "Refresh", 28, 268)
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
	case httpButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case httpButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(httpButtonExit, "Exit", 170, 268)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(httpWindowX, httpWindowY, httpWindowWidth, httpWindowHeight, httpWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample uses ordinary import \"net/http\" and only keeps kos.HTTP for transport readiness reporting")
	kos.DrawText(28, 92, ui.Aqua, app.requestLine)
	kos.DrawText(28, 114, ui.Lime, app.headerLine)
	kos.DrawText(28, 136, ui.Yellow, app.transportLine)
	kos.DrawText(28, 158, ui.Black, app.noteLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	request, err := http.NewRequest(http.MethodPost, httpProbeURL, strings.NewReader(httpProbeBody))
	if err != nil {
		app.fail("new request failed", "Info: "+err.Error())
		return
	}

	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("X-Demo", "one")
	request.Header.Add("X-Demo", "two")
	cloned := request.Header.Clone()

	if request.Method != http.MethodPost ||
		request.URL == nil ||
		request.URL.Host != "127.0.0.1:8080" ||
		request.URL.Path != "/sys/default.skn" ||
		request.URL.RawQuery != "name=go+demo" ||
		request.ContentLength != int64(len(httpProbeBody)) ||
		request.Body == nil {
		app.fail("request mismatch", "Info: parsed request fields differ from expected net/http contract")
		return
	}

	values := request.Header.Values("x-demo")
	if request.Header.Get("X-Demo") != "one" || len(values) != 2 || values[1] != "two" {
		app.fail("header values mismatch", "Info: Header Add/Get/Values did not preserve insertion order")
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if cloned.Get("Content-Type") != "text/plain" || len(cloned.Values("x-demo")) != 2 {
		app.fail("header clone mismatch", "Info: Header Clone should deep-copy header values")
		return
	}

	request.Header.Del("X-Demo")
	if request.Header.Get("x-demo") != "" || len(request.Header.Values("x-demo")) != 0 {
		app.fail("header delete mismatch", "Info: Header Del should remove keys case-insensitively")
		return
	}

	if http.StatusText(http.StatusCreated) != "Created" ||
		http.StatusText(http.StatusNoContent) != "No Content" ||
		http.StatusText(http.StatusMethodNotAllowed) != "Method Not Allowed" ||
		http.StatusText(http.StatusNotFound) != "Not Found" {
		app.fail("status text mismatch", "Info: StatusText should resolve common bootstrap HTTP codes")
		return
	}

	ftpErr := ""
	if _, err = http.Get("ftp://127.0.0.1/"); err == nil {
		app.fail("unsupported scheme mismatch", "Info: ftp scheme should fail before transport start")
		return
	} else {
		ftpErr = err.Error()
	}
	formErr := ""
	formData := make(url.Values)
	formData.Set("mode", "demo")
	if _, err = http.PostForm("ftp://127.0.0.1/", formData); err == nil {
		app.fail("postform scheme mismatch", "Info: PostForm should fail before transport start on unsupported schemes")
		return
	} else {
		formErr = err.Error()
	}
	putRequest, err := http.NewRequest(http.MethodPut, httpProbeURL, strings.NewReader("replace=1"))
	if err != nil {
		app.fail("put request failed", "Info: "+err.Error())
		return
	}
	putErr := ""
	if _, err = http.DefaultClient.Do(putRequest); err == nil {
		app.fail("unsupported method mismatch", "Info: PUT should fail early on the bootstrap transport")
		return
	} else {
		putErr = err.Error()
	}

	transport, ok := kos.LoadHTTP()
	if !ok {
		app.fail("http.obj unavailable", "Info: failed to load "+kos.HTTPDLLPath)
		return
	}

	app.ok = true
	app.summary = "http probe ok / net/http request contract resolved"
	app.requestLine = fmt.Sprintf("Request: %s %s / len %d", request.Method, request.URL.String(), request.ContentLength)
	app.headerLine = fmt.Sprintf("Header: clone=%s / method=%s / Status=%s", cloned.Get("Content-Type"), http.MethodPut, http.StatusText(http.StatusMethodNotAllowed))
	app.transportLine = fmt.Sprintf("Transport: %s / ftp=%s / postform=%s / put=%s", kos.HTTPDLLPath, httpShortDetail(ftpErr), httpShortDetail(formErr), httpShortDetail(putErr))
	app.noteLine = fmt.Sprintf("Info: table 0x%x / ver 0x%x / transfer %s", uint32(transport.ExportTable()), transport.Version(), httpReadyText(transport.Ready()))
	if transport.Ready() {
		return
	}
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "http probe failed / " + detail
	app.noteLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func httpReadyText(ready bool) string {
	if ready {
		return "ready"
	}

	return "not-ready"
}

func httpShortDetail(value string) string {
	cut := strings.Index(value, ": ")
	if cut >= 0 && cut+2 < len(value) {
		return value[cut+2:]
	}

	return value
}
