package elements

import "ui"

// Textarea creates a multiline text area using CreateElement defaults.
func Textarea(value string) *ui.Element {
	textarea := ui.CreateTextarea()
	textarea.Text = value
	return textarea
}

// TextareaStyle returns a neutral textarea style if you want to override defaults.
func TextareaStyle() ui.Style {
	return ui.Style{
		Background:      ui.ColorPtr(ui.White),
		Foreground:      ui.ColorPtr(ui.Black),
		BorderColor:     ui.ColorPtr(ui.Gray),
		BorderWidth:     ui.IntPtr(1),
		TextAlign:       ui.AlignPtr(ui.TextAlignLeft),
		Overflow:        ui.OverflowPtr(ui.OverflowAuto),
		ScrollbarWidth:  ui.IntPtr(6),
		ScrollbarTrack:  ui.ColorPtr(ui.Silver),
		ScrollbarThumb:  ui.ColorPtr(ui.Gray),
		ScrollbarRadius: ui.IntPtr(3),
		ScrollbarPadding: &ui.Spacing{
			Left:   1,
			Top:    1,
			Right:  1,
			Bottom: 1,
		},
		Padding: &ui.Spacing{
			Left:   6,
			Top:    6,
			Right:  6,
			Bottom: 6,
		},
	}
}
