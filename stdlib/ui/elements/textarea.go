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
	return ui.DefaultTextareaStyle()
}
