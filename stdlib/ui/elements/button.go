package elements

import (
	"kos"
	"ui"
)

// Button creates a neutral button that uses CreateElement defaults
// and adds hover/active visuals similar to standard UI buttons.
func Button(text string) *ui.Element {
	button := ui.CreateButton()
	button.Label = text
	button.Text = ""
	return button
}

// ButtonAt creates a button at a fixed position with the given Kolibri button ID.
func ButtonAt(id kos.ButtonID, label string, x int, y int) *ui.Element {
	button := ui.CreateButton()
	button.ID = id
	button.Label = label
	button.Text = ""
	button.SetLeft(x)
	button.SetTop(y)
	return button
}

// ButtonStyle returns a neutral base style if you want to override defaults.
func ButtonStyle() ui.Style {
	return ui.DefaultButtonStyle()
}

func ButtonHoverStyle() ui.Style {
	return ui.DefaultButtonHoverStyle()
}

func ButtonActiveStyle() ui.Style {
	return ui.DefaultButtonActiveStyle()
}
