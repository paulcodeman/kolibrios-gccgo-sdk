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

func DefaultButtonHoverStyle() Style {
	style := Style{}
	style.SetBackground(White)
	style.SetBorderColor(Gray)
	style.SetBorderWidth(1)
	style.SetGradient(Gradient{
		From:      White,
		To:        Silver,
		Direction: GradientVertical,
	})
	style.SetShadow(Shadow{
		OffsetX: 1,
		OffsetY: 1,
		Blur:    2,
		Color:   Black,
		Alpha:   70,
	})
	return style
}

func DefaultButtonActiveStyle() Style {
	style := Style{}
	style.SetBackground(Gray)
	style.SetForeground(Black)
	style.SetBorderWidth(1)
	style.SetBorderColor(Gray)
	style.SetGradient(Gradient{
		From:      Gray,
		To:        Silver,
		Direction: GradientVertical,
	})
	style.SetShadow(Shadow{
		OffsetX: 0,
		OffsetY: 0,
		Blur:    1,
		Color:   Black,
		Alpha:   40,
	})
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

func DefaultCheckboxStyle() Style {
	style := Style{}
	style.SetDisplay(DisplayInlineBlock)
	style.SetForeground(Black)
	style.SetPadding(4, 6, 4, 4)
	style.SetTextAlign(TextAlignLeft)
	return style
}

func DefaultCheckboxHoverStyle() Style {
	style := Style{}
	style.SetBackground(Silver)
	style.SetBorderRadius(6)
	return style
}

func DefaultCheckboxActiveStyle() Style {
	style := Style{}
	style.SetBackground(White)
	style.SetBorderRadius(6)
	return style
}

func DefaultRadioStyle() Style {
	return DefaultCheckboxStyle()
}

func DefaultRadioHoverStyle() Style {
	return DefaultCheckboxHoverStyle()
}

func DefaultRadioActiveStyle() Style {
	return DefaultCheckboxActiveStyle()
}

func DefaultProgressStyle() Style {
	style := Style{}
	style.SetDisplay(DisplayBlock)
	style.SetWidth(180)
	style.SetHeight(18)
	style.SetPadding(2)
	style.SetBackground(White)
	style.SetBorderColor(Silver)
	style.SetBorderWidth(1)
	style.SetBorderRadius(999)
	style.SetForeground(Blue)
	return style
}

func DefaultRangeStyle() Style {
	style := Style{}
	style.SetDisplay(DisplayBlock)
	style.SetWidth(180)
	style.SetHeight(24)
	style.SetPadding(4, 6)
	style.SetForeground(Blue)
	return style
}

func DefaultRangeHoverStyle() Style {
	style := Style{}
	style.SetBackground(Silver)
	style.SetBorderRadius(8)
	return style
}

func DefaultRangeActiveStyle() Style {
	style := Style{}
	style.SetBackground(White)
	style.SetBorderRadius(8)
	return style
}

func (element *Element) Spec() *ElementSpec {
	if element == nil {
		return nil
	}
	if element.spec != nil {
		return element.spec
	}
	return ElementSpecForKind(element.kind)
}

func (element *Element) hasSpecFlag(flag ElementSpecFlags) bool {
	if element == nil {
		return false
	}
	if spec := element.Spec(); spec != nil {
		return spec.hasFlag(flag)
	}
	return false
}

func (element *Element) isFocusable() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecFocusable) {
		return true
	}
	return element.OnClick != nil
}

func (element *Element) hasEventHandlers() bool {
	if element == nil {
		return false
	}
	return element.OnEvent != nil ||
		element.OnEventCapture != nil ||
		element.OnClick != nil ||
		element.OnChange != nil ||
		element.OnInput != nil ||
		element.OnPointerDown != nil ||
		element.OnPointerUp != nil ||
		element.OnPointerMove != nil ||
		element.OnPointerEnter != nil ||
		element.OnPointerLeave != nil ||
		element.OnPointerCancel != nil ||
		element.OnMouseDown != nil ||
		element.OnMouseUp != nil ||
		element.OnMouseMove != nil ||
		element.OnMouseEnter != nil ||
		element.OnMouseLeave != nil ||
		element.OnScroll != nil ||
		element.OnFocus != nil ||
		element.OnBlur != nil ||
		element.OnFocusIn != nil ||
		element.OnFocusOut != nil ||
		element.OnKeyDown != nil
}

func (element *Element) isClickable() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecClickable) {
		return true
	}
	return element.OnClick != nil
}

func (element *Element) isButtonLike() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecButtonLike) {
		return true
	}
	return element.kind == ElementKindButton
}

func (element *Element) isContainerElement() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecContainer) {
		return true
	}
	return element.kind == ElementKindBox
}

func (element *Element) isCheckable() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecCheckable) {
		return true
	}
	return element.kind == ElementKindCheckbox || element.kind == ElementKindRadio
}

func (element *Element) isRadio() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecRadio) {
		return true
	}
	return element.kind == ElementKindRadio
}

func (element *Element) isProgress() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecProgress) {
		return true
	}
	return element.kind == ElementKindProgress
}

func (element *Element) isRange() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecRange) {
		return true
	}
	return element.kind == ElementKindRange
}

func (element *Element) isTinyGL() bool {
	if element == nil {
		return false
	}
	if element.hasSpecFlag(ElementSpecTinyGL) {
		return true
	}
	return element.kind == ElementKindTinyGL
}

func CreateElementFromSpec(spec *ElementSpec) *Element {
	if spec == nil {
		return newElement(ElementKindUnknown, "", nil)
	}
	element := newElement(spec.Kind, "", spec)
	element.Style = spec.defaultBaseStyle()
	element.StyleHover = spec.defaultHoverStyle()
	element.StyleActive = spec.defaultActiveStyle()
	element.StyleFocus = spec.defaultFocusStyle()
	element.specInit()
	return element
}

func CreateElementByName(name string) *Element {
	if spec := ElementSpecForName(name); spec != nil {
		return CreateElementFromSpec(spec)
	}
	return newElement(ElementKindUnknown, "", nil)
}

func createElementTyped(kind ElementKind) *Element {
	if spec := ElementSpecForKind(kind); spec != nil {
		return CreateElementFromSpec(spec)
	}
	return newElement(kind, "", nil)
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

func CreateCheckbox() *Element {
	return createElementTyped(ElementKindCheckbox)
}

func CreateRadio() *Element {
	return createElementTyped(ElementKindRadio)
}

func CreateProgress() *Element {
	return createElementTyped(ElementKindProgress)
}

func CreateRange() *Element {
	return createElementTyped(ElementKindRange)
}
