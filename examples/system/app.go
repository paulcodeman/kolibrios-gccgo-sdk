package main

import (
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	sysinfoButtonExit                  kos.ButtonID = 1
	sysinfoButtonToggleTitle           kos.ButtonID = 2
	sysinfoButtonRefresh               kos.ButtonID = 3
	sysinfoButtonFocusSelf             kos.ButtonID = 4
	sysinfoButtonReapplyLayout         kos.ButtonID = 5
	sysinfoButtonReapplySystemLanguage kos.ButtonID = 6
	sysinfoButtonApplySkinLegacy       kos.ButtonID = 7
	sysinfoButtonApplySkinUTF8         kos.ButtonID = 8
	sysinfoButtonCursorProbe           kos.ButtonID = 9

	sysinfoWindowX          = 350
	sysinfoWindowY          = 180
	sysinfoWindowWidth      = 540
	sysinfoWindowHeight     = 452
	sysinfoWindowTitle      = "KolibriOS System Demo"
	sysinfoUTF8Title        = "KolibriOS Система UTF-8"
	sysinfoCursorImageBytes = 32 * 32 * 4
)

type App struct {
	version               kos.KernelVersionInfo
	screenWidth           int
	screenHeight          int
	workArea              kos.Rect
	skinHeight            int
	skinMargins           kos.SkinMargins
	keyboardLanguage      kos.KeyboardLanguage
	systemLanguage        kos.KeyboardLanguage
	normalLayout          kos.KeyboardLayoutTable
	shiftLayout           kos.KeyboardLayoutTable
	altLayout             kos.KeyboardLayoutTable
	hasKeyboardLayouts    bool
	currentSlot           int
	activeSlot            int
	hasCurrentSlot        bool
	usingUTF8Title        bool
	focusStatus           string
	layoutStatus          string
	systemLanguageStatus  string
	skinLegacyStatus      string
	skinUTF8Status        string
	cursorStatus          string
	toggleTitle           *ui.Element
	refresh               *ui.Element
	focusSelf             *ui.Element
	reapplyLayout         *ui.Element
	reapplySystemLanguage *ui.Element
	applySkinLegacy       *ui.Element
	applySkinUTF8         *ui.Element
	cursorProbe           *ui.Element
}

