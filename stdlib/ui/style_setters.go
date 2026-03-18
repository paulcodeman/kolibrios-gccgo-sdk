package ui

import "kos"

func (style *Style) Absolute(x int, y int) {
	if style == nil {
		return
	}
	style.position = PositionPtr(PositionAbsolute)
	style.left = IntPtr(x)
	style.top = IntPtr(y)
}

func (style *Style) Size(width int, height int) {
	if style == nil {
		return
	}
	if width >= 0 {
		style.width = IntPtr(width)
	}
	if height >= 0 {
		style.height = IntPtr(height)
	}
}

func (style *Style) SetWidth(value int) bool {
	if style == nil {
		return false
	}
	style.width = IntPtr(value)
	return true
}

func (style *Style) SetHeight(value int) bool {
	if style == nil {
		return false
	}
	style.height = IntPtr(value)
	return true
}

func (style *Style) SetLeft(value int) bool {
	if style == nil {
		return false
	}
	style.left = IntPtr(value)
	return true
}

func (style *Style) SetTop(value int) bool {
	if style == nil {
		return false
	}
	style.top = IntPtr(value)
	return true
}

func (style *Style) SetRight(value int) bool {
	if style == nil {
		return false
	}
	style.right = IntPtr(value)
	return true
}

func (style *Style) SetBottom(value int) bool {
	if style == nil {
		return false
	}
	style.bottom = IntPtr(value)
	return true
}

func (style *Style) SetBackground(color kos.Color) {
	if style == nil {
		return
	}
	style.background = ColorPtr(color)
}

func (style *Style) SetBackgroundAttachment(value BackgroundAttachment) {
	if style == nil {
		return
	}
	style.backgroundAttachment = BackgroundAttachmentPtr(value)
}

func (style *Style) SetForeground(color kos.Color) {
	if style == nil {
		return
	}
	style.foreground = ColorPtr(color)
}

func (style *Style) SetBorderColor(color kos.Color) {
	if style == nil {
		return
	}
	style.borderColor = ColorPtr(color)
}

func (style *Style) SetBorderTopColor(color kos.Color) {
	if style == nil {
		return
	}
	style.borderTopColor = ColorPtr(color)
}

func (style *Style) SetBorderRightColor(color kos.Color) {
	if style == nil {
		return
	}
	style.borderRightColor = ColorPtr(color)
}

func (style *Style) SetBorderBottomColor(color kos.Color) {
	if style == nil {
		return
	}
	style.borderBottomColor = ColorPtr(color)
}

func (style *Style) SetBorderLeftColor(color kos.Color) {
	if style == nil {
		return
	}
	style.borderLeftColor = ColorPtr(color)
}

func (style *Style) SetBorderWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderWidth = IntPtr(width)
}

func (style *Style) SetBorderTopWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderTopWidth = IntPtr(width)
}

func (style *Style) SetBorderRightWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderRightWidth = IntPtr(width)
}

func (style *Style) SetBorderBottomWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderBottomWidth = IntPtr(width)
}

func (style *Style) SetBorderLeftWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderLeftWidth = IntPtr(width)
}

func (style *Style) SetMargin(values ...int) {
	if style == nil {
		return
	}
	style.margin = SpacingCSS(values...)
}

func (style *Style) SetPadding(values ...int) {
	if style == nil {
		return
	}
	style.padding = SpacingCSS(values...)
}

func (style *Style) SetBorder(width int, color kos.Color) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.borderWidth = IntPtr(width)
	style.borderColor = ColorPtr(color)
}

func (style *Style) SetBorderRadius(values ...int) {
	if style == nil {
		return
	}
	style.borderRadius = BorderRadiusCSS(values...)
}

func (style *Style) SetDisplay(value DisplayMode) {
	if style == nil {
		return
	}
	style.display = DisplayPtr(value)
}

func (style *Style) SetVisibility(value VisibilityMode) {
	if style == nil {
		return
	}
	style.visibility = VisibilityPtr(value)
}

func (style *Style) SetPosition(value PositionMode) {
	if style == nil {
		return
	}
	style.position = PositionPtr(value)
}

func (style *Style) SetTextAlign(value TextAlign) {
	if style == nil {
		return
	}
	style.textAlign = AlignPtr(value)
}

func (style *Style) SetTextDecoration(value TextDecoration) {
	if style == nil {
		return
	}
	style.textDecoration = TextDecorationPtr(value)
}

func (style *Style) SetWhiteSpace(value WhiteSpaceMode) {
	if style == nil {
		return
	}
	style.whiteSpace = WhiteSpacePtr(value)
}

func (style *Style) SetOverflowWrap(value OverflowWrapMode) {
	if style == nil {
		return
	}
	style.overflowWrap = OverflowWrapPtr(value)
}

func (style *Style) SetWordBreak(value WordBreakMode) {
	if style == nil {
		return
	}
	style.wordBreak = WordBreakPtr(value)
}

