package elements

import "ui"

func Radio(label string, group string, checked bool) *ui.Element {
	element := ui.CreateRadio()
	element.Label = label
	element.SetGroup(group)
	element.SetChecked(checked)
	return element
}

func RadioStyle() ui.Style {
	return ui.DefaultRadioStyle()
}
