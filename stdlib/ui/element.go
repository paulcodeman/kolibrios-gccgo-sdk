package ui

func DefaultButtonStyle() Style {
	style := Style{}
	style.SetBackground(Silver)
	style.SetForeground(Black)
	style.SetBorderColor(Gray)
	style.SetBorderWidth(1)
	style.SetTextAlign(TextAlignCenter)
	style.SetPadding(4, 6, 4, 6)
	return style
}

func DefaultLabelStyle() Style {
	style := Style{}
	style.SetForeground(Black)
	style.SetTextAlign(TextAlignLeft)
	return style
}

func DefaultInputStyle() Style {
	style := Style{}
	style.SetBackground(White)
	style.SetForeground(Black)
	style.SetBorderColor(Gray)
	style.SetBorderWidth(1)
	style.SetTextAlign(TextAlignLeft)
	style.SetPadding(4, 6, 4, 6)
	return style
}

func DefaultTextareaStyle() Style {
	style := Style{}
	style.SetBackground(White)
	style.SetForeground(Black)
	style.SetBorderColor(Gray)
	style.SetBorderWidth(1)
	style.SetTextAlign(TextAlignLeft)
	style.SetOverflow(OverflowAuto)
	style.SetScrollbarWidth(6)
	style.SetScrollbarTrack(Silver)
	style.SetScrollbarThumb(Gray)
	style.SetScrollbarRadius(3)
	style.SetScrollbarPadding(1)
	style.SetPadding(6)
	return style
}

func DefaultTinyGLStyle() Style {
	style := Style{}
	style.SetBackground(Black)
	style.SetBorderColor(Gray)
	style.SetBorderWidth(1)
	return style
}

func DefaultBoxStyle() Style {
	style := Style{}
	style.SetDisplay(DisplayBlock)
	return style
}

func (element *Element) isFocusable() bool {
	if element == nil {
		return false
	}
	switch element.kind {
	case ElementKindInput, ElementKindTextarea, ElementKindButton:
		return true
	}
	return element.OnClick != nil
}

func createElementTyped(kind ElementKind) *Element {
	element := newElement(kind, "")
	switch kind {
	case ElementKindButton:
		element.Style = DefaultButtonStyle()
	case ElementKindLabel:
		element.Style = DefaultLabelStyle()
	case ElementKindInput:
		element.Style = DefaultInputStyle()
	case ElementKindTextarea:
		element.Style = DefaultTextareaStyle()
	case ElementKindTinyGL:
		element.Style = DefaultTinyGLStyle()
	case ElementKindBox:
		element.Style = DefaultBoxStyle()
	}
	return element
}

func CreateButton() *Element {
	return createElementTyped(ElementKindButton)
}

func CreateLabel() *Element {
	return createElementTyped(ElementKindLabel)
}

func CreateInput() *Element {
	return createElementTyped(ElementKindInput)
}

func CreateTextarea() *Element {
	return createElementTyped(ElementKindTextarea)
}

func CreateTinyGL() *Element {
	return createElementTyped(ElementKindTinyGL)
}

func CreateBox() *Element {
	return createElementTyped(ElementKindBox)
}