func NewApp() App {
	toggleTitle := elements.ButtonAt(sysinfoButtonToggleTitle, "Use UTF-8", 28, 312)
	toggleTitle.SetWidth(128)

	refresh := elements.ButtonAt(sysinfoButtonRefresh, "Refresh", 176, 312)
	refresh.SetWidth(112)

	focusSelf := elements.ButtonAt(sysinfoButtonFocusSelf, "Focus self", 320, 312)
	focusSelf.SetWidth(120)

	reapplyLayout := elements.ButtonAt(sysinfoButtonReapplyLayout, "Reapply layout", 28, 344)
	reapplyLayout.SetWidth(144)

	reapplySystemLanguage := elements.ButtonAt(sysinfoButtonReapplySystemLanguage, "Reapply sys lang", 196, 344)
	reapplySystemLanguage.SetWidth(156)

	applySkinLegacy := elements.ButtonAt(sysinfoButtonApplySkinLegacy, "Skin 48.8", 28, 376)
	applySkinLegacy.SetWidth(128)

	applySkinUTF8 := elements.ButtonAt(sysinfoButtonApplySkinUTF8, "Skin 48.13", 172, 376)
	applySkinUTF8.SetWidth(136)

	cursorProbe := elements.ButtonAt(sysinfoButtonCursorProbe, "Cursor probe", 324, 376)
	cursorProbe.SetWidth(132)

	app := App{
		toggleTitle:           toggleTitle,
		refresh:               refresh,
		focusSelf:             focusSelf,
		reapplyLayout:         reapplyLayout,
		reapplySystemLanguage: reapplySystemLanguage,
		applySkinLegacy:       applySkinLegacy,
		applySkinUTF8:         applySkinUTF8,
		cursorProbe:           cursorProbe,
		focusStatus:           "ready",
		layoutStatus:          "ready",
		systemLanguageStatus:  "ready",
		skinLegacyStatus:      "idle",
		skinUTF8Status:        "idle",
		cursorStatus:          "idle",
	}
	app.refreshInfo()

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
	case sysinfoButtonToggleTitle:
		app.usingUTF8Title = !app.usingUTF8Title
		if app.usingUTF8Title {
			app.toggleTitle.Label = "Use ASCII"
			kos.SetWindowTitleWithEncodingPrefix(kos.EncodingUTF8, sysinfoUTF8Title)
		} else {
			app.toggleTitle.Label = "Use UTF-8"
			kos.SetWindowTitle(sysinfoWindowTitle)
		}
		app.Redraw()
	case sysinfoButtonRefresh:
		app.refreshInfo()
		app.focusStatus = "refreshed"
		app.layoutStatus = "reloaded"
		app.systemLanguageStatus = "reloaded"
		app.Redraw()
	case sysinfoButtonFocusSelf:
		app.focusSelfWindow()
		app.Redraw()
	case sysinfoButtonReapplyLayout:
		app.reapplyKeyboardLayout()
		app.Redraw()
	case sysinfoButtonReapplySystemLanguage:
		app.reapplySystemLanguageValue()
		app.Redraw()
	case sysinfoButtonApplySkinLegacy:
		app.applyDefaultSkinLegacy()
		app.Redraw()
	case sysinfoButtonApplySkinUTF8:
		app.applyDefaultSkinUTF8()
		app.Redraw()
	case sysinfoButtonCursorProbe:
		app.runCursorProbe()
		app.Redraw()
	case sysinfoButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(sysinfoWindowX, sysinfoWindowY, sysinfoWindowWidth, sysinfoWindowHeight, sysinfoWindowTitle)
	kos.DrawText(28, 46, ui.White, "Kernel version: "+formatKernelVersion(app.version))
	kos.DrawText(28, 64, ui.Silver, "Kernel ABI: "+formatKernelABI(app.version))
	kos.DrawText(28, 82, ui.Aqua, "Commit id: "+formatHex32(app.version.CommitID))
	kos.DrawText(28, 100, ui.Lime, "Debug tag: "+app.debugTagString())
	kos.DrawText(28, 118, ui.Yellow, "Screen size: "+formatInt(app.screenWidth)+"x"+formatInt(app.screenHeight))
	kos.DrawText(28, 136, ui.White, "Work area: "+formatRect(app.workArea))
	kos.DrawText(28, 154, ui.Silver, "Work size: "+formatInt(app.workArea.Width())+"x"+formatInt(app.workArea.Height()))
	kos.DrawText(28, 172, ui.Aqua, "Skin height: "+formatInt(app.skinHeight))
	kos.DrawText(28, 190, ui.Lime, "Skin margins: "+formatSkinMargins(app.skinMargins))
	kos.DrawText(28, 208, ui.Yellow, "Keyboard lang: "+formatKeyboardLanguage(app.keyboardLanguage))
	kos.DrawText(28, 226, ui.White, "System lang: "+formatKeyboardLanguage(app.systemLanguage))
	kos.DrawText(28, 244, ui.Aqua, "Layout sums: "+app.layoutChecksumsString())
	kos.DrawText(28, 262, ui.Silver, "Default skin path: "+kos.DefaultSkinPath)
	kos.DrawText(320, 46, ui.Yellow, "Title mode: "+app.titleMode())
	kos.DrawText(320, 64, ui.White, "Current slot: "+app.currentSlotString())
	kos.DrawText(320, 82, ui.Silver, "Active slot: "+formatInt(app.activeSlot))
	kos.DrawText(320, 100, ui.Aqua, "Focus state: "+app.focusStatus)
	kos.DrawText(320, 118, ui.Lime, "Layout state: "+app.layoutStatus)
	kos.DrawText(320, 136, ui.White, "System lang state: "+app.systemLanguageStatus)
	kos.DrawText(320, 154, ui.Yellow, "Skin 48.8 state: "+app.skinLegacyStatus)
	kos.DrawText(320, 172, ui.White, "Skin 48.13 state: "+app.skinUTF8Status)
	kos.DrawText(320, 190, ui.Aqua, "Cursor state: "+app.cursorStatus)
	kos.DrawText(320, 208, ui.Silver, "37.4/37.5/37.6 cursor round-trip")
	kos.DrawText(320, 226, ui.Silver, "48.8/48.13 apply /DEFAULT.SKN")
	kos.DrawText(320, 244, ui.Silver, "18.3 focuses a slot / 18.7 reports the active slot")
	app.toggleTitle.Draw()
	app.refresh.Draw()
	app.focusSelf.Draw()
	app.reapplyLayout.Draw()
	app.reapplySystemLanguage.Draw()
	app.applySkinLegacy.Draw()
	app.applySkinUTF8.Draw()
	app.cursorProbe.Draw()
	kos.EndRedraw()
}

func (app *App) refreshInfo() {
	var normalOK bool
	var shiftOK bool
	var altOK bool

	app.version = kos.KernelVersion()
	app.screenWidth, app.screenHeight = kos.ScreenSize()
	app.workArea = kos.ScreenWorkingArea()
	app.skinHeight = kos.SkinHeight()
	app.skinMargins = kos.WindowSkinMargins()
	app.keyboardLanguage = kos.KeyboardLayoutLanguage()
	app.systemLanguage = kos.SystemLanguage()
	app.normalLayout, normalOK = kos.ReadKeyboardLayoutTable(kos.KeyboardLayoutNormal)
	app.shiftLayout, shiftOK = kos.ReadKeyboardLayoutTable(kos.KeyboardLayoutShift)
	app.altLayout, altOK = kos.ReadKeyboardLayoutTable(kos.KeyboardLayoutAlt)
	app.hasKeyboardLayouts = normalOK && shiftOK && altOK
	app.currentSlot, app.hasCurrentSlot = kos.CurrentThreadSlotIndex()
	app.activeSlot = kos.ActiveWindowSlot()
}

