package main

import (
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	runtimeCheckButtonExit    kos.ButtonID = 1
	runtimeCheckButtonRecheck kos.ButtonID = 2

	runtimeCheckWindowX      = 280
	runtimeCheckWindowY      = 180
	runtimeCheckWindowWidth  = 720
	runtimeCheckWindowHeight = 308
	runtimeCheckWindowTitle  = "KolibriOS Runtime Demo"
)

type sourceText interface {
	Text() string
}

type targetText interface {
	Text() string
}

type onText struct {
	text string
}

func (value onText) Text() string {
	return value.text
}

type offText struct {
	text string
}

func (value offText) Text() string {
	return value.text
}

type bridgeText struct {
	text string
}

func (value bridgeText) Text() string {
	return value.text
}

type runtimeMapPair struct {
	label string
	count int
}

type App struct {
	enabled        bool
	stringsOK      bool
	arraysOK       bool
	slicesOK       bool
	mapsOK         bool
	ifaceOK        bool
	emptyIfaceOK   bool
	assertionsOK   bool
	summary        string
	stringsLine    string
	arraysLine     string
	slicesLine     string
	mapsLine       string
	ifaceLine      string
	emptyIfaceLine string
	assertionsLine string
	recheck        *ui.Element
}

func NewApp() App {
	recheck := elements.ButtonAt(runtimeCheckButtonRecheck, "Recheck", 28, 250)
	recheck.SetWidth(132)

	app := App{
		enabled: true,
		recheck: recheck,
	}
	app.runChecks()
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
	case runtimeCheckButtonRecheck:
		app.enabled = !app.enabled
		app.runChecks()
		app.Redraw()
	case runtimeCheckButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(runtimeCheckButtonExit, "Exit", 182, 250)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(runtimeCheckWindowX, runtimeCheckWindowY, runtimeCheckWindowWidth, runtimeCheckWindowHeight, runtimeCheckWindowTitle)
	kos.DrawText(28, 48, app.summaryColor(), app.summary)
	kos.DrawText(28, 74, ui.Silver, "recheck toggles the string/int assertion branch and reruns the runtime smoke set")
	kos.DrawText(28, 104, app.statusColor(app.stringsOK), app.stringsLine)
	kos.DrawText(28, 124, app.statusColor(app.arraysOK), app.arraysLine)
	kos.DrawText(28, 144, app.statusColor(app.slicesOK), app.slicesLine)
	kos.DrawText(28, 164, app.statusColor(app.mapsOK), app.mapsLine)
	kos.DrawText(28, 184, app.statusColor(app.ifaceOK), app.ifaceLine)
	kos.DrawText(28, 204, app.statusColor(app.emptyIfaceOK), app.emptyIfaceLine)
	kos.DrawText(28, 224, app.statusColor(app.assertionsOK), app.assertionsLine)
	app.recheck.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) runChecks() {
	app.stringsOK, app.stringsLine = checkStrings(app.enabled)
	app.arraysOK, app.arraysLine = checkArrays(app.enabled)
	app.slicesOK, app.slicesLine = checkSlices(app.enabled)
	app.mapsOK, app.mapsLine = checkMaps(app.enabled)
	app.ifaceOK, app.ifaceLine = checkInterfaces(app.enabled)
	app.emptyIfaceOK, app.emptyIfaceLine = checkEmptyInterface(app.enabled)
	app.assertionsOK, app.assertionsLine = checkAssertions(app.enabled)

	if app.allOK() {
		if app.enabled {
			app.summary = "runtime ok / string assertion branch"
		} else {
			app.summary = "runtime ok / int assertion branch"
		}
		return
	}

	if app.enabled {
		app.summary = "runtime failure / string assertion branch"
	} else {
		app.summary = "runtime failure / int assertion branch"
	}
}

func (app *App) allOK() bool {
	return app.stringsOK &&
		app.arraysOK &&
		app.slicesOK &&
		app.mapsOK &&
		app.ifaceOK &&
		app.emptyIfaceOK &&
		app.assertionsOK
}

func (app *App) summaryColor() kos.Color {
	return app.statusColor(app.allOK())
}

func (app *App) statusColor(ok bool) kos.Color {
	if ok {
		return ui.Lime
	}

	return ui.Red
}

func checkStrings(enabled bool) (bool, string) {
	base := "runtime off"
	if enabled {
		base = "runtime on"
	}

	message := base + " / strings"
	expected := base + " / strings"
	if message == expected {
		return true, "strings: PASS / " + message
	}

	return false, "strings: FAIL / equality mismatch"
}

func checkArrays(enabled bool) (bool, string) {
	left := [4]byte{'k', 'o', 's', '!'}
	copyValue := left
	match := [4]byte{'k', 'o', 's', '!'}

	if enabled {
		copyValue[3] = '+'
	} else {
		copyValue[3] = '.'
	}

	if left == match && copyValue != match && len(left) == 4 && left[1] == 'o' {
		return true, "arrays : PASS / eq + copy ok"
	}

	return false, "arrays : FAIL / fixed array value mismatch"
}

