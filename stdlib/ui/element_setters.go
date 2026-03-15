package ui

import "kos"

// Style setters on Element that use UpdateStyle for change detection.

func (element *Element) SetWidth(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetWidth(value)
	})
}

func (element *Element) SetHeight(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetHeight(value)
	})
}

func (element *Element) SetLeft(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetLeft(value)
	})
}

func (element *Element) SetTop(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetTop(value)
	})
}

func (element *Element) SetRight(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetRight(value)
	})
}

func (element *Element) SetBottom(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBottom(value)
	})
}

func (element *Element) SetBackground(color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBackground(color)
	})
}

func (element *Element) SetForeground(color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetForeground(color)
	})
}

func (element *Element) SetBorderColor(color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBorderColor(color)
	})
}

func (element *Element) SetBorderWidth(width int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBorderWidth(width)
	})
}

func (element *Element) SetBorder(width int, color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBorder(width, color)
	})
}

func (element *Element) SetMargin(values ...int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetMargin(values...)
	})
}

func (element *Element) SetPadding(values ...int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetPadding(values...)
	})
}

func (element *Element) SetBorderRadius(values ...int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetBorderRadius(values...)
	})
}

func (element *Element) SetDisplay(value DisplayMode) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetDisplay(value)
	})
}

func (element *Element) SetPosition(value PositionMode) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetPosition(value)
	})
}

func (element *Element) SetTextAlign(value TextAlign) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetTextAlign(value)
	})
}

func (element *Element) SetTextShadow(value TextShadow) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetTextShadow(value)
	})
}

func (element *Element) SetTextShadowPtr(value *TextShadow) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetTextShadowPtr(value)
	})
}

func (element *Element) SetGradient(value Gradient) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetGradient(value)
	})
}

func (element *Element) SetGradientPtr(value *Gradient) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetGradientPtr(value)
	})
}

func (element *Element) SetShadow(value Shadow) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetShadow(value)
	})
}

func (element *Element) SetShadowPtr(value *Shadow) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetShadowPtr(value)
	})
}

func (element *Element) SetOpacity(value uint8) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetOpacity(value)
	})
}

func (element *Element) SetOverflow(value OverflowMode) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetOverflow(value)
	})
}

func (element *Element) SetOverflowX(value OverflowMode) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetOverflowX(value)
	})
}

func (element *Element) SetOverflowY(value OverflowMode) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetOverflowY(value)
	})
}

func (element *Element) SetScrollbarWidth(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetScrollbarWidth(value)
	})
}

func (element *Element) SetScrollbarTrack(color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetScrollbarTrack(color)
	})
}

func (element *Element) SetScrollbarThumb(color kos.Color) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetScrollbarThumb(color)
	})
}

func (element *Element) SetScrollbarRadius(value int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetScrollbarRadius(value)
	})
}

func (element *Element) SetScrollbarPadding(values ...int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.SetScrollbarPadding(values...)
	})
}

func (element *Element) SetAbsolute(x int, y int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.Absolute(x, y)
	})
}

func (element *Element) SetSize(width int, height int) bool {
	return element.UpdateStyle(func(style *Style) {
		style.Size(width, height)
	})
}
