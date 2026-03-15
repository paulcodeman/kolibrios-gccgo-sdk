package main

import (
	"fmt"
	"os"
	"sort"
	"ui/elements"

	"kos"
	"ui"
)

const (
	sortButtonExit    kos.ButtonID = 1
	sortButtonRefresh kos.ButtonID = 2

	sortWindowTitle  = "KolibriOS Sort Demo"
	sortWindowX      = 214
	sortWindowY      = 128
	sortWindowWidth  = 900
	sortWindowHeight = 332
)

type record struct {
	name string
	size int
}

type recordSlice []record

type App struct {
	summary     string
	intsLine    string
	stringsLine string
	stableLine  string
	searchLine  string
	infoLine    string
	ok          bool
	refreshBtn  *ui.Element
}

func (records recordSlice) Len() int {
	return len(records)
}

func (records recordSlice) Less(i int, j int) bool {
	if records[i].size != records[j].size {
		return records[i].size < records[j].size
	}

	return recordNameLess(records[i].name, records[j].name)
}

func (records recordSlice) Swap(i int, j int) {
	records[i], records[j] = records[j], records[i]
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(sortButtonRefresh, "Refresh", 28, 278)
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
	case sortButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case sortButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(sortButtonExit, "Exit", 170, 278)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(sortWindowX, sortWindowY, sortWindowWidth, sortWindowHeight, sortWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary sort package")
	kos.DrawText(28, 92, ui.Aqua, app.intsLine)
	kos.DrawText(28, 114, ui.Lime, app.stringsLine)
	kos.DrawText(28, 136, ui.Yellow, app.stableLine)
	kos.DrawText(28, 158, ui.White, app.searchLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	info, err := os.Stat("/sys/default.skn")
	if err != nil {
		app.fail("stat failed", "Info: "+err.Error())
		return
	}

	currentFolder, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed", "Info: "+err.Error())
		return
	}

	ints := []int{42, int(info.Size()), 7, 1, 19}
	sort.Ints(ints)
	if !sort.IntsAreSorted(ints) {
		app.fail("Ints mismatch", "Info: IntsAreSorted returned false")
		return
	}

	names := []string{"sys", "diag", "base64", "console"}
	sort.Strings(names)
	if !sort.StringsAreSorted(names) {
		app.fail("Strings mismatch", "Info: StringsAreSorted returned false")
		return
	}

	records := recordSlice{
		{name: "probe", size: 2},
		{name: "cwd", size: len(currentFolder)},
		{name: "skin", size: 1},
		{name: "base64", size: 2},
	}
	sort.Stable(records)

	descending := sort.IntSlice{1, 3, 2}
	sort.Sort(sort.Reverse(descending))

	indexInt := sort.SearchInts(ints, 19)
	indexString := sort.SearchStrings(names, "console")
	indexGeneric := sort.Search(len(ints), func(index int) bool {
		return ints[index] >= int(info.Size())
	})

	if ints[0] != 1 || ints[len(ints)-1] != int(info.Size()) {
		app.fail("sorted ints mismatch", "Info: unexpected Ints result")
		return
	}
	if names[0] != "base64" || names[len(names)-1] != "sys" {
		app.fail("sorted strings mismatch", "Info: unexpected Strings result")
		return
	}
	if records[0].name != "skin" || records[1].name != "base64" || records[2].name != "probe" {
		app.fail("stable mismatch", "Info: Stable did not preserve equal-key order")
		return
	}
	if descending[0] != 3 || descending[2] != 1 {
		app.fail("reverse mismatch", "Info: Reverse sort contract mismatch")
		return
	}
	if indexInt != 2 || indexString != 1 || indexGeneric != len(ints)-1 {
		app.fail("search mismatch", "Info: Search helper results differ from expected order")
		return
	}

	app.ok = true
	app.summary = "sort probe ok / ordinary import sort resolved"
	app.intsLine = fmt.Sprintf("Ints: %v / sorted %v / reverse %v", ints, sort.IntsAreSorted(ints), []int(descending))
	app.stringsLine = fmt.Sprintf("Strings: %v / sorted %v", names, sort.StringsAreSorted(names))
	app.stableLine = fmt.Sprintf("Stable: [%s %s %s %s]", records[0].name, records[1].name, records[2].name, records[3].name)
	app.searchLine = fmt.Sprintf("Search: ints 19 -> %d / strings console -> %d / size -> %d", indexInt, indexString, indexGeneric)
	app.infoLine = "Info: interface sort, stable sort, reverse sort, and search helpers all stay on ordinary sort"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "sort probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func recordNameLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}

	for index := 0; index < limit; index++ {
		if left[index] == right[index] {
			continue
		}

		return left[index] < right[index]
	}

	return len(left) < len(right)
}
