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
	style := ui.Style{}
	style.SetBackground(ui.White)
	style.SetForeground(ui.Black)
	style.SetBorderColor(ui.Gray)
	style.SetBorderWidth(1)
	style.SetTextAlign(ui.TextAlignLeft)
	style.SetOverflow(ui.OverflowAuto)
	style.SetScrollbarWidth(6)
	style.SetScrollbarTrack(ui.Silver)
	style.SetScrollbarThumb(ui.Gray)
	style.SetScrollbarRadius(3)
	style.SetScrollbarPadding(1)
	style.SetPadding(6)
	return style
}
