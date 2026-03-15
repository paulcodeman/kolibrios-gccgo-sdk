package main

import (
	"errors"
	"fmt"
	"strings"
	"ui/elements"

	"kos"
	"ui"
)

const (
	fmtButtonExit    kos.ButtonID = 1
	fmtButtonRefresh kos.ButtonID = 2

	fmtWindowTitle  = "KolibriOS Fmt Demo"
	fmtWindowX      = 228
	fmtWindowY      = 140
	fmtWindowWidth  = 852
	fmtWindowHeight = 356

	fmtProbePath    = "/sys/default.skn"
	fmtPreviewBytes = 12
)

type probeLabel struct {
	text string
}

func (label probeLabel) String() string {
	return label.text
}

type bufferWriter struct {
	data []byte
}

func (writer *bufferWriter) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

type App struct {
	summary      string
	sprintfLine  string
	sprintlnLine string
	fprintfLine  string
	printLine    string
	errorLine    string
	scanLine     string
	infoLine     string
	ok           bool
	refreshBtn   *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(fmtButtonRefresh, "Refresh", 28, 304)
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
	case fmtButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case fmtButtonExit:
		kos.Exit()
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(fmtButtonExit, "Exit", 170, 304)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(fmtWindowX, fmtWindowY, fmtWindowWidth, fmtWindowHeight, fmtWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary fmt package: import \"fmt\"")
	kos.DrawText(28, 92, ui.Aqua, app.sprintfLine)
	kos.DrawText(28, 114, ui.Lime, app.sprintlnLine)
	kos.DrawText(28, 136, ui.Navy, app.fprintfLine)
	kos.DrawText(28, 158, ui.Maroon, app.printLine)
	kos.DrawText(28, 180, ui.Yellow, app.errorLine)
	kos.DrawText(28, 202, ui.Aqua, app.scanLine)
	kos.DrawText(28, 224, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, status := kos.GetFileInfo(fmtProbePath)
	if status != kos.FileSystemOK {
		app.fail("file info unavailable", "Info: "+fmtProbePath+" / status "+formatFileSystemStatus(status))
		return
	}
	rawInfo := info

	data, status := kos.ReadAllFile(fmtProbePath)
	if status != kos.FileSystemOK && status != kos.FileSystemEOF {
		app.fail("file read unavailable", "Info: "+fmtProbePath+" / status "+formatFileSystemStatus(status))
		return
	}
	if len(data) > fmtPreviewBytes {
		data = data[:fmtPreviewBytes]
	}

	currentFolder := kos.CurrentFolder()
	if currentFolder == "" {
		app.fail("getwd failed", "Info: current folder unavailable")
		return
	}
	label := probeLabel{text: "fmt"}

	sprintfText := fmt.Sprintf("%v/%s/%d/%x/%t/%%", label, "ok", 42, uint32(0x2A), true)
	printlnText := fmt.Sprintln(label, "line", true, 7)
	writer := &bufferWriter{}
	written, writeErr := fmt.Fprintf(writer, "cwd=%s / head=%x / size=%d", currentFolder, data, len(data))
	printWriter := &bufferWriter{}
	printWritten, printErr := fmt.Fprint(printWriter, label, " print ", 7, "\n")
	printfWritten, printfErr := fmt.Fprintf(printWriter, "cwd=%s / size=%d", currentFolder, len(data))
	printlnWritten, printlnErr := fmt.Fprintln(printWriter, " / tail", true)
	formatErr := fmt.Errorf("%v error %d", label, 7)
	wrapCause := errors.New("fmt wrapped")
	wrappedErr := fmt.Errorf("wrap: %w", wrapCause)
	leftCause := errors.New("fmt left")
	rightCause := errors.New("fmt right")
	multiWrappedErr := fmt.Errorf("multi: %w + %w", leftCause, rightCause)
	invalidWrapText := fmt.Sprintf("%w", "x")
	quotedText := fmt.Sprintf("%q/%q/%q/%q/%q/%q/%q/%q", label, "go\tfmt", []byte{0x4B, '\n'}, '\n', string([]byte{0xFF, 'A'}), "é", "\u00a0", "\u00ad")
	quotedASCIIText := fmt.Sprintf("%+q/%+q/%+q", "☺", []byte("☺"), '☺')
	rawQuotedText := fmt.Sprintf("%#q/%#q/%#+q", "abc", []byte("xy"), "a\tb")
	quotedErrorText := fmt.Sprintf("%q", formatErr)
	sliceText := fmt.Sprintf("%v/%v/%v/%v", []int{1, 2, 3}, []string{"go", "fmt"}, []byte{0x4B, '\n'}, []interface{}{"fmt", 7, true})
	widthText := fmt.Sprintf("%06X/%6d", uint32(0x2A), 42)
	var scanWord string
	var scanValue int
	var scanOK bool

	scanned, scanErr := fmt.Fscanln(strings.NewReader("scan 42 true\n"), &scanWord, &scanValue, &scanOK)
	if scanErr != nil {
		app.fail("Fscanln failed", "Info: "+scanErr.Error())
		return
	}

	var stdinWord string
	var stdinValue int
	var stdinOK bool
	stdinScanned, stdinErr := fmt.Fscanln(strings.NewReader("stdin 7 false\n"), &stdinWord, &stdinValue, &stdinOK)
	if stdinErr != nil {
		app.fail("Scanln failed", "Info: "+stdinErr.Error())
		return
	}

	expectedPrint := "fmt print 7\ncwd=" + currentFolder + " / size=" + formatInt(len(data)) + " / tail true\n"
	stdoutData := printWriter.data
	stdoutRead := len(stdoutData)

	app.sprintfLine = "Sprintf: " + sprintfText + " / slices " + sliceText + " / width " + widthText
	app.sprintlnLine = "Sprintln: " + trimTrailingNewline(printlnText) + " / newline " + fmt.Sprintf("%t", hasTrailingNewline(printlnText))
	app.fprintfLine = "Fprintf: wrote " + formatInt(written) + " / " + string(writer.data)
	app.printLine = "Print*: wrote " + formatInt(printWritten+printfWritten+printlnWritten) + " / stdout match " + fmt.Sprintf("%t", stdoutRead == len(expectedPrint) && string(stdoutData[:stdoutRead]) == expectedPrint)
	app.errorLine = "Errorf: " + formatErr.Error() + " / wrap " + fmt.Sprintf("%t", errors.Unwrap(wrappedErr) == wrapCause) + " / quote " + quotedText
	app.scanLine = "Scan*: Fscanln " + scanWord + "/" + formatInt(scanValue) + "/" + fmt.Sprintf("%t", scanOK) + " / Scanln " + stdinWord + "/" + formatInt(stdinValue) + "/" + fmt.Sprintf("%t", stdinOK)
	app.infoLine = fmt.Sprintf("File: %s / size %d / attrs 0x%x / head %x", fmtProbePath, rawInfo.Size, uint32(rawInfo.Attributes), data)

	expectedSprintf := "fmt/ok/42/2a/true/%"
	if sprintfText != expectedSprintf {
		app.fail("Sprintf mismatch", "Info: expected "+expectedSprintf)
		return
	}

	expectedSprintln := "fmt line true 7\n"
	if printlnText != expectedSprintln {
		app.fail("Sprintln mismatch", "Info: expected "+trimTrailingNewline(expectedSprintln))
		return
	}

	expectedFprintf := "cwd=" + currentFolder + " / head=" + formatHexBytes(data) + " / size=" + formatInt(len(data))
	if writeErr != nil {
		app.fail("Fprintf returned error", "Info: "+writeErr.Error())
		return
	}
	if written != len(expectedFprintf) || string(writer.data) != expectedFprintf {
		app.fail("Fprintf mismatch", "Info: expected "+expectedFprintf)
		return
	}

	if printErr != nil || printfErr != nil || printlnErr != nil {
		app.fail("Print returned error", "Info: stdout write failed")
		return
	}
	if stdoutRead != len(expectedPrint) || string(stdoutData[:stdoutRead]) != expectedPrint {
		app.fail("Print mismatch", "Info: expected "+trimTrailingNewline(expectedPrint))
		return
	}

	if formatErr.Error() != "fmt error 7" {
		app.fail("Errorf mismatch", "Info: expected fmt error 7")
		return
	}
	if errors.Unwrap(wrappedErr) != wrapCause {
		app.fail("Errorf wrap mismatch", "Info: expected single %w unwrap")
		return
	}
	if !errors.Is(multiWrappedErr, leftCause) || !errors.Is(multiWrappedErr, rightCause) {
		app.fail("Errorf multi-wrap mismatch", "Info: expected both %w causes in errors.Is")
		return
	}
	if invalidWrapText != "%!w(string=x)" {
		app.fail("Sprintf %w mismatch", "Info: expected %!w(string=x)")
		return
	}
	if quotedText != "\"fmt\"/\"go\\tfmt\"/\"K\\n\"/'\\n'/\"\\xffA\"/\"é\"/\"\\u00a0\"/\"\\u00ad\"" {
		app.fail("Sprintf %q mismatch", "Info: expected quoted Stringer, string, bytes, rune, invalid UTF-8, and Unicode printability parity")
		return
	}
	if quotedErrorText != "\"fmt error 7\"" {
		app.fail("Errorf %q mismatch", "Info: expected quoted error text")
		return
	}
	if sliceText != "[1 2 3]/[go fmt]/[75 10]/[fmt 7 true]" {
		app.fail("Sprintf slice %v mismatch", "Info: expected Go-style bracketed slice formatting")
		return
	}
	if widthText != "00002A/    42" {
		app.fail("Sprintf width mismatch", "Info: expected zero-padded hex and space-padded decimal width")
		return
	}
	if quotedASCIIText != "\"\\u263a\"/\"\\u263a\"/'\\u263a'" {
		app.fail("Sprintf %+q mismatch", "Info: expected ASCII-only quoted string, bytes, and rune")
		return
	}
	if rawQuotedText != "`abc`/`xy`/`a\tb`" {
		app.fail("Sprintf %#q mismatch", "Info: expected raw-string quoting when backquote-safe")
		return
	}
	if scanned != 3 || scanWord != "scan" || scanValue != 42 || !scanOK {
		app.fail("Fscanln mismatch", "Info: expected scan/42/true")
		return
	}
	if stdinScanned != 3 || stdinWord != "stdin" || stdinValue != 7 || stdinOK {
		app.fail("Scanln mismatch", "Info: expected stdin/7/false")
		return
	}

	app.ok = true
	app.summary = "fmt probe ok / ordinary import fmt package resolved"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "fmt probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
