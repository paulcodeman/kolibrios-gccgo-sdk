package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	application := app.New()
	window := application.NewWindow("Fyne Hello")

	status := widget.NewLabel("Status: idle")
	input := widget.NewEntry()
	input.SetPlaceHolder("Type text and press Apply")

	clicks := 0
	button := widget.NewButton("Apply", func() {
		clicks++
		value := input.Text
		if value == "" {
			value = "(empty)"
		}
		status.SetText(fmt.Sprintf("Clicks: %d | Text: %s", clicks, value))
	})

	window.SetContent(container.NewVBox(
		widget.NewLabel("Fyne 2.2.4 on KolibriOS"),
		status,
		input,
		button,
	))
	window.Resize(fyne.NewSize(320, 140))
	window.ShowAndRun()
}