func (style *Style) SetTextShadow(value TextShadow) {
	if style == nil {
		return
	}
	v := value
	style.textShadow = &v
}

func (style *Style) SetTextShadowPtr(value *TextShadow) {
	if style == nil {
		return
	}
	style.textShadow = copyTextShadowPtr(value)
}

func (style *Style) SetFontPath(value string) {
	if style == nil {
		return
	}
	if value == "" {
		style.fontPath = nil
		return
	}
	style.fontPath = StringPtr(value)
}

func (style *Style) SetFontSize(value int) {
	if style == nil {
		return
	}
	if value <= 0 {
		style.fontSize = nil
		return
	}
	style.fontSize = IntPtr(value)
}

func (style *Style) SetLineHeight(value int) {
	if style == nil {
		return
	}
	if value <= 0 {
		style.lineHeight = nil
		return
	}
	style.lineHeight = IntPtr(value)
}

func (style *Style) SetGradient(value Gradient) {
	if style == nil {
		return
	}
	v := value
	style.gradient = &v
}

func (style *Style) SetGradientPtr(value *Gradient) {
	if style == nil {
		return
	}
	style.gradient = copyGradientPtr(value)
}

func (style *Style) SetShadow(value Shadow) {
	if style == nil {
		return
	}
	v := value
	style.shadow = &v
}

func (style *Style) SetShadowPtr(value *Shadow) {
	if style == nil {
		return
	}
	style.shadow = copyShadowPtr(value)
}

func (style *Style) SetOpacity(value uint8) {
	if style == nil {
		return
	}
	style.opacity = BytePtr(value)
}

func (style *Style) SetBoxSizing(value BoxSizing) {
	if style == nil {
		return
	}
	style.boxSizing = BoxSizingPtr(value)
}

func (style *Style) SetOutlineColor(color kos.Color) {
	if style == nil {
		return
	}
	style.outlineColor = ColorPtr(color)
}

func (style *Style) SetOutlineWidth(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.outlineWidth = IntPtr(value)
}

func (style *Style) SetOutlineOffset(value int) {
	if style == nil {
		return
	}
	style.outlineOffset = IntPtr(value)
}

func (style *Style) SetOutlineRadius(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.outlineRadius = IntPtr(value)
}

func (style *Style) SetOutline(width int, color kos.Color) {
	if style == nil {
		return
	}
	style.SetOutlineWidth(width)
	style.SetOutlineColor(color)
}

func (style *Style) SetOverflow(value OverflowMode) {
	if style == nil {
		return
	}
	style.overflow = OverflowPtr(value)
}

func (style *Style) SetOverflowX(value OverflowMode) {
	if style == nil {
		return
	}
	style.overflowX = OverflowPtr(value)
}

func (style *Style) SetOverflowY(value OverflowMode) {
	if style == nil {
		return
	}
	style.overflowY = OverflowPtr(value)
}

func (style *Style) SetScrollbarWidth(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.scrollbarWidth = IntPtr(value)
}

func (style *Style) SetScrollbarTrack(color kos.Color) {
	if style == nil {
		return
	}
	style.scrollbarTrack = ColorPtr(color)
}

func (style *Style) SetScrollbarThumb(color kos.Color) {
	if style == nil {
		return
	}
	style.scrollbarThumb = ColorPtr(color)
}

func (style *Style) SetScrollbarRadius(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.scrollbarRadius = IntPtr(value)
}

func (style *Style) SetScrollbarPadding(values ...int) {
	if style == nil {
		return
	}
	style.scrollbarPadding = SpacingCSS(values...)
}

func (style *Style) SetMinWidth(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.minWidth = IntPtr(value)
}

func (style *Style) SetMaxWidth(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		style.maxWidth = nil
		return
	}
	style.maxWidth = IntPtr(value)
}

func (style *Style) SetMinHeight(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.minHeight = IntPtr(value)
}

func (style *Style) SetMaxHeight(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		style.maxHeight = nil
		return
	}
	style.maxHeight = IntPtr(value)
}

func SpacingCSS(values ...int) *Spacing {
	if len(values) == 0 {
		return nil
	}
	top, right, bottom, left := expandBoxShorthand(values)
	return &Spacing{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
	}
}

func BorderRadiusCSS(values ...int) *CornerRadii {
	if len(values) == 0 {
		return nil
	}
	topLeft, topRight, bottomRight, bottomLeft := expandCornerShorthand(values)
	if topLeft < 0 {
		topLeft = 0
	}
	if topRight < 0 {
		topRight = 0
	}
	if bottomRight < 0 {
		bottomRight = 0
	}
	if bottomLeft < 0 {
		bottomLeft = 0
	}
	return &CornerRadii{
		TopLeft:     topLeft,
		TopRight:    topRight,
		BottomRight: bottomRight,
		BottomLeft:  bottomLeft,
	}
}
