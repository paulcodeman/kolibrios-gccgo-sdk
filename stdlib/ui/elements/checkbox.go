package elements

import "ui"

func Checkbox(label string, checked bool) *ui.Element {
	element := ui.CreateCheckbox()
	element.Label = label
	element.SetChecked(checked)
	return element
}

func CheckboxStyle() ui.Style {
	return ui.DefaultCheckboxStyle()
}
