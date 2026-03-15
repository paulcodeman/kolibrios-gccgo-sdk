package elements

import "ui"

// Label creates a neutral label using CreateElement defaults.
func Label(text string) *ui.Element {
	label := ui.CreateLabel()
	label.Text = text
	return label
}
