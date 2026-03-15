package main

import (
	"kos"
	"ui"
	"ui/elements"
)

const (
	windowX      = 200
	windowY      = 200
	windowWidth  = 760
	windowHeight = 560
	windowTitle  = "C-- window example (Go)"

	exitButtonID kos.ButtonID = 1
	demoButtonID kos.ButtonID = 2

	listStartX     = 8
	listTitleY     = 12
	listStartY     = 32
	listLineHeight = 16
	listMaxEntries = 20

	asciiStartX     = 360
	asciiStartY     = 40
	asciiCols       = 20
	asciiCellWidth  = 16
	asciiCellHeight = 16
)

var (
	dirEntries []kos.FolderEntry
	dirPath    string
	dirStatus  kos.FileSystemStatus
	demoButton *ui.Element
)

func main() {
	loadDirectory()
	configureButton()

	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			redraw()
		case kos.EventButton:
			if handleButton(kos.CurrentButtonID()) {
				return
			}
		case kos.EventKey:
			if handleKey() {
				return
			}
		}
	}
}

func configureButton() {
	demoButton = elements.ButtonAt(demoButtonID, "Button", 100, 10)
	demoButton.SetSize(100, 22)
	demoButton.SetBackground(ui.Silver)
	demoButton.SetForeground(ui.Black)
}

func loadDirectory() {
	dirPath = kos.CurrentFolder()
	if dirPath == "" {
		dirPath = "/"
	}

	result, status := kos.ReadDirectory(dirPath, 0, 64)
	dirStatus = status
	if status == kos.FileSystemOK || status == kos.FileSystemEOF {
		dirEntries = result.Entries
		return
	}

	dirEntries = nil
}

func handleButton(id kos.ButtonID) bool {
	switch id {
	case exitButtonID:
		kos.Exit()
		return true
	case demoButtonID:
		loadDirectory()
		redraw()
	}

	return false
}

func handleKey() bool {
	key := kos.ReadKey()
	if key.Empty {
		return false
	}

	if key.Code == 27 || key.ScanCode == 1 {
		kos.Exit()
		return true
	}

	return false
}

func redraw() {
	kos.BeginRedraw()
	kos.OpenWindow(windowX, windowY, windowWidth, windowHeight, windowTitle)
	drawDirectory()
	drawWidgets()
	drawASCII()
	kos.EndRedraw()
}

func drawDirectory() {
	kos.DrawText(listStartX, listTitleY, ui.Black, "Dir: "+dirPath)
	if dirStatus != kos.FileSystemOK && dirStatus != kos.FileSystemEOF {
		kos.DrawText(listStartX, listStartY, ui.Red, "ReadDirectory failed: "+itoa(int(dirStatus)))
		return
	}

	y := listStartY
	for index := 0; index < len(dirEntries) && index < listMaxEntries; index++ {
		entry := dirEntries[index]
		name := entry.Name
		if entry.Info.Attributes&kos.FileAttributeDirectory != 0 {
			name += "/"
		}
		kos.DrawText(listStartX, y, ui.Fuchsia, name)
		y += listLineHeight
	}
}

func drawWidgets() {
	demoButton.Draw()
	kos.DrawText(100, 50, ui.Black, "Textline small")
	kos.DrawText(100, 70, ui.Black, "Textline big")
	kos.FillRect(100, 110, 100, 100, kos.Color(0x66AF86))
}

func drawASCII() {
	for value := 0; value < 256; value++ {
		row := value / asciiCols
		col := value % asciiCols
		x := asciiStartX + col*asciiCellWidth
		y := asciiStartY + row*asciiCellHeight
		kos.DrawText(x, y, ui.Black, asciiChar(value))
	}
}

func asciiChar(value int) string {
	if value < 32 || value > 126 {
		return "."
	}

	return string([]byte{byte(value)})
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	negative := false
	if value < 0 {
		negative = true
		value = -value
	}

	var buf [20]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}

	if negative {
		index--
		buf[index] = '-'
	}

	return string(buf[index:])
}
