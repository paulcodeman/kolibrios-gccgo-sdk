package elements

import "ui"

func Progress(min int, max int, value int) *ui.Element {
	element := ui.CreateProgress()
	element.SetRangeBounds(min, max)
	element.SetValue(value)
	return element
}

func ProgressStyle() ui.Style {
	return ui.DefaultProgressStyle()
}
