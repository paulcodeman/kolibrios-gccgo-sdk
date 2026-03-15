package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"syscall"
	"ui/elements"

	"kos"
	"ui"
)

const (
	bufioButtonExit    kos.ButtonID = 1
	bufioButtonRefresh kos.ButtonID = 2

	bufioWindowTitle  = "KolibriOS Bufio Demo"
	bufioWindowX      = 220
	bufioWindowY      = 132
	bufioWindowWidth  = 860
	bufioWindowHeight = 338

	bufioProbePath = "/sys/default.skn"
)

type App struct {
	summary     string
	readerLine  string
	writerLine  string
	scannerLine string
	bytesLine   string
	infoLine    string
	ok          bool
	refreshBtn  *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(bufioButtonRefresh, "Refresh", 28, 286)
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
	case bufioButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case bufioButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(bufioButtonExit, "Exit", 170, 286)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(bufioWindowX, bufioWindowY, bufioWindowWidth, bufioWindowHeight, bufioWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary bufio package")
	kos.DrawText(28, 92, ui.Aqua, app.readerLine)
	kos.DrawText(28, 114, ui.Lime, app.writerLine)
	kos.DrawText(28, 136, ui.Yellow, app.scannerLine)
	kos.DrawText(28, 158, ui.White, app.bytesLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, err := os.Stat(bufioProbePath)
	if err != nil {
		app.fail("file info unavailable", "Info: "+bufioProbePath+" / "+err.Error())
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

	readerPipe, writerPipe, err := os.Pipe()
	if err != nil {
		app.fail("pipe unavailable", "Info: "+err.Error())
		return
	}
	bufferedWriter := bufio.NewWriter(writerPipe)
	if _, err = bufferedWriter.WriteString("alpha beta\n"); err != nil {
		_ = readerPipe.Close()
		_ = writerPipe.Close()
		app.fail("WriteString failed", "Info: "+err.Error())
		return
	}
	if err = bufferedWriter.WriteByte('g'); err != nil {
		_ = readerPipe.Close()
		_ = writerPipe.Close()
		app.fail("WriteByte failed", "Info: "+err.Error())
		return
	}
	if _, err = bufferedWriter.WriteString("amma\n"); err != nil {
		_ = readerPipe.Close()
		_ = writerPipe.Close()
		app.fail("WriteString tail failed", "Info: "+err.Error())
		return
	}
	if err = bufferedWriter.Flush(); err != nil {
		_ = readerPipe.Close()
		_ = writerPipe.Close()
		app.fail("Flush failed", "Info: "+err.Error())
		return
	}
	_ = writerPipe.Close()

	bufferedReader := bufio.NewReader(readerPipe)
	firstByte, err := bufferedReader.ReadByte()
	if err != nil {
		_ = readerPipe.Close()
		app.fail("ReadByte failed", "Info: "+err.Error())
		return
	}
	if err = bufferedReader.UnreadByte(); err != nil {
		_ = readerPipe.Close()
		app.fail("UnreadByte failed", "Info: "+err.Error())
		return
	}
	firstLine, err := bufferedReader.ReadString('\n')
	if err != nil {
		_ = readerPipe.Close()
		app.fail("ReadString failed", "Info: "+err.Error())
		return
	}
	secondLine, err := bufferedReader.ReadBytes('\n')
	if err != nil {
		_ = readerPipe.Close()
		app.fail("ReadBytes failed", "Info: "+err.Error())
		return
	}
	_, eofErr := bufferedReader.ReadByte()
	_ = readerPipe.Close()

	brokenReader, brokenWriter, err := os.Pipe()
	if err != nil {
		app.fail("broken pipe unavailable", "Info: "+err.Error())
		return
	}
	_ = brokenReader.Close()
	_, brokenErr := brokenWriter.Write([]byte("x"))
	_ = brokenWriter.Close()

	linesReader, linesWriter, err := os.Pipe()
	if err != nil {
		app.fail("line pipe unavailable", "Info: "+err.Error())
		return
	}
	if _, err = linesWriter.Write([]byte("line one\nline two\n")); err != nil {
		_ = linesReader.Close()
		_ = linesWriter.Close()
		app.fail("line pipe write failed", "Info: "+err.Error())
		return
	}
	_ = linesWriter.Close()

	lineScanner := bufio.NewScanner(linesReader)
	lineA := ""
	lineB := ""
	if lineScanner.Scan() {
		lineA = lineScanner.Text()
	}
	if lineScanner.Scan() {
		lineB = lineScanner.Text()
	}
	lineScanErr := lineScanner.Err()
	_ = linesReader.Close()

	wordsReader, wordsWriter, err := os.Pipe()
	if err != nil {
		app.fail("word pipe unavailable", "Info: "+err.Error())
		return
	}
	if _, err = wordsWriter.Write([]byte("one two three\n")); err != nil {
		_ = wordsReader.Close()
		_ = wordsWriter.Close()
		app.fail("word pipe write failed", "Info: "+err.Error())
		return
	}
	_ = wordsWriter.Close()

	wordScanner := bufio.NewScanner(wordsReader)
	wordScanner.Split(bufio.ScanWords)
	wordA := ""
	wordB := ""
	wordC := ""
	if wordScanner.Scan() {
		wordA = wordScanner.Text()
	}
	if wordScanner.Scan() {
		wordB = wordScanner.Text()
	}
	if wordScanner.Scan() {
		wordC = wordScanner.Text()
	}
	wordScanErr := wordScanner.Err()
	_ = wordsReader.Close()

	bytesReader, bytesWriter, err := os.Pipe()
	if err != nil {
		app.fail("byte pipe unavailable", "Info: "+err.Error())
		return
	}
	if _, err = bytesWriter.Write([]byte("AZ")); err != nil {
		_ = bytesReader.Close()
		_ = bytesWriter.Close()
		app.fail("byte pipe write failed", "Info: "+err.Error())
		return
	}
	_ = bytesWriter.Close()

	byteScanner := bufio.NewScanner(bytesReader)
	byteScanner.Split(bufio.ScanBytes)
	byteA := ""
	byteB := ""
	if byteScanner.Scan() {
		byteA = byteScanner.Text()
	}
	if byteScanner.Scan() {
		byteB = byteScanner.Text()
	}
	byteScanErr := byteScanner.Err()
	_ = bytesReader.Close()

	app.readerLine = "Reader: byte " + string([]byte{firstByte}) + " / line " + trimTrailingNewline(firstLine) + " / bytes " + trimTrailingNewline(string(secondLine))
	app.writerLine = "Writer: flush true / eof " + formatBool(errors.Is(eofErr, io.EOF)) + " / epipe " + formatBool(errors.Is(brokenErr, syscall.EPIPE)) + " / cwd " + currentFolder
	app.scannerLine = "Scanner: lines " + lineA + " | " + lineB + " / words " + wordA + "," + wordB + "," + wordC
	app.bytesLine = "ScanBytes: " + byteA + "," + byteB + " / unread true"
	app.infoLine = "File: size " + formatHex64(uint64(info.Size())) + " bytes / attrs " + formatHex32(uint32(rawInfo.Attributes))

	if firstByte != 'a' {
		app.fail("ReadByte mismatch", "Info: expected a")
		return
	}
	if firstLine != "alpha beta\n" {
		app.fail("ReadString mismatch", "Info: expected alpha beta")
		return
	}
	if string(secondLine) != "gamma\n" {
		app.fail("ReadBytes mismatch", "Info: expected gamma")
		return
	}
	if !errors.Is(eofErr, io.EOF) {
		app.fail("EOF mismatch", "Info: expected EOF after writer close")
		return
	}
	if !errors.Is(brokenErr, syscall.EPIPE) {
		app.fail("EPIPE mismatch", "Info: expected broken pipe after reader close")
		return
	}
	if lineScanErr != nil || lineA != "line one" || lineB != "line two" {
		app.fail("ScanLines mismatch", "Info: expected line one / line two")
		return
	}
	if wordScanErr != nil || wordA != "one" || wordB != "two" || wordC != "three" {
		app.fail("ScanWords mismatch", "Info: expected one/two/three")
		return
	}
	if byteScanErr != nil || byteA != "A" || byteB != "Z" {
		app.fail("ScanBytes mismatch", "Info: expected A/Z")
		return
	}

	app.ok = true
	app.summary = "bufio probe ok / ordinary import bufio resolved"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "bufio probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
