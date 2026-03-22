package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"ui/elements"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"kos"
	"ui"
)

const (
	go9pDemoButtonExit    kos.ButtonID = 1
	go9pDemoButtonRefresh kos.ButtonID = 2

	go9pDemoWindowTitle  = "KolibriOS go9p Demo"
	go9pDemoWindowX      = 210
	go9pDemoWindowY      = 132
	go9pDemoWindowWidth  = 760
	go9pDemoWindowHeight = 260

	go9pDemoUser = "demo"
)

type App struct {
	summary     string
	line1       string
	line2       string
	line3       string
	line4       string
	ok          bool
	probeID     int
	activeProbe int
	updateCh    chan probeResult
	refreshBtn  *ui.Element
}

type probeResult struct {
	id      int
	summary string
	line2   string
	line3   string
	line4   string
	ok      bool
}

func NewApp() App {
	refresh := elements.ButtonAt(go9pDemoButtonRefresh, "Refresh", 28, 204)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.startProbe()
	return app
}

func (app *App) Run() {
	for {
		app.pollProbe()
		switch kos.WaitEventFor(5) {
		case kos.EventNone:
			app.pollProbe()
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
	case go9pDemoButtonRefresh:
		app.startProbe()
		app.Redraw()
	case go9pDemoButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(go9pDemoButtonExit, "Exit", 170, 204)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(go9pDemoWindowX, go9pDemoWindowY, go9pDemoWindowWidth, go9pDemoWindowHeight, go9pDemoWindowTitle)
	kos.DrawText(28, 42, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample serves a 9p tree through go9p and mounts it through plan9/client")
	kos.DrawText(28, 94, ui.Aqua, app.line1)
	kos.DrawText(28, 116, ui.Lime, app.line2)
	kos.DrawText(28, 138, ui.Yellow, app.line3)
	kos.DrawText(28, 160, ui.Black, app.line4)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) startProbe() {
	app.probeID++
	service := fmt.Sprintf("go9pdemo-%d", app.probeID)
	app.activeProbe = app.probeID
	app.updateCh = make(chan probeResult, 8)
	app.ok = false
	app.summary = "go9p/plan9 client probe in progress / waiting for PostSrv and mount path"
	app.line1 = "Namespace: " + client.Namespace()
	app.line2 = "Service: " + service
	app.line3 = "Result: pending"
	app.line4 = "Info: starting background probe"
	updateCh := app.updateCh
	probeID := app.probeID
	go func() {
		runProbe(updateCh, probeID, service)
	}()
}

func runProbe(updateCh chan<- probeResult, probeID int, service string) {
	expected := "hello from go9p on kolibrios\n"
	reportProgress(updateCh, probeID, service, "building static fs")

	staticFS, root := fs.NewFS(go9pDemoUser, go9pDemoUser, 0555)
	if err := root.AddChild(fs.NewStaticFile(staticFS.NewStat("hello", go9pDemoUser, go9pDemoUser, 0444), []byte(expected))); err != nil {
		updateCh <- failedProbe(probeID, service, "root add failed", err.Error())
		return
	}

	reportProgress(updateCh, probeID, service, "starting PostSrv")
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- go9p.PostSrv(service, staticFS.Server())
	}()

	reportProgress(updateCh, probeID, service, "waiting for service registry")
	if err := waitForService(service, serverErr, 2*time.Second); err != nil {
		updateCh <- failedProbe(probeID, service, "service post failed", err.Error())
		return
	}

	reportProgress(updateCh, probeID, service, "dialing service")
	conn, err := client.DialService(service)
	if err != nil {
		updateCh <- failedProbe(probeID, service, "dial service failed", err.Error())
		return
	}
	defer conn.Close()

	reportProgress(updateCh, probeID, service, "attaching root fid")
	fsys, err := conn.Attach(nil, go9pDemoUser, "")
	if err != nil {
		updateCh <- failedProbe(probeID, service, "attach failed", err.Error())
		return
	}
	defer fsys.Close()

	reportProgress(updateCh, probeID, service, "opening hello")
	fid, err := fsys.Open("hello", plan9.OREAD)
	if err != nil {
		updateCh <- failedProbe(probeID, service, "open failed", err.Error())
		return
	}
	reportProgress(updateCh, probeID, service, "reading hello")
	data, readErr := readAll(fid)
	closeErr := fid.Close()
	if readErr != nil {
		updateCh <- failedProbe(probeID, service, "read failed", readErr.Error())
		return
	}
	if closeErr != nil {
		updateCh <- failedProbe(probeID, service, "clunk failed", closeErr.Error())
		return
	}

	if string(data) != expected {
		updateCh <- failedProbe(probeID, service, "content mismatch", fmt.Sprintf("got %q", string(data)))
		return
	}

	registryPath := filepath.Join(client.Namespace(), service)
	result := probeResult{
		id:      probeID,
		ok:      true,
		summary: "go9p/plan9 client probe ok / PostSrv, DialService, Attach and Open all succeeded",
		line2:   "Service: " + service + " / registry " + formatBool(!pathExists(registryPath)),
		line3:   "Result: " + strings.TrimSpace(string(data)),
		line4:   "Info: namespace transport and 9p read path both work",
	}

	select {
	case err := <-serverErr:
		if err != nil {
			updateCh <- failedProbe(probeID, service, "server returned error", err.Error())
			return
		}
	case <-time.After(500 * time.Millisecond):
		result.line4 = "Info: server still closing after client disconnect"
	}
	updateCh <- result
}

func (app *App) fail(summary, info string) {
	app.ok = false
	app.summary = "go9p/plan9 client probe failed / " + summary
	app.line3 = "Result: failed"
	app.line4 = "Info: " + info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}
	return ui.Red
}

func (app *App) pollProbe() {
	if app.updateCh == nil {
		return
	}
	for {
		select {
		case result := <-app.updateCh:
			if result.id != app.activeProbe {
				continue
			}
			app.ok = result.ok
			app.summary = result.summary
			app.line2 = result.line2
			app.line3 = result.line3
			app.line4 = result.line4
			app.Redraw()
		default:
			return
		}
	}
}

func reportProgress(updateCh chan<- probeResult, probeID int, service string, info string) {
	select {
	case updateCh <- probeResult{
		id:      probeID,
		ok:      false,
		summary: "go9p/plan9 client probe in progress / waiting for PostSrv and mount path",
		line2:   "Service: " + service,
		line3:   "Result: pending",
		line4:   "Info: " + info,
	}:
	default:
	}
}

func waitForService(service string, serverErr <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	path := filepath.Join(client.Namespace(), service)
	for time.Now().Before(deadline) {
		if pathExists(path) {
			return nil
		}
		select {
		case err := <-serverErr:
			if err == nil {
				return fmt.Errorf("service exited before registry publish")
			}
			return err
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("service registry missing: %s", path)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readAll(fid io.Reader) ([]byte, error) {
	buf := make([]byte, 128)
	var out []byte
	for {
		n, err := fid.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, err
		}
	}
}

func formatBool(v bool) string {
	if v {
		return "ok"
	}
	return "bad"
}

func failedProbe(probeID int, service string, summary string, info string) probeResult {
	return probeResult{
		id:      probeID,
		ok:      false,
		summary: "go9p/plan9 client probe failed / " + summary,
		line2:   "Service: " + service,
		line3:   "Result: failed",
		line4:   "Info: " + info,
	}
}

func main() {
	app := NewApp()
	app.Run()
}
