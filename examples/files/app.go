package main

import (
	"errors"
	"io"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	filesButtonExit    kos.ButtonID = 1
	filesButtonRefresh kos.ButtonID = 2

	filesProbePath = "/sys/default.skn"

	filesWindowX      = 320
	filesWindowY      = 180
	filesWindowWidth  = 620
	filesWindowHeight = 252
	filesWindowTitle  = "KolibriOS Files Demo"
	filesPreviewBytes = 16
)

var (
	errPathInfo = &pathSentinel{text: "path info failed"}
	errPathRead = &pathSentinel{text: "path read failed"}
)

type pathSentinel struct {
	text string
}

func (err *pathSentinel) Error() string {
	return err.text
}

type probeError struct {
	op     string
	path   string
	status kos.FileSystemStatus
	detail string
	cause  error
}

func (err probeError) Error() string {
	if err.detail != "" {
		return err.op + " " + err.path + " / " + err.detail
	}

	return err.op + " " + err.path + " / " + formatFileSystemStatus(err.status)
}

func (err probeError) Unwrap() error {
	return err.cause
}

func (err probeError) As(target interface{}) bool {
	switch typed := target.(type) {
	case *probeError:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}

	return false
}

type App struct {
	path       string
	summary    string
	infoLine   string
	readLine   string
	classLine  string
	errorLine  string
	errorsOK   bool
	lastError  error
	refreshBtn *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(filesButtonRefresh, "Refresh", 28, 198)
	refresh.SetWidth(116)

	app := App{
		path:       filesProbePath,
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
	case filesButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case filesButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(filesButtonExit, "Exit", 170, 198)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(filesWindowX, filesWindowY, filesWindowWidth, filesWindowHeight, filesWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports errors/io/os with ordinary Go import paths")
	kos.DrawText(28, 88, ui.Black, "Path: "+app.path)
	kos.DrawText(28, 112, ui.Aqua, app.infoLine)
	kos.DrawText(28, 134, ui.Lime, app.readLine)
	kos.DrawText(28, 156, ui.Yellow, app.classLine)
	kos.DrawText(28, 178, ui.Black, app.errorLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	app.lastError = nil
	app.errorsOK = checkErrorsCompatibility()

	info, err := os.Stat(app.path)
	if err != nil {
		app.fail(&probeError{
			op:     "stat",
			path:   app.path,
			detail: err.Error(),
			cause:  errPathInfo,
		})
		return
	}
	rawInfo, ok := info.Sys().(kos.FileInfo)
	if !ok {
		app.fail(&probeError{
			op:     "stat",
			path:   app.path,
			detail: "unexpected sys info payload",
			cause:  errPathInfo,
		})
		return
	}

	app.infoLine = "Info: size " + formatHex64(uint64(info.Size())) + " bytes / attrs " + formatHex32(uint32(rawInfo.Attributes))

	previewSize := filesPreviewBytes
	if info.Size() > 0 && info.Size() < int64(previewSize) {
		previewSize = int(info.Size())
	}
	if previewSize == 0 {
		previewSize = filesPreviewBytes
	}

	file, err := os.Open(app.path)
	if err != nil {
		app.fail(&probeError{
			op:     "open",
			path:   app.path,
			detail: err.Error(),
			cause:  errPathRead,
		})
		return
	}
	buffer := make([]byte, previewSize)
	read, err := file.Read(buffer)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		app.fail(&probeError{
			op:     "read",
			path:   app.path,
			detail: err.Error(),
			cause:  errPathRead,
		})
		return
	}

	app.readLine = "Read: " + formatUint32(uint32(read)) + " bytes / head " + formatBytePreview(buffer[:read])
	if app.errorsOK {
		app.classLine = "errors: Is / As / Join / Unwrap ok"
		app.errorLine = "Error: none"
		app.summary = "file probe ok / bootstrap errors package resolved"
		return
	}

	app.errorLine = "Error: none"
	app.classLine = "errors: bootstrap self-check failed"
	app.summary = "file probe ok / bootstrap errors self-check failed"
}

func (app *App) fail(err error) {
	app.lastError = err
	app.readLine = "Read: unavailable"
	app.errorLine = "Error: " + err.Error()

	if errors.Is(err, errPathInfo) {
		app.infoLine = "Info: unavailable"
		app.summary = "file probe failed / errors.Is(errPathInfo)"
		app.classLine = "errors.Is(errPathInfo) = true"
		return
	}

	if errors.Is(err, errPathRead) {
		app.summary = "file probe failed / errors.Is(errPathRead)"
		app.classLine = "errors.Is(errPathRead) = true"
		return
	}

	app.summary = "file probe failed / unclassified error"
	app.classLine = "errors.Is: no sentinel match"
}

func (app *App) summaryColor() kos.Color {
	if app.lastError == nil && app.errorsOK {
		return ui.Lime
	}

	return ui.Red
}

func checkErrorsCompatibility() bool {
	wrapped := &probeError{
		op:     "diag",
		path:   filesProbePath,
		status: kos.FileSystemNotFound,
		cause:  errPathInfo,
	}
	joined := errors.Join(errPathInfo, errPathRead)
	var matched probeError
	return errors.Is(errPathInfo, errPathInfo) &&
		errors.Unwrap(wrapped) == errPathInfo &&
		errors.Is(wrapped, errPathInfo) &&
		errors.As(wrapped, &matched) &&
		matched.op == "diag" &&
		matched.path == filesProbePath &&
		errors.Is(joined, errPathInfo) &&
		errors.Is(joined, errPathRead)
}
