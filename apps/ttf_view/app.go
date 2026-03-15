package main

import (
	"os"
	"path"
	"strconv"

	"kos"
	"ui"
	"ui/elements"
)

const (
	windowWidth  = 520
	windowHeight = 460
)

func Run() {
	fontPath := resolveFontPath()
	if fontPath == "" {
		selected, ok := chooseFontFile()
		if !ok {
			return
		}
		fontPath = selected
	}
	showFontViewer(fontPath)
}

func resolveFontPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return ""
}

func chooseFontFile() (string, bool) {
	lib, ok := kos.LoadProcLib()
	if !ok {
		return "", false
	}
	dialog := kos.NewOpenDialog(kos.OpenDialogOpen, 60, 60, 420, 320)
	dialog.SetDirectory("/")
	dialog.SetDefaultDirectory("/")
	if !lib.InitOpenDialog(dialog) {
		return "", false
	}
	lib.SetOpenDialogFileExtension(dialog, "ttf")
	status, ok := lib.StartOpenDialog(dialog)
	if !ok || status != kos.OpenDialogOK {
		return "", false
	}
	filePath := dialog.FilePath()
	if filePath == "" {
		return "", false
	}
	return filePath, true
}

func showFontViewer(fontPath string) {
	window := ui.NewWindowDefault()
	window.UpdateStyle(func(style *ui.Style) {
		style.SetWidth(windowWidth)
		style.SetHeight(windowHeight)
		style.SetOverflow(ui.OverflowAuto)
		style.SetBackground(ui.White)
	})

	fileName := path.Base(fontPath)
	if fileName == "" {
		fileName = fontPath
	}
	window.SetTitle("TTF Viewer - " + fileName)
	window.CenterOnScreen()

	root := ui.CreateBox()
	apply(root, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
	})

	title := elements.Label("TTF Viewer")
	apply(title, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetForeground(ui.Black)
		style.SetFontSize(20)
	})

	fileLine := elements.Label("File: " + fontPath)
	apply(fileLine, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 10, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	divider := ui.CreateBox()
	apply(divider, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(1)
		style.SetBackground(ui.Silver)
		style.SetMargin(4, 0, 10, 0)
	})

	sampleTitle := elements.Label("Sample")
	apply(sampleTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(14)
	})

	sampleBox := ui.CreateBox()
	apply(sampleBox, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
		style.SetBorderRadius(8)
		style.SetBorderWidth(1)
		style.SetBorderColor(ui.Silver)
		style.SetBackground(ui.White)
	})

	if _, err := os.Stat(fontPath); err != nil {
		errorLine := elements.Label("Font file not found")
		apply(errorLine, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(ui.Red)
			style.SetFontSize(14)
		})
		sampleBox.Append(errorLine)
	} else {
		addSampleLines(sampleBox, fontPath)
	}

	root.Append(title)
	root.Append(fileLine)
	root.Append(divider)
	root.Append(sampleTitle)
	root.Append(sampleBox)

	window.Append(root)

	window.Start()
}

func addSampleLines(container *ui.Element, fontPath string) {
	samples := []struct {
		size int
		text string
	}{
		{12, "The quick brown fox jumps over the lazy dog."},
		{14, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{14, "abcdefghijklmnopqrstuvwxyz"},
		{14, "0123456789 !@#$%^&*()"},
		{20, "The quick brown fox jumps over the lazy dog."},
		{28, "AaBbCcDdEeFfGg"},
		{40, "Sample"},
	}

	for i, sample := range samples {
		line := elements.Label(strconv.Itoa(sample.size) + "px  " + sample.text)
		apply(line, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			if i < len(samples)-1 {
				style.SetMargin(0, 0, 8, 0)
			}
			style.SetFontPath(fontPath)
			style.SetFontSize(sample.size)
			style.SetForeground(ui.Black)
		})
		container.Append(line)
	}
}

func apply(element *ui.Element, update func(*ui.Style)) {
	element.UpdateStyle(func(style *ui.Style) {
		if update != nil {
			update(style)
		}
	})
}
