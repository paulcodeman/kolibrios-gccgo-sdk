package ui

import "kos"

func (window *Window) backgroundOverride() bool {
	if window == nil {
		return false
	}
	if window.Style.Background != nil {
		return true
	}
	return window.Style.Gradient != nil && !FastNoGradients
}

func (window *Window) backgroundStyle() Style {
	if window == nil {
		return Style{}
	}
	style := window.Style
	if style.BackgroundAttachment == nil {
		// Match HTML body behavior: background is fixed to the viewport by default.
		style.BackgroundAttachment = BackgroundAttachmentPtr(BackgroundAttachmentFixed)
	}
	if FastNoGradients {
		style.Gradient = nil
	}
	if FastNoShadows {
		style.Shadow = nil
	}
	if FastNoBorders {
		style.BorderWidth = nil
		style.BorderColor = nil
	}
	if FastNoRadius {
		style.BorderRadius = nil
	}
	if style.Background == nil && style.Gradient == nil {
		style.Background = ColorPtr(window.Background)
	}
	return style
}

func (window *Window) backgroundAttachment(style Style) BackgroundAttachment {
	if value, ok := resolveBackgroundAttachment(style.BackgroundAttachment); ok {
		return value
	}
	return BackgroundAttachmentFixed
}

func (window *Window) backgroundRectFor(style Style, rect Rect) Rect {
	attachment := window.backgroundAttachment(style)
	switch attachment {
	case BackgroundAttachmentFixed:
		return rect
	case BackgroundAttachmentLocal, BackgroundAttachmentScroll:
		if window.scrollEnabled() && window.scrollMaxY > 0 {
			height := rect.Height + window.scrollMaxY
			return Rect{X: rect.X, Y: rect.Y - window.scrollY, Width: rect.Width, Height: height}
		}
		return rect
	default:
		return rect
	}
}

func (window *Window) simpleBackgroundColor() (kos.Color, bool) {
	if window == nil {
		return 0, false
	}
	style := window.backgroundStyle()
	if style.Gradient != nil {
		return 0, false
	}
	if style.Shadow != nil {
		return 0, false
	}
	if borderWidthFor(style) > 0 {
		return 0, false
	}
	if resolveBorderRadius(style).Active() {
		return 0, false
	}
	if value, ok := resolveOpacity(style.Opacity); ok && value < 255 {
		return 0, false
	}
	background := window.Background
	if value, ok := resolveColor(style.Background); ok {
		background = value
	}
	_, alpha := colorValueAndAlpha(background)
	if alpha < 255 {
		return 0, false
	}
	return background, true
}

func (window *Window) backgroundNeedsTransparentClearStyle(style Style) bool {
	if resolveBorderRadius(style).Active() {
		return true
	}
	if value, ok := resolveOpacity(style.Opacity); ok && value < 255 {
		return true
	}
	if value, ok := resolveGradient(style.Gradient); ok && value != nil {
		_, fromAlpha := colorValueAndAlpha(value.From)
		_, toAlpha := colorValueAndAlpha(value.To)
		return fromAlpha < 255 || toAlpha < 255
	}
	background := window.Background
	if value, ok := resolveColor(style.Background); ok {
		background = value
	}
	_, alpha := colorValueAndAlpha(background)
	return alpha < 255
}

func (window *Window) backgroundNeedsTransparentClear() bool {
	if window == nil {
		return false
	}
	style := window.backgroundStyle()
	return window.backgroundNeedsTransparentClearStyle(style)
}

func (window *Window) drawBackgroundRectWith(canvas *Canvas, rect Rect, style Style, bgRect Rect) {
	if window == nil || canvas == nil || rect.Empty() {
		return
	}
	background := window.Background
	drawStyledBox(canvas, rect, style, bgRect, &background)
}

func (window *Window) drawBackgroundRect(rect Rect) {
	if window == nil || window.canvas == nil || rect.Empty() {
		return
	}
	style := window.backgroundStyle()
	bgRect := window.backgroundRectFor(style, rect)
	window.drawBackgroundRectWith(window.canvas, rect, style, bgRect)
}

func (window *Window) ensureBackgroundCache() *Canvas {
	if window == nil {
		return nil
	}
	if window.client.Empty() {
		window.backgroundCache = nil
		return nil
	}
	if _, ok := window.simpleBackgroundColor(); ok {
		window.backgroundCache = nil
		return nil
	}
	style := window.backgroundStyle()
	key := visualKeyFor(style)
	rect := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	bgRect := window.backgroundRectFor(style, rect)
	cache := window.backgroundCache
	if cache == nil || cache.Width() != window.client.Width || cache.Height() != window.client.Height ||
		!styleVisualKeyEqual(window.backgroundCacheKey, key) || window.backgroundCacheRect != bgRect {
		cache = NewCanvas(window.client.Width, window.client.Height)
		window.backgroundCache = cache
		window.backgroundCacheKey = key
		window.backgroundCacheRect = bgRect
		if window.backgroundNeedsTransparentClearStyle(style) {
			cache.ClearTransparent()
		}
		window.drawBackgroundRectWith(cache, rect, style, bgRect)
	}
	return cache
}

func (window *Window) drawBackgroundFull() {
	if window == nil || window.canvas == nil {
		return
	}
	rect := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	if rect.Empty() {
		return
	}
	if color, ok := window.simpleBackgroundColor(); ok {
		window.canvas.Clear(color)
		return
	}
	if cache := window.ensureBackgroundCache(); cache != nil {
		window.canvas.BlitFrom(cache, rect, rect.X, rect.Y)
	}
}
