package elements

import "ui"

func Range(min int, max int, value int) *ui.Element {
	element := ui.CreateRange()
	element.SetRangeBounds(min, max)
	element.SetValue(value)
	return element
}

func RangeStyle() ui.Style {
	return ui.DefaultRangeStyle()
}
