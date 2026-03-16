package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
	"ui/elements"

	"kos"
	"ui"
)

const (
	logButtonExit    kos.ButtonID = 1
	logButtonRefresh kos.ButtonID = 2

	logWindowTitle  = "KolibriOS Log Demo"
	logWindowX      = 220
	logWindowY      = 136
	logWindowWidth  = 920
	logWindowHeight = 344

	logProbePath = "/sys/default.skn"
)

type App struct {
	summary     string
	defaultLine string
	customLine  string
	outputLine  string
	infoLine    string
	ok          bool
	refreshBtn  *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(logButtonRefresh, "Refresh", 28, 288)
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
	case logButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case logButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(logButtonExit, "Exit", 170, 288)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(logWindowX, logWindowY, logWindowWidth, logWindowHeight, logWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary log package")
	kos.DrawText(28, 92, ui.Aqua, app.defaultLine)
	kos.DrawText(28, 114, ui.Lime, app.customLine)
	kos.DrawText(28, 136, ui.Yellow, app.outputLine)
	kos.DrawText(28, 158, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, err := os.Stat(logProbePath)
	if err != nil {
		app.fail("stat failed", "Info: "+err.Error())
		return
	}

	currentFolder, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed", "Info: "+err.Error())
		return
	}

	defaultWriter := log.Writer()
	defaultFlags := log.Flags()
	defaultPrefix := log.Prefix()
	originalLocal := time.Local

	var defaultBuffer bytes.Buffer
	log.SetOutput(&defaultBuffer)
	log.SetFlags(0)
	log.SetPrefix("demo: ")
	log.Print("hello")
	log.Println("world", int(info.Size()))
	defaultText := defaultBuffer.String()

	var customBuffer bytes.Buffer
	logger := log.New(&customBuffer, "hdr: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix)
	logger.Printf("size=%d", info.Size())
	customText := customBuffer.String()

	fixedLocal := time.FixedZone("UTC+03", 3*60*60)
	time.Local = fixedLocal
	var localBuffer bytes.Buffer
	var utcBuffer bytes.Buffer
	localLogger := log.New(&localBuffer, "tz: ", log.Ltime|log.Lmsgprefix)
	utcLogger := log.New(&utcBuffer, "tz: ", log.Ltime|log.Lmsgprefix|log.LUTC)
	localLogger.Print("stamp")
	utcLogger.Print("stamp")
	localText := localBuffer.String()
	utcText := utcBuffer.String()

	logger.SetPrefix("line: ")
	logger.SetFlags(0)
	if err = logger.Output(0, "manual"); err != nil {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("Output failed", "Info: "+err.Error())
		return
	}
	outputText := customBuffer.String()

	if defaultText != "demo: hello\ndemo: world "+fmt.Sprintf("%d", info.Size())+"\n" {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("default logger mismatch", "Info: unexpected Print/Println text")
		return
	}
	if !strings.Contains(customText, "hdr: size=") || !strings.Contains(customText, "\n") {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("custom logger mismatch", "Info: expected dated header plus hdr prefix")
		return
	}
	if logger.Prefix() != "line: " || logger.Flags() != 0 {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("setter mismatch", "Info: Prefix/Flags did not update")
		return
	}
	if !strings.HasSuffix(outputText, "line: manual\n") {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("Output contract mismatch", "Info: expected newline-appended Output text")
		return
	}
	if !strings.HasSuffix(localText, "tz: stamp\n") || !strings.HasSuffix(utcText, "tz: stamp\n") || localText == utcText {
		restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
		app.fail("LUTC mismatch", "Info: expected UTC header to differ from fixed local header")
		return
	}

	restoreDefaultLogger(defaultWriter, defaultFlags, defaultPrefix, originalLocal)
	app.ok = true
	app.summary = "log probe ok / ordinary import log resolved"
	app.defaultLine = "Default: " + trimLineEnd(defaultText)
	app.customLine = "Custom: " + trimLineEnd(customText)
	app.outputLine = "Output: " + trimLineEnd(outputText) + " / LUTC " + trimLineEnd(utcText)
	app.infoLine = "Info: cwd " + currentFolder + " / size " + fmt.Sprintf("%d", info.Size()) + " / local " + trimLineEnd(localText)
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "log probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func trimLineEnd(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value[:len(value)-1]
	}

	return value
}

func restoreDefaultLogger(writer io.Writer, flags int, prefix string, local *time.Location) {
	log.SetOutput(writer)
	log.SetFlags(flags)
	log.SetPrefix(prefix)
	time.Local = local
}
