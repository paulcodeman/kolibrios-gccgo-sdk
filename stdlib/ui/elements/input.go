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
	style := ui.Style{}
	style.SetBackground(ui.White)
	style.SetForeground(ui.Black)
	style.SetBorderColor(ui.Gray)
	style.SetBorderWidth(1)
	style.SetTextAlign(ui.TextAlignLeft)
	style.SetPadding(4, 6, 4, 6)
	return style
}
