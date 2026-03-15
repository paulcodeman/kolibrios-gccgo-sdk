package main

import (
	"errors"
	"os"
	"strconv"
	"ui/elements"

	"kos"
	"ui"
)

const (
	strconvButtonExit    kos.ButtonID = 1
	strconvButtonRefresh kos.ButtonID = 2

	strconvWindowTitle  = "KolibriOS Strconv Demo"
	strconvWindowX      = 226
	strconvWindowY      = 138
	strconvWindowWidth  = 844
	strconvWindowHeight = 324

	strconvProbePath = "/sys/default.skn"
)

type App struct {
	summary    string
	formatLine string
	parseLine  string
	appendLine string
	errorLine  string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(strconvButtonRefresh, "Refresh", 28, 272)
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
	case strconvButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case strconvButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(strconvButtonExit, "Exit", 170, 272)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(strconvWindowX, strconvWindowY, strconvWindowWidth, strconvWindowHeight, strconvWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary strconv package")
	kos.DrawText(28, 92, ui.Aqua, app.formatLine)
	kos.DrawText(28, 114, ui.Lime, app.parseLine)
	kos.DrawText(28, 136, ui.Yellow, app.appendLine)
	kos.DrawText(28, 158, ui.White, app.errorLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, err := os.Stat(strconvProbePath)
	if err != nil {
		app.fail("file info unavailable", "Info: "+strconvProbePath+" / "+err.Error())
		return
	}
	rawInfo, ok := info.Sys().(kos.FileInfo)
	if !ok {
		app.fail("stat sys payload mismatch", "Info: unexpected os.FileInfo.Sys() payload")
		return
	}
	currentFolder, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed", "Info: "+err.Error())
		return
	}

	formatBool := strconv.FormatBool(true)
	formatInt := strconv.Itoa(-42)
	formatHex := strconv.FormatInt(-42, 16)
	formatUint := strconv.FormatUint(uint64(info.Size()), 16)

	parseBool, parseBoolErr := strconv.ParseBool("TRUE")
	parseInt, parseIntErr := strconv.Atoi("214")
	parseHex, parseHexErr := strconv.ParseInt("-0x2a", 0, 32)
	parseBin, parseBinErr := strconv.ParseUint("0b1010", 0, 32)

	appendInt := string(strconv.AppendInt([]byte("n="), -42, 10))
	appendUint := string(strconv.AppendUint([]byte("h="), uint64(info.Size()), 16))
	appendBool := string(strconv.AppendBool([]byte("ok="), true))

	_, rangeErr := strconv.ParseUint("999", 10, 8)
	_, syntaxErr := strconv.ParseBool("maybe")

	app.formatLine = "Format: bool " + formatBool + " / itoa " + formatInt + " / hex " + formatHex + " / size " + formatUint
	app.parseLine = "Parse: bool " + strconv.FormatBool(parseBool) + " / atoi " + strconv.Itoa(parseInt) + " / hex " + strconv.FormatInt(parseHex, 10) + " / bin " + strconv.FormatUint(parseBin, 10)
	app.appendLine = "Append: " + appendInt + " / " + appendUint + " / " + appendBool
	app.errorLine = "Errors: range " + strconv.FormatBool(errors.Is(rangeErr, strconv.ErrRange)) + " / syntax " + strconv.FormatBool(errors.Is(syntaxErr, strconv.ErrSyntax))
	app.infoLine = "Info: cwd " + currentFolder + " / size " + strconv.FormatInt(info.Size(), 10) + " / attrs 0x" + strconv.FormatUint(uint64(rawInfo.Attributes), 16)

	if formatBool != "true" || formatInt != "-42" || formatHex != "-2a" {
		app.fail("format mismatch", "Info: unexpected bool/int/hex text")
		return
	}
	if formatUint == "" {
		app.fail("uint format mismatch", "Info: empty size text")
		return
	}
	if parseBoolErr != nil || !parseBool {
		app.fail("ParseBool mismatch", "Info: expected TRUE -> true")
		return
	}
	if parseIntErr != nil || parseInt != 214 {
		app.fail("Atoi mismatch", "Info: expected 214")
		return
	}
	if parseHexErr != nil || parseHex != -42 {
		app.fail("ParseInt mismatch", "Info: expected -0x2a -> -42")
		return
	}
	if parseBinErr != nil || parseBin != 10 {
		app.fail("ParseUint mismatch", "Info: expected 0b1010 -> 10")
		return
	}
	if appendInt != "n=-42" || appendBool != "ok=true" {
		app.fail("append mismatch", "Info: unexpected append text")
		return
	}
	if appendUint != "h="+formatUint {
		app.fail("append uint mismatch", "Info: expected h=<size-hex>")
		return
	}
	if !errors.Is(rangeErr, strconv.ErrRange) || !errors.Is(syntaxErr, strconv.ErrSyntax) {
		app.fail("error mismatch", "Info: expected ErrRange and ErrSyntax")
		return
	}

	app.ok = true
	app.summary = "strconv probe ok / ordinary import strconv resolved"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "strconv probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
