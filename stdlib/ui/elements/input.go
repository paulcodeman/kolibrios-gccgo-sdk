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
	return ui.Style{
		Background:  ui.ColorPtr(ui.White),
		Foreground:  ui.ColorPtr(ui.Black),
		BorderColor: ui.ColorPtr(ui.Gray),
		BorderWidth: ui.IntPtr(1),
		TextAlign:   ui.AlignPtr(ui.TextAlignLeft),
		Padding: &ui.Spacing{
			Left:   6,
			Top:    4,
			Right:  6,
			Bottom: 4,
		},
	}
}
