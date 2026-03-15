package main

import (
	"fmt"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	libImgButtonExit    kos.ButtonID = 1
	libImgButtonRefresh kos.ButtonID = 2

	libImgWindowTitle  = "KolibriOS LIBIMG Demo"
	libImgWindowX      = 180
	libImgWindowY      = 108
	libImgWindowWidth  = 980
	libImgWindowHeight = 520

	libImgSamplePath = "/sys/ICONS32.PNG"
)

type App struct {
	summary       string
	imageLine     string
	convertLine   string
	infoLine      string
	ok            bool
	refreshBtn    *ui.Element
	lib           kos.LibImg
	libLoaded     bool
	image         kos.ImageHandle
	converted     kos.ImageHandle
	imageLoadPath string
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(libImgButtonRefresh, "Refresh", 28, 464)
	refresh.SetWidth(116)

	app := App{
		refreshBtn:    refresh,
		imageLoadPath: libImgSamplePath,
	}
	app.refreshImage()
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
	case libImgButtonRefresh:
		if !app.image.Valid() || !app.converted.Valid() {
			app.refreshImage()
		}
		app.Redraw()
	case libImgButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(libImgButtonExit, "Exit", 162, 464)
	exit.SetWidth(92)

	kos.BeginRedraw()
	kos.OpenWindow(libImgWindowX, libImgWindowY, libImgWindowWidth, libImgWindowHeight, libImgWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "LIBIMG.OBJ image load/convert/draw through the typed kos wrapper")
	kos.DrawText(28, 92, ui.Aqua, app.imageLine)
	kos.DrawText(28, 114, ui.Lime, app.convertLine)
	kos.DrawText(28, 136, ui.Yellow, app.infoLine)
	kos.DrawText(28, 160, ui.Black, "The demo loads /sys/ICONS32.PNG through libimg, converts it to 32bpp, and draws both images into the current window")
	if app.image.Valid() {
		_ = app.lib.Draw(app.image, 28, 196, app.image.Width(), app.image.Height(), 0, 0)
	}
	if app.converted.Valid() {
		_ = app.lib.Draw(app.converted, 28+app.image.Width()+24, 196, app.converted.Width(), app.converted.Height(), 0, 0)
	}
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshImage() {
	var lib kos.LibImg
	var ok bool
	var image kos.ImageHandle
	var converted kos.ImageHandle

	lib, ok = kos.LoadLibImg()
	if !ok {
		app.fail("libimg.obj unavailable", "Info: failed to load "+kos.LibImgDLLPath)
		return
	}

	image, ok = lib.FromFile(app.imageLoadPath)
	if !ok {
		app.fail("img_from_file failed", "Info: could not load "+app.imageLoadPath)
		return
	}

	converted, ok = lib.Convert(image, kos.ImageTypeBPP32)
	if !ok {
		app.fail("img_convert failed", "Info: conversion to 32bpp returned nil")
		return
	}

	app.lib = lib
	app.libLoaded = true
	app.image = image
	app.converted = converted
	app.ok = true
	app.summary = "libimg probe ok / png load convert draw paths active"
	app.imageLine = fmt.Sprintf("Image: %s / %dx%d / type=%d / count=%d", app.imageLoadPath, image.Width(), image.Height(), image.Type(), lib.Count(image))
	app.convertLine = fmt.Sprintf("Convert: %dx%d / type=%d", converted.Width(), converted.Height(), converted.Type())
	app.infoLine = fmt.Sprintf("Info: %s / version 0x%x / table 0x%x", kos.LibImgDLLPath, lib.Version(), uint32(lib.ExportTable()))
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "libimg probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}
