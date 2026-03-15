package ui

func DefaultButtonStyle() Style {
	style := Style{
		Background:  ColorPtr(Silver),
		Foreground:  ColorPtr(Black),
		BorderColor: ColorPtr(Gray),
		BorderWidth: IntPtr(1),
		TextAlign:   AlignPtr(TextAlignCenter),
		Padding: &Spacing{
			Left:   6,
			Top:    4,
			Right:  6,
			Bottom: 4,
		},
	}
	return style
}

func DefaultLabelStyle() Style {
	style := Style{
		Foreground: ColorPtr(Black),
		TextAlign:  AlignPtr(TextAlignLeft),
	}
	return style
}

func DefaultInputStyle() Style {
	style := Style{
		Background:  ColorPtr(White),
		Foreground:  ColorPtr(Black),
		BorderColor: ColorPtr(Gray),
		BorderWidth: IntPtr(1),
		TextAlign:   AlignPtr(TextAlignLeft),
		Padding: &Spacing{
			Left:   6,
			Top:    4,
			Right:  6,
			Bottom: 4,
		},
	}
	return style
}

func DefaultTextareaStyle() Style {
	style := Style{
		Background:      ColorPtr(White),
		Foreground:      ColorPtr(Black),
		BorderColor:     ColorPtr(Gray),
		BorderWidth:     IntPtr(1),
		TextAlign:       AlignPtr(TextAlignLeft),
		Overflow:        OverflowPtr(OverflowAuto),
		ScrollbarWidth:  IntPtr(6),
		ScrollbarTrack:  ColorPtr(Silver),
		ScrollbarThumb:  ColorPtr(Gray),
		ScrollbarRadius: IntPtr(3),
		ScrollbarPadding: &Spacing{
			Left:   1,
			Top:    1,
			Right:  1,
			Bottom: 1,
		},
		Padding: &Spacing{
			Left:   6,
			Top:    6,
			Right:  6,
			Bottom: 6,
		},
	}
	return style
}

func DefaultTinyGLStyle() Style {
	style := Style{
		Background:  ColorPtr(Black),
		BorderColor: ColorPtr(Gray),
		BorderWidth: IntPtr(1),
	}
	return style
}

func DefaultBoxStyle() Style {
	return Style{
		Display: DisplayPtr(DisplayBlock),
	}
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
