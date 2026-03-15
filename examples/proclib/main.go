package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	procLibButtonExit      kos.ButtonID = 1
	procLibButtonOpen      kos.ButtonID = 2
	procLibButtonSave      kos.ButtonID = 3
	procLibButtonSelectDir kos.ButtonID = 4
	procLibButtonColor     kos.ButtonID = 5

	procLibWindowTitle  = "KolibriOS PROC_LIB Demo"
	procLibWindowX      = 204
	procLibWindowY      = 132
	procLibWindowWidth  = 920
	procLibWindowHeight = 356
)

type App struct {
	summary     string
	dialogLine  string
	fileLine    string
	colorLine   string
	noteLine    string
	ok          bool
	openBtn     *ui.Element
	saveBtn     *ui.Element
	dirBtn      *ui.Element
	colorBtn    *ui.Element
	lib         kos.ProcLib
	libLoaded   bool
	openDialog  *kos.OpenDialog
	colorDialog *kos.ColorDialog
}

func NewApp() App {
	openBtn := elements.ButtonAt(procLibButtonOpen, "Open file", 28, 300)
	openBtn.SetWidth(122)

	saveBtn := elements.ButtonAt(procLibButtonSave, "Save as", 166, 300)
	saveBtn.SetWidth(112)

	dirBtn := elements.ButtonAt(procLibButtonSelectDir, "Select dir", 294, 300)
	dirBtn.SetWidth(126)

	colorBtn := elements.ButtonAt(procLibButtonColor, "Color", 436, 300)
	colorBtn.SetWidth(100)

	app := App{
		openBtn:  openBtn,
		saveBtn:  saveBtn,
		dirBtn:   dirBtn,
		colorBtn: colorBtn,
	}
	app.resetState()
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
	case procLibButtonOpen:
		app.runOpenDialog(kos.OpenDialogOpen)
		app.Redraw()
	case procLibButtonSave:
		app.runOpenDialog(kos.OpenDialogSave)
		app.Redraw()
	case procLibButtonSelectDir:
		app.runOpenDialog(kos.OpenDialogSelectDirectory)
		app.Redraw()
	case procLibButtonColor:
		app.runColorDialog()
		app.Redraw()
	case procLibButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(procLibButtonExit, "Exit", 792, 300)
	exit.SetWidth(92)

	kos.BeginRedraw()
	kos.OpenWindow(procLibWindowX, procLibWindowY, procLibWindowWidth, procLibWindowHeight, procLibWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample uses kos.LoadProcLib() for PROC_LIB.OBJ open/save/folder dialogs and the color picker")
	kos.DrawText(28, 92, ui.Aqua, app.dialogLine)
	kos.DrawText(28, 114, ui.Lime, app.fileLine)
	kos.DrawText(28, 136, ui.Yellow, app.colorLine)
	kos.DrawText(28, 158, ui.Black, app.noteLine)
	kos.DrawText(28, 184, ui.Black, "Open/Save/Select dir call OpenDialog_start and block until the modal helper exits")
	kos.DrawText(28, 206, ui.Black, "Color calls ColorDialog_start and reports the selected RGB value when status=ok")
	app.openBtn.Draw()
	app.saveBtn.Draw()
	app.dirBtn.Draw()
	app.colorBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) resetState() {
	app.ok = false
	app.summary = "proclib wrapper ready / choose a dialog to launch"
	app.dialogLine = "Dialog: no result yet"
	app.fileLine = "File: no path selected yet"
	app.colorLine = "Color: no color selected yet"
	app.noteLine = "Info: PROC_LIB.OBJ dialogs are initialized once and then reused across repeated launches"
}

func (app *App) runOpenDialog(mode kos.OpenDialogMode) {
	if !app.ensureDialogs() {
		return
	}

	dialog := app.openDialog
	dialog.SetMode(mode)
	if mode == kos.OpenDialogSave {
		if !app.lib.SetOpenDialogFileName(dialog, "go-proclib.tmp") || !app.lib.SetOpenDialogFileExtension(dialog, "txt") {
			app.fail("OpenDialog_set_file_* failed", "Info: save-dialog suggested name/extension could not be prepared")
			return
		}
	}

	status, ok := app.lib.StartOpenDialog(dialog)
	if !ok {
		app.fail("OpenDialog_start failed", "Info: PROC_LIB open dialog export was not callable")
		return
	}

	app.ok = status != kos.OpenDialogAlternative
	app.summary = "open dialog returned"
	app.dialogLine = fmt.Sprintf("Dialog: %s / mode=%s / area=%s / com=0x%x", openDialogStatusText(status), openDialogModeText(mode), dialog.AreaName(), dialog.ComArea())
	app.fileLine = fmt.Sprintf("File: %q / name=%q / version 0x%x", dialog.FilePath(), dialog.FileName(), app.lib.OpenDialogVersion())
	app.colorLine = fmt.Sprintf("Color: PROC_LIB 0x%x / ready=%t", app.lib.Version(), app.lib.Ready())
	app.noteLine = "Info: save mode preloads go-proclib.txt and the same initialized dialog struct is reused for later launches"
}

func (app *App) runColorDialog() {
	if !app.ensureDialogs() {
		return
	}

	dialog := app.colorDialog

	status, ok := app.lib.StartColorDialog(dialog)
	if !ok {
		app.fail("ColorDialog_start failed", "Info: PROC_LIB color dialog export was not callable")
		return
	}

	app.ok = status != kos.ColorDialogAlternative
	app.summary = "color dialog returned"
	app.dialogLine = fmt.Sprintf("Dialog: %s / area=%s / com=0x%x", colorDialogStatusText(status), dialog.AreaName(), dialog.ComArea())
	app.fileLine = fmt.Sprintf("File: PROC_LIB open-dialog version 0x%x / color-dialog version 0x%x", app.lib.OpenDialogVersion(), app.lib.ColorDialogVersion())
	app.colorLine = fmt.Sprintf("Color: #%06X / type=%d / lib 0x%x", dialog.Color()&0xFFFFFF, dialog.ColorType(), app.lib.Version())
	app.noteLine = "Info: the color dialog is initialized once with #00A0FF and then reused for later modal launches"
}

func (app *App) ensureDialogs() bool {
	if app.libLoaded {
		return true
	}

	lib, ok := kos.LoadProcLib()
	if !ok {
		app.fail("proc_lib.obj unavailable", "Info: failed to load "+kos.ProcLibDLLPath)
		return false
	}

	openDialog := kos.NewOpenDialog(kos.OpenDialogOpen, 60, 60, 420, 320)
	openDialog.SetDirectory("/sys")
	openDialog.SetDefaultDirectory("/sys")
	if !lib.InitOpenDialog(openDialog) {
		app.fail("OpenDialog_init failed", "Info: communication area for PROC_LIB open dialog was not created")
		return false
	}

	colorDialog := kos.NewColorDialog(kos.ColorDialogPaletteAndTone, 80, 80, 420, 320)
	colorDialog.SetColor(0x00A0FF)
	if !lib.InitColorDialog(colorDialog) {
		app.fail("ColorDialog_init failed", "Info: communication area for PROC_LIB color dialog was not created")
		return false
	}

	app.lib = lib
	app.libLoaded = true
	app.openDialog = openDialog
	app.colorDialog = colorDialog
	return true
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "proclib probe failed / " + detail
	app.noteLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func openDialogModeText(mode kos.OpenDialogMode) string {
	switch mode {
	case kos.OpenDialogOpen:
		return "open"
	case kos.OpenDialogSave:
		return "save"
	case kos.OpenDialogSelectDirectory:
		return "dir"
	default:
		return fmt.Sprintf("%d", uint32(mode))
	}
}

func openDialogStatusText(status kos.OpenDialogStatus) string {
	switch status {
	case kos.OpenDialogCanceled:
		return "cancel"
	case kos.OpenDialogOK:
		return "ok"
	case kos.OpenDialogAlternative:
		return "fallback"
	default:
		return fmt.Sprintf("%d", uint32(status))
	}
}

func colorDialogStatusText(status kos.ColorDialogStatus) string {
	switch status {
	case kos.ColorDialogCanceled:
		return "cancel"
	case kos.ColorDialogOK:
		return "ok"
	case kos.ColorDialogAlternative:
		return "fallback"
	default:
		return fmt.Sprintf("%d", uint32(status))
	}
}

func main() {
	app := NewApp()
	app.Run()
}
