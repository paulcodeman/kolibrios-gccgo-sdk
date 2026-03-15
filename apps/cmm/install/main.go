package main

import (
	"kos"
	"ui"
	"ui/elements"
)

const (
	windowWidth  = 400
	windowHeight = 300

	btnInstall kos.ButtonID = 10
	btnExit    kos.ButtonID = 11
)

var introLines = []string{
	"Try a new visual design of KolibriOS.",
	"This will copy KolibriNext settings",
	"into /sys/settings and restart UI.",
	"Close other apps before install.",
}

const (
	srcBase = "/kolibrios/KolibriNext/settings"
	dstBase = "/sys/settings"
)

var copyFiles = []string{
	"app_plus.ini",
	"docky.ini",
	"icon.ini",
}

type App struct {
	x          int
	y          int
	clientX    int
	clientY    int
	clientW    int
	clientH    int
	status     string
	detail     string
	installed  bool
	installBtn *ui.Element
	exitBtn    *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() *App {
	width, height := kos.ScreenSize()
	x := (width - windowWidth) / 2
	y := (height - windowHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	clientX, clientY, clientW, clientH := windowClientRect(windowWidth, windowHeight)
	buttonX := clientX + (clientW-110)/2
	buttonY := clientY + clientH - 50

	install := elements.ButtonAt(btnInstall, "Install", buttonX, buttonY)
	install.SetSize(110, 28)
	install.SetBackground(ui.Blue)
	install.SetForeground(ui.White)

	exit := elements.ButtonAt(btnExit, "Exit", buttonX, buttonY)
	exit.SetSize(110, 28)
	exit.SetBackground(ui.Blue)
	exit.SetForeground(ui.White)

	return &App{
		x:          x,
		y:          y,
		clientX:    clientX,
		clientY:    clientY,
		clientW:    clientW,
		clientH:    clientH,
		status:     "Ready",
		installBtn: install,
		exitBtn:    exit,
	}
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
		case kos.EventKey:
			if app.handleKey() {
				return
			}
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case 1:
		kos.Exit()
		return true
	case btnInstall:
		if !app.installed {
			app.install()
			app.Redraw()
		}
	case btnExit:
		kos.Exit()
		return true
	}
	return false
}

func (app *App) handleKey() bool {
	key := kos.ReadKey()
	if key.Empty {
		return false
	}
	if key.Code == 27 || key.ScanCode == 1 {
		kos.Exit()
		return true
	}
	if key.Code == 13 && !app.installed {
		app.install()
		app.Redraw()
	}
	return false
}

func (app *App) install() {
	app.status = "Installing..."
	app.detail = ""
	ini, ok := kos.LoadINI()
	if !ok {
		app.status = "Install failed"
		app.detail = "libini.obj unavailable"
		return
	}

	if !ini.SetInt("/sys/settings/taskbar.ini", "Flags", "Attachment", 0) {
		app.status = "Install failed"
		app.detail = "taskbar.ini update failed"
		return
	}

	for _, name := range copyFiles {
		src := srcBase + "/" + name
		dst := dstBase + "/" + name
		data, status := kos.ReadAllFile(src)
		if status != kos.FileSystemOK && status != kos.FileSystemEOF {
			app.status = "Install failed"
			app.detail = "missing " + name
			return
		}
		written, status := kos.CreateOrRewriteFile(dst, data)
		if status != kos.FileSystemOK || written != uint32(len(data)) {
			app.status = "Install failed"
			app.detail = "write failed: " + name
			return
		}
	}

	kos.StartApplication("/sys/@icon", "", false)
	kos.StartApplication("/sys/@taskbar", "", false)
	kos.StartApplication("/sys/@docky", "", false)
	kos.StartApplication("/sys/media/kiv", "\\S__/kolibrios/res/Wallpapers/Free yourself.jpg", false)

	app.installed = true
	app.status = "Install complete"
	app.detail = ""
}

func (app *App) Redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(app.x, app.y, windowWidth, windowHeight, "KolibriN10 Installer")
	kos.FillRect(app.clientX, app.clientY, app.clientW, app.clientH, ui.White)

	drawTextLines(app.clientX+20, app.clientY+30, introLines, ui.Black)

	if app.detail != "" {
		drawTextLines(app.clientX+20, app.clientY+140, []string{app.status, app.detail}, ui.Red)
	} else {
		drawTextLines(app.clientX+20, app.clientY+140, []string{app.status}, ui.Gray)
	}

	if app.installed {
		app.exitBtn.Draw()
	} else {
		app.installBtn.Draw()
	}

	kos.EndRedraw()
}

func drawTextLines(x int, y int, lines []string, color kos.Color) {
	for i := 0; i < len(lines); i++ {
		kos.DrawText(x, y+i*18, color, lines[i])
	}
}

func windowClientRect(width int, height int) (int, int, int, int) {
	skin := kos.SkinHeight()
	x := 5
	y := skin
	w := width - 9
	h := height - skin - 4
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return x, y, w, h
}
