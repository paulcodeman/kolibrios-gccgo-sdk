package ui

import "kos"

func (style *Style) Absolute(x int, y int) {
	if style == nil {
		return
	}
	style.Position = PositionPtr(PositionAbsolute)
	style.Left = IntPtr(x)
	style.Top = IntPtr(y)
}

func (style *Style) Size(width int, height int) {
	if style == nil {
		return
	}
	if width >= 0 {
		style.Width = IntPtr(width)
	}
	if height >= 0 {
		style.Height = IntPtr(height)
	}
}

func (style *Style) SetWidth(value int) bool {
	if style == nil {
		return false
	}
	style.Width = IntPtr(value)
	return true
}

func (style *Style) SetHeight(value int) bool {
	if style == nil {
		return false
	}
	style.Height = IntPtr(value)
	return true
}

func (style *Style) SetLeft(value int) bool {
	if style == nil {
		return false
	}
	style.Left = IntPtr(value)
	return true
}

func (style *Style) SetTop(value int) bool {
	if style == nil {
		return false
	}
	style.Top = IntPtr(value)
	return true
}

func (style *Style) SetRight(value int) bool {
	if style == nil {
		return false
	}
	style.Right = IntPtr(value)
	return true
}

func (style *Style) SetBottom(value int) bool {
	if style == nil {
		return false
	}
	style.Bottom = IntPtr(value)
	return true
}

func (style *Style) SetBackground(color kos.Color) {
	if style == nil {
		return
	}
	style.Background = ColorPtr(color)
}

func (style *Style) SetBackgroundAttachment(value BackgroundAttachment) {
	if style == nil {
		return
	}
	style.BackgroundAttachment = BackgroundAttachmentPtr(value)
}

func (style *Style) SetForeground(color kos.Color) {
	if style == nil {
		return
	}
	style.Foreground = ColorPtr(color)
}

func (style *Style) SetBorderColor(color kos.Color) {
	if style == nil {
		return
	}
	style.BorderColor = ColorPtr(color)
}

func (style *Style) SetBorderWidth(width int) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.BorderWidth = IntPtr(width)
}

func (style *Style) SetMargin(values ...int) {
	if style == nil {
		return
	}
	style.Margin = SpacingCSS(values...)
}

func (style *Style) SetPadding(values ...int) {
	if style == nil {
		return
	}
	style.Padding = SpacingCSS(values...)
}

func (style *Style) SetBorder(width int, color kos.Color) {
	if style == nil {
		return
	}
	if width < 0 {
		width = 0
	}
	style.BorderWidth = IntPtr(width)
	style.BorderColor = ColorPtr(color)
}

func (style *Style) SetBorderRadius(values ...int) {
	if style == nil {
		return
	}
	style.BorderRadius = BorderRadiusCSS(values...)
}

func (style *Style) SetDisplay(value DisplayMode) {
	if style == nil {
		return
	}
	style.Display = DisplayPtr(value)
}

func (style *Style) SetPosition(value PositionMode) {
	if style == nil {
		return
	}
	style.Position = PositionPtr(value)
}

func (style *Style) SetTextAlign(value TextAlign) {
	if style == nil {
		return
	}
	style.TextAlign = AlignPtr(value)
}

func (style *Style) SetTextShadow(value TextShadow) {
	if style == nil {
		return
	}
	v := value
	style.TextShadow = &v
}

func (style *Style) SetTextShadowPtr(value *TextShadow) {
	if style == nil {
		return
	}
	style.TextShadow = value
}

func (style *Style) SetFontPath(value string) {
	if style == nil {
		return
	}
	if value == "" {
		style.FontPath = nil
		return
	}
	style.FontPath = StringPtr(value)
}

func (style *Style) SetFontSize(value int) {
	if style == nil {
		return
	}
	if value <= 0 {
		style.FontSize = nil
		return
	}
	style.FontSize = IntPtr(value)
}

func (style *Style) SetGradient(value Gradient) {
	if style == nil {
		return
	}
	v := value
	style.Gradient = &v
}

func (style *Style) SetGradientPtr(value *Gradient) {
	if style == nil {
		return
	}
	style.Gradient = value
}

func (style *Style) SetShadow(value Shadow) {
	if style == nil {
		return
	}
	v := value
	style.Shadow = &v
}

func (style *Style) SetShadowPtr(value *Shadow) {
	if style == nil {
		return
	}
	style.Shadow = value
}

func (style *Style) SetOpacity(value uint8) {
	if style == nil {
		return
	}
	style.Opacity = BytePtr(value)
}

func (style *Style) SetOverflow(value OverflowMode) {
	if style == nil {
		return
	}
	style.Overflow = OverflowPtr(value)
}

func (style *Style) SetOverflowX(value OverflowMode) {
	if style == nil {
		return
	}
	style.OverflowX = OverflowPtr(value)
}

func (style *Style) SetOverflowY(value OverflowMode) {
	if style == nil {
		return
	}
	style.OverflowY = OverflowPtr(value)
}

func (style *Style) SetScrollbarWidth(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.ScrollbarWidth = IntPtr(value)
}

func (style *Style) SetScrollbarTrack(color kos.Color) {
	if style == nil {
		return
	}
	style.ScrollbarTrack = ColorPtr(color)
}

func (style *Style) SetScrollbarThumb(color kos.Color) {
	if style == nil {
		return
	}
	style.ScrollbarThumb = ColorPtr(color)
}

func (style *Style) SetScrollbarRadius(value int) {
	if style == nil {
		return
	}
	if value < 0 {
		value = 0
	}
	style.ScrollbarRadius = IntPtr(value)
}

func (style *Style) SetScrollbarPadding(values ...int) {
	if style == nil {
		return
	}
	style.ScrollbarPadding = SpacingCSS(values...)
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
