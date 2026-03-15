package main

import (
	"io"
	"kos"
	"os"
	"path"
	"ui"
	"ui/elements"
)

const (
	osButtonExit    kos.ButtonID = 1
	osButtonRefresh kos.ButtonID = 2

	osWindowTitle  = "KolibriOS OS Demo"
	osWindowX      = 230
	osWindowY      = 138
	osWindowWidth  = 820
	osWindowHeight = 366

	osDemoDirName      = "go-os-demo"
	osDemoFileName     = "sample.txt"
	osDemoRenamedName  = "renamed.txt"
	osDemoPayloadBase  = "KolibriOS os demo"
	osDemoPayloadExtra = " / append"
	osPreferredRoot    = "/FD/1"
)

type bufferWriter struct {
	data []byte
}

func (writer *bufferWriter) Write(buffer []byte) (int, error) {
	writer.data = append(writer.data, buffer...)
	return len(buffer), nil
}

type App struct {
	summary    string
	cwdLine    string
	writeLine  string
	readLine   string
	seekLine   string
	renameLine string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(osButtonRefresh, "Refresh", 28, 308)
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
	case osButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case osButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(osButtonExit, "Exit", 170, 308)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(osWindowX, osWindowY, osWindowWidth, osWindowHeight, osWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary os package: import \"os\"")
	kos.DrawText(28, 92, ui.Aqua, app.cwdLine)
	kos.DrawText(28, 114, ui.Lime, app.writeLine)
	kos.DrawText(28, 136, ui.Yellow, app.readLine)
	kos.DrawText(28, 158, ui.Navy, app.seekLine)
	kos.DrawText(28, 180, ui.Maroon, app.renameLine)
	kos.DrawText(28, 202, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	cwd, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed")
		return
	}

	baseDir := osPreferredRoot
	if _, err := os.Stat(baseDir); err != nil {
		baseDir = cwd
	}

	demoDir := path.Join(baseDir, osDemoDirName)
	demoFile := path.Join(demoDir, osDemoFileName)
	renamedFile := path.Join(demoDir, osDemoRenamedName)
	payload := osDemoPayloadBase + osDemoPayloadExtra

	if err := removeIfExists(renamedFile); err != nil {
		app.fail("cleanup renamed file failed")
		return
	}
	if err := removeIfExists(demoFile); err != nil {
		app.fail("cleanup demo file failed")
		return
	}
	if err := removeIfExists(demoDir); err != nil {
		app.fail("cleanup demo dir failed")
		return
	}

	if err := os.Mkdir(demoDir, 0); err != nil {
		app.fail("mkdir failed")
		return
	}

	file, err := os.Create(demoFile)
	if err != nil {
		app.fail("create failed")
		return
	}

	wrote, err := io.WriteString(file, osDemoPayloadBase)
	if err == nil {
		var appendFile *os.File
		appendFile, err = os.OpenFile(demoFile, os.O_WRONLY|os.O_APPEND, 0)
		if err == nil {
			_, err = io.WriteString(appendFile, osDemoPayloadExtra)
			closeErr := appendFile.Close()
			if err == nil {
				err = closeErr
			}
		}
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		app.fail("write failed")
		return
	}

	data, err := os.ReadFile(demoFile)
	if err != nil {
		app.fail("readfile failed")
		return
	}

	reader, err := os.Open(demoFile)
	if err != nil {
		app.fail("open failed")
		return
	}
	readerInfo, err := reader.Stat()
	if err != nil {
		_ = reader.Close()
		app.fail("file stat failed")
		return
	}
	headAt := make([]byte, len(osDemoPayloadBase))
	headAtCount, headAtErr := reader.ReadAt(headAt, 0)
	if headAtErr != nil && headAtErr != io.EOF {
		_ = reader.Close()
		app.fail("readat failed")
		return
	}
	seekPos, seekErr := reader.Seek(-int64(len(osDemoPayloadExtra)), io.SeekEnd)
	if seekErr != nil {
		_ = reader.Close()
		app.fail("seek end failed")
		return
	}
	tail := make([]byte, len(osDemoPayloadExtra))
	tailRead, tailErr := reader.Read(tail)
	if tailErr != nil && tailErr != io.EOF {
		_ = reader.Close()
		app.fail("seek read failed")
		return
	}
	restartPos, restartErr := reader.Seek(0, io.SeekStart)
	if restartErr != nil {
		_ = reader.Close()
		app.fail("seek start failed")
		return
	}
	copyTarget := &bufferWriter{}
	copied, copyErr := io.Copy(copyTarget, reader)
	closeErr = reader.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		app.fail("copy failed")
		return
	}

	info, err := os.Stat(demoFile)
	if err != nil {
		app.ok = false
		app.summary = "os probe failed / stat unavailable"
		app.infoLine = "Info: " + demoFile + " / " + err.Error()
		return
	}
	rawInfo, ok := info.Sys().(kos.FileInfo)
	if !ok {
		app.fail("stat sys payload mismatch")
		return
	}
	modTime := info.ModTime()

	if err := os.Rename(demoFile, renamedFile); err != nil {
		app.fail("rename failed")
		return
	}

	renamedData, err := os.ReadFile(renamedFile)
	if err != nil {
		app.fail("renamed read failed")
		return
	}

	if err := os.Remove(renamedFile); err != nil {
		app.fail("remove file failed")
		return
	}
	if err := os.Remove(demoDir); err != nil {
		app.fail("remove dir failed")
		return
	}

	app.cwdLine = "Getwd: " + cwd + " / probe root " + baseDir
	app.writeLine = "Mkdir/Create/OpenFile: wrote " + formatInt(wrote) + " + " + formatInt(len(osDemoPayloadExtra)) + " bytes into " + demoFile
	app.readLine = "ReadFile/ReadAt: len " + formatInt(len(data)) + " / head " + string(headAt[:headAtCount]) + " / match " + formatBool(equalBytes(copyTarget.data, data))
	app.seekLine = "Seek/Open+Copy: tail " + string(tail[:tailRead]) + " / pos " + formatInt64(seekPos) + " -> " + formatInt64(restartPos) + " / copy " + formatInt64(copied)
	app.renameLine = "Rename/Remove: " + demoFile + " -> " + renamedFile + " / cleanup ok"

	if string(data) != payload || !equalBytes(copyTarget.data, []byte(payload)) {
		app.fail("payload mismatch")
		return
	}
	if copied != int64(len(payload)) {
		app.fail("copy length mismatch")
		return
	}
	if headAtCount != len(osDemoPayloadBase) || string(headAt[:headAtCount]) != osDemoPayloadBase {
		app.fail("readat mismatch")
		return
	}
	if seekPos != int64(len(osDemoPayloadBase)) || tailRead != len(osDemoPayloadExtra) || string(tail[:tailRead]) != osDemoPayloadExtra || restartPos != 0 {
		app.fail("seek mismatch")
		return
	}
	if readerInfo.Size() != int64(len(payload)) {
		app.fail("file stat size mismatch")
		return
	}
	if string(renamedData) != payload {
		app.fail("renamed payload mismatch")
		return
	}
	if info.Size() != int64(len(payload)) {
		app.fail("file size mismatch")
		return
	}
	if modTime.IsZero() || modTime.Year() < 2000 {
		app.fail("modtime unavailable")
		return
	}
	if os.Getpid() <= 0 {
		app.fail("getpid failed")
		return
	}
	os.Clearenv()
	if err := os.Setenv("GOOS_DEMO", "kolibri"); err != nil {
		app.fail("setenv failed")
		return
	}
	envValue, envOK := os.LookupEnv("GOOS_DEMO")
	envList := os.Environ()
	if err := os.Unsetenv("GOOS_DEMO"); err != nil {
		app.fail("unsetenv failed")
		return
	}
	if !envOK || envValue != "kolibri" || len(envList) != 1 || envList[0] != "GOOS_DEMO=kolibri" {
		app.fail("environment mismatch")
		return
	}

	argv0 := ""
	if len(os.Args) > 0 {
		argv0 = path.Base(os.Args[0])
		if argv0 == "" {
			argv0 = os.Args[0]
		}
	}

	app.ok = true
	app.summary = "os probe ok / ordinary import os package resolved"
	app.cwdLine = "Getwd/Getpid/Getppid/Args: " + cwd + " / pid " + formatInt(os.Getpid()) + " / ppid " + formatInt(os.Getppid()) + " / args " + formatInt(len(os.Args)) + " / argv0 " + argv0
	app.infoLine = "Info: size " + formatHex64(uint64(info.Size())) + " bytes / attrs " + formatHex32(uint32(rawInfo.Attributes)) + " / mod " + formatTimeStamp(modTime) + " / env " + envValue
}

func (app *App) fail(detail string) {
	app.ok = false
	app.summary = "os probe failed / " + detail
	app.infoLine = "Info: unavailable"
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func removeIfExists(name string) error {
	err := os.Remove(name)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	return err
}