func checkSlices(enabled bool) (bool, string) {
	base := "slice off"
	if enabled {
		base = "slice on"
	}

	src := []byte(base)
	buf := make([]byte, 0, 2)
	buf = append(buf, src...)
	if enabled {
		buf = append(buf, '!')
	} else {
		buf = append(buf, '.')
	}

	out := make([]byte, len(buf))
	copied := copy(out, buf)
	text := string(out)
	expected := base + "."
	if enabled {
		expected = base + "!"
	}

	if copied == len(buf) && text == expected {
		return true, "slices : PASS / " + text
	}

	return false, "slices : FAIL / copy or growth mismatch"
}

func checkMaps(enabled bool) (bool, string) {
	small := make(map[string]int)
	small["alpha"] = 1
	if enabled {
		small["beta"] = small["alpha"] + 2
	} else {
		small["beta"] = small["alpha"] + 4
	}
	delete(small, "alpha")
	if _, ok := small["alpha"]; ok {
		return false, "maps   : FAIL / string delete left comma-ok hit"
	}

	hinted := make(map[int]runtimeMapPair, 100)
	hinted[7] = runtimeMapPair{label: "seven", count: 7}
	deleteValue := runtimeMapPair{label: "nine", count: 9}
	if !enabled {
		deleteValue = runtimeMapPair{label: "nine", count: 11}
	}
	hinted[9] = deleteValue
	delete(hinted, 9)
	if _, ok := hinted[9]; ok {
		return false, "maps   : FAIL / int delete left comma-ok hit"
	}

	sum := 0
	seenSeven := false
	for key, value := range hinted {
		sum += key + value.count
		if key == 7 && value.label == "seven" {
			seenSeven = true
		}
	}

	expected := 17
	if !enabled {
		expected = 19
	}
	if small["beta"] == expected-14 && seenSeven && sum == 14 {
		return true, "maps   : PASS / string int delete range"
	}

	return false, "maps   : FAIL / assign lookup or range mismatch"
}

func checkInterfaces(enabled bool) (bool, string) {
	var source sourceText
	var mirror sourceText

	if enabled {
		source = onText{text: "iface on"}
		mirror = onText{text: "iface on"}
	} else {
		source = offText{text: "iface off"}
		mirror = offText{text: "iface off"}
	}

	if source == mirror {
		return true, "iface  : PASS / " + source.Text() + " / eq ok"
	}

	return false, "iface  : FAIL / dispatch or equality mismatch"
}

func checkEmptyInterface(enabled bool) (bool, string) {
	var left interface{}
	var right interface{}

	if enabled {
		left = "empty on"
		right = "empty on"
		if left == right {
			return true, "empty  : PASS / string eq ok"
		}
		return false, "empty  : FAIL / string equality mismatch"
	}

	left = 2026
	right = 2026
	if left == right {
		return true, "empty  : PASS / int eq ok"
	}

	return false, "empty  : FAIL / int equality mismatch"
}

func checkAssertions(enabled bool) (bool, string) {
	var directValue interface{}
	var candidate interface{}
	var anySource interface{}
	var ifaceSource sourceText
	var candidateText string
	var okString bool
	var switchPart string

	directValue = "direct string"
	anySource = bridgeText{text: "empty->iface"}
	ifaceSource = bridgeText{text: "iface->iface"}

	forcedString := directValue.(string)
	forcedAny := anySource.(targetText)
	anyTarget, okAny := anySource.(targetText)
	forcedIface := ifaceSource.(targetText)
	ifaceTarget, okIface := ifaceSource.(targetText)

	if enabled {
		candidate = "switch string"
		candidateText, okString = candidate.(string)
		switchPart = describeAssertionValue(candidate)
		if forcedString == "direct string" &&
			okString &&
			candidateText == "switch string" &&
			forcedAny.Text() == "empty->iface" &&
			okAny &&
			anyTarget.Text() == "empty->iface" &&
			forcedIface.Text() == "iface->iface" &&
			okIface &&
			ifaceTarget.Text() == "iface->iface" &&
			switchPart == "switch string" {
			return true, "assert : PASS / direct / e2i / i2i / string ok / " + switchPart
		}

		return false, "assert : FAIL / string assertion branch mismatch"
	}

	candidate = 2026
	_, okString = candidate.(string)
	switchPart = describeAssertionValue(candidate)
	_, missIface := candidate.(targetText)
	if forcedString == "direct string" &&
		!okString &&
		!missIface &&
		forcedAny.Text() == "empty->iface" &&
		okAny &&
		anyTarget.Text() == "empty->iface" &&
		forcedIface.Text() == "iface->iface" &&
		okIface &&
		ifaceTarget.Text() == "iface->iface" &&
		switchPart == "switch int" {
		return true, "assert : PASS / direct / e2i / i2i / string miss / " + switchPart
	}

	return false, "assert : FAIL / int assertion branch mismatch"
}

func describeAssertionValue(value interface{}) string {
	switch value.(type) {
	case string:
		return "switch string"
	case int:
		return "switch int"
	default:
		return "switch default"
	}
}

func Run() {
	app := NewApp()
	app.Run()
}
