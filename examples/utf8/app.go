package main

import (
	"fmt"
	"os"
	"strconv"
	"ui/elements"
	"unicode/utf8"

	"kos"
	"ui"
)

const (
	utf8ButtonExit    kos.ButtonID = 1
	utf8ButtonRefresh kos.ButtonID = 2

	utf8WindowTitle  = "KolibriOS UTF-8 Demo"
	utf8WindowX      = 216
	utf8WindowY      = 132
	utf8WindowWidth  = 900
	utf8WindowHeight = 344

	utf8ProbePath = "/sys/default.skn"
)

type App struct {
	summary    string
	decodeLine string
	countLine  string
	encodeLine string
	validLine  string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(utf8ButtonRefresh, "Refresh", 28, 288)
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
	case utf8ButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case utf8ButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(utf8ButtonExit, "Exit", 170, 288)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(utf8WindowX, utf8WindowY, utf8WindowWidth, utf8WindowHeight, utf8WindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary unicode/utf8 package")
	kos.DrawText(28, 92, ui.Aqua, app.decodeLine)
	kos.DrawText(28, 114, ui.Lime, app.countLine)
	kos.DrawText(28, 136, ui.Yellow, app.encodeLine)
	kos.DrawText(28, 158, ui.White, app.validLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, err := os.Stat(utf8ProbePath)
	if err != nil {
		app.fail("stat failed", "Info: "+err.Error())
		return
	}

	currentFolder, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed", "Info: "+err.Error())
		return
	}

	const euro = "\xe2\x82\xac"
	const smile = "\xf0\x9f\x98\x80"
	const sample = "A" + euro + smile
	const truncated = "\xe2\x82"
	const invalid = "\xffabc"

	firstRune, firstSize := utf8.DecodeRuneInString(euro)
	lastRune, lastSize := utf8.DecodeLastRuneInString(sample)
	count := utf8.RuneCountInString(sample)
	fullShort := utf8.FullRuneInString(truncated)
	fullEuro := utf8.FullRuneInString(euro)

	encoded := make([]byte, utf8.UTFMax)
	encodedSize := utf8.EncodeRune(encoded, rune(0x20AC))
	appended := string(utf8.AppendRune([]byte("append:"), rune(0x1F600)))

	invalidRune, invalidSize := utf8.DecodeRune([]byte{0xFF})
	validSample := utf8.ValidString(sample)
	validPath := utf8.ValidString(currentFolder) && utf8.ValidString(utf8ProbePath)

	if firstRune != rune(0x20AC) || firstSize != 3 {
		app.fail("DecodeRuneInString mismatch", "Info: expected euro U+20AC / 3")
		return
	}
	if lastRune != rune(0x1F600) || lastSize != 4 {
		app.fail("DecodeLastRuneInString mismatch", "Info: expected smile rune / 4")
		return
	}
	if count != 3 || len(sample) != 8 {
		app.fail("RuneCount mismatch", "Info: expected 3 runes / 8 bytes")
		return
	}
	if fullShort || !fullEuro {
		app.fail("FullRune mismatch", "Info: expected truncated=false / euro=true")
		return
	}
	if string(encoded[:encodedSize]) != euro || encodedSize != 3 {
		app.fail("EncodeRune mismatch", "Info: expected euro encoding length 3")
		return
	}
	if appended != "append:"+smile {
		app.fail("AppendRune mismatch", "Info: expected smile append text")
		return
	}
	if invalidRune != utf8.RuneError || invalidSize != 1 {
		app.fail("invalid decode mismatch", "Info: expected RuneError / 1")
		return
	}
	if !validSample || utf8.ValidString(invalid) || !validPath {
		app.fail("validity mismatch", "Info: expected sample/path valid and invalid bytes rejected")
		return
	}
	if utf8.RuneLen(rune(0x1F600)) != 4 || utf8.RuneLen(rune(0xD800)) != -1 {
		app.fail("RuneLen mismatch", "Info: expected smile=4 and surrogate=-1")
		return
	}
	if !utf8.ValidRune(rune(0x20AC)) || utf8.ValidRune(rune(0xD800)) {
		app.fail("ValidRune mismatch", "Info: expected euro valid and surrogate invalid")
		return
	}

	app.ok = true
	app.summary = "utf8 probe ok / ordinary import unicode/utf8 resolved"
	app.decodeLine = fmt.Sprintf("Decode: euro U+%x/%d / last U+%x/%d", int(firstRune), firstSize, int(lastRune), lastSize)
	app.countLine = fmt.Sprintf("Count: runes %d / bytes %d / full %v -> %v", count, len(sample), fullShort, fullEuro)
	app.encodeLine = fmt.Sprintf("Encode: %s / append %s / len %d", string(encoded[:encodedSize]), appended, utf8.RuneLen(rune(0x1F600)))
	app.validLine = fmt.Sprintf("Valid: sample %v / invalid %v / path %v", validSample, utf8.ValidString(invalid), validPath)
	app.infoLine = "Info: cwd " + currentFolder +
		" / size " + strconv.FormatInt(info.Size(), 10) +
		" / invalid U+" + strconv.FormatInt(int64(invalidRune), 16) +
		" / start " + strconv.FormatBool(utf8.RuneStart(euro[0])) + ":" + strconv.FormatBool(utf8.RuneStart(euro[1]))
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "utf8 probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