func (app *App) debugTagString() string {
	if !app.version.IsDebug() {
		return "release"
	}

	return formatHex8(app.version.DebugTag)
}

func (app *App) titleMode() string {
	if app.usingUTF8Title {
		return "71.1 UTF-8 prefix"
	}

	return "71.2 direct encoding"
}

func (app *App) currentSlotString() string {
	if !app.hasCurrentSlot {
		return "-"
	}

	return formatInt(app.currentSlot)
}

func (app *App) focusSelfWindow() {
	if !app.hasCurrentSlot {
		app.focusStatus = "current slot unavailable"
		return
	}

	kos.FocusWindowSlot(app.currentSlot)
	app.refreshInfo()
	if app.activeSlot == app.currentSlot {
		app.focusStatus = "self active"
		return
	}

	app.focusStatus = "focus requested for slot " + formatInt(app.currentSlot)
}

func (app *App) layoutChecksumsString() string {
	if !app.hasKeyboardLayouts {
		return "unavailable"
	}

	return formatLayoutChecksums(app.normalLayout, app.shiftLayout, app.altLayout)
}

func (app *App) reapplyKeyboardLayout() {
	if !app.hasKeyboardLayouts {
		app.layoutStatus = "layout tables unavailable"
		return
	}

	if !kos.SetKeyboardLayoutTable(kos.KeyboardLayoutNormal, &app.normalLayout) {
		app.layoutStatus = "normal layout apply failed"
		return
	}

	if !kos.SetKeyboardLayoutTable(kos.KeyboardLayoutShift, &app.shiftLayout) {
		app.layoutStatus = "shift layout apply failed"
		return
	}

	if !kos.SetKeyboardLayoutTable(kos.KeyboardLayoutAlt, &app.altLayout) {
		app.layoutStatus = "alt layout apply failed"
		return
	}

	if !kos.SetKeyboardLayoutLanguage(app.keyboardLanguage) {
		app.layoutStatus = "language apply failed"
		return
	}

	app.refreshInfo()
	app.layoutStatus = "layout round-trip ok"
}

func (app *App) reapplySystemLanguageValue() {
	if !kos.SetSystemLanguage(app.systemLanguage) {
		app.systemLanguageStatus = "system language apply failed"
		return
	}

	app.refreshInfo()
	app.systemLanguageStatus = "system language round-trip ok"
}

func (app *App) applyDefaultSkinLegacy() {
	status := kos.SetSystemSkinLegacy(kos.DefaultSkinPath)
	if status != kos.FileSystemOK {
		app.skinLegacyStatus = formatFileSystemStatus(status)
		return
	}

	app.refreshInfo()
	app.skinLegacyStatus = "ok"
}

func (app *App) applyDefaultSkinUTF8() {
	status := kos.SetSystemSkin(kos.DefaultSkinPath)
	if status != kos.FileSystemOK {
		app.skinUTF8Status = formatFileSystemStatus(status)
		return
	}

	app.refreshInfo()
	app.skinUTF8Status = "ok"
}

func (app *App) runCursorProbe() {
	image := make([]byte, sysinfoCursorImageBytes)
	handle := kos.LoadCursorARGB(buildCursorProbeImage(image), 15, 15)
	if handle == 0 {
		app.cursorStatus = "load failed"
		return
	}

	previous := kos.SetCursor(handle)
	kos.RestoreDefaultCursor()
	kos.DeleteCursor(handle)
	app.cursorStatus = "ok prev=" + formatHex32(uint32(previous))
}

func buildCursorProbeImage(image []byte) []byte {
	center := 15

	for axis := 0; axis < 32; axis++ {
		setCursorProbePixel(image, center, axis, 0xFFFF4040)
		setCursorProbePixel(image, axis, center, 0xFFFF4040)
	}

	for axis := 12; axis <= 18; axis++ {
		setCursorProbePixel(image, center-1, axis, 0xFFFFFFFF)
		setCursorProbePixel(image, center+1, axis, 0xFFFFFFFF)
		setCursorProbePixel(image, axis, center-1, 0xFFFFFFFF)
		setCursorProbePixel(image, axis, center+1, 0xFFFFFFFF)
	}

	setCursorProbePixel(image, center, center, 0xFF101010)
	return image
}

func setCursorProbePixel(image []byte, x int, y int, argb uint32) {
	index := ((y * 32) + x) * 4

	image[index] = byte(argb)
	image[index+1] = byte(argb >> 8)
	image[index+2] = byte(argb >> 16)
	image[index+3] = byte(argb >> 24)
}
