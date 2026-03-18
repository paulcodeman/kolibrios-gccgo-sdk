package ui

import "kos"

func copySpacingPtr(value *Spacing) *Spacing {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyCornerRadiiPtr(value *CornerRadii) *CornerRadii {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyGradientPtr(value *Gradient) *Gradient {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyShadowPtr(value *Shadow) *Shadow {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyTextShadowPtr(value *TextShadow) *TextShadow {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (style Style) GetBackground() (kos.Color, bool) {
	return resolveColor(style.background)
}

func (style Style) GetForeground() (kos.Color, bool) {
	return resolveColor(style.foreground)
}

func (style Style) GetBorderColor() (kos.Color, bool) {
	return resolveColor(style.borderColor)
}

func (style Style) GetBorderWidth() (int, bool) {
	return resolveLength(style.borderWidth)
}

func (style Style) GetBorderRadius() (CornerRadii, bool) {
	if value, ok := resolveCornerRadii(style.borderRadius); ok && value != nil {
		return *value, true
	}
	return CornerRadii{}, false
}

func (style Style) GetGradient() (Gradient, bool) {
	if value, ok := resolveGradient(style.gradient); ok && value != nil {
		return *value, true
	}
	return Gradient{}, false
}

func (style Style) GetBackgroundAttachment() (BackgroundAttachment, bool) {
	return resolveBackgroundAttachment(style.backgroundAttachment)
}

func (style Style) GetShadow() (Shadow, bool) {
	if value, ok := resolveShadow(style.shadow); ok && value != nil {
		return *value, true
	}
	return Shadow{}, false
}

func (style Style) GetDisplay() (DisplayMode, bool) {
	return resolveDisplay(style.display)
}

func (style Style) GetTextAlign() (TextAlign, bool) {
	return resolveTextAlign(style.textAlign)
}

func (style Style) GetTextShadow() (TextShadow, bool) {
	if value, ok := resolveTextShadow(style.textShadow); ok && value != nil {
		return *value, true
	}
	return TextShadow{}, false
}

func (style Style) GetFontPath() (string, bool) {
	return resolveFontPath(style.fontPath)
}

func (style Style) GetFontSize() (int, bool) {
	return resolveFontSize(style.fontSize)
}

func (style Style) GetPadding() (Spacing, bool) {
	if value, ok := resolveSpacing(style.padding); ok && value != nil {
		return *value, true
	}
	return Spacing{}, false
}

func (style Style) GetOpacity() (uint8, bool) {
	return resolveOpacity(style.opacity)
}

func (style Style) GetPosition() (PositionMode, bool) {
	return resolvePosition(style.position)
}

func (style Style) GetLeft() (int, bool) {
	return resolveLength(style.left)
}

func (style Style) GetTop() (int, bool) {
	return resolveLength(style.top)
}

func (style Style) GetRight() (int, bool) {
	return resolveLength(style.right)
}

func (style Style) GetBottom() (int, bool) {
	return resolveLength(style.bottom)
}

func (style Style) GetWidth() (int, bool) {
	return resolveLength(style.width)
}

func (style Style) GetHeight() (int, bool) {
	return resolveLength(style.height)
}

func (style Style) GetMargin() (Spacing, bool) {
	if value, ok := resolveSpacing(style.margin); ok && value != nil {
		return *value, true
	}
	return Spacing{}, false
}

func (style Style) GetOverflow() (OverflowMode, bool) {
	return resolveOverflow(style.overflow)
}

func (style Style) GetOverflowX() (OverflowMode, bool) {
	return resolveOverflow(style.overflowX)
}

func (style Style) GetOverflowY() (OverflowMode, bool) {
	return resolveOverflow(style.overflowY)
}

func (style Style) GetScrollbarWidth() (int, bool) {
	return resolveScrollbarWidth(style.scrollbarWidth)
}

func (style Style) GetScrollbarTrack() (kos.Color, bool) {
	return resolveColor(style.scrollbarTrack)
}

func (style Style) GetScrollbarThumb() (kos.Color, bool) {
	return resolveColor(style.scrollbarThumb)
}

func (style Style) GetScrollbarRadius() (int, bool) {
	return resolveScrollbarRadius(style.scrollbarRadius)
}

func (style Style) GetScrollbarPadding() (Spacing, bool) {
	if value, ok := resolveSpacing(style.scrollbarPadding); ok && value != nil {
		return *value, true
	}
	return Spacing{}, false
}
