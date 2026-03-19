package elements

import "ui"

// Input creates a simple text input using CreateElement defaults.
func Input(value string) *ui.Element {
	input := ui.CreateInput()
	input.Text = value
	return input
}

// InputStyle returns a neutral input style if you want to override defaults.
func InputStyle() ui.Style {
	return ui.DefaultInputStyle()
}
