package ui

import "kos"

// Draw draws the element directly to the window using raw syscalls.
// Prefer DrawTo with a Canvas for styled rendering.
func (element *Element) Draw() {
	if element == nil {
		return
	}
	if display, ok := resolveDisplay(element.effectiveStyle().Display); ok && display == DisplayNone {
		return
	}
	if element.isTextInput() {
		style := element.effectiveStyle()
		x, y := element.rawPosition(style)
		rect := Rect{X: x, Y: y, Width: element.resolvedWidth(style), Height: element.resolvedHeight(style)}
		foreground, ok := resolveColor(style.Foreground)
		if !ok {
			foreground = Black
		}
		layout := element.textInputLayout(rect, style)
		defer layout.release()
		selectionColor := defaultSelectionBackground
		element.drawEditableTextLines(layout, style,
			func(tx int, ty int, line string) {
				kos.DrawText(tx, ty, foreground, line)
			},
			func(cx int, cy int) {
				kos.DrawBar(cx, cy, 1, defaultFontHeight, uint32(foreground))
			},
			func(x int, y int, width int, height int) {
				if width <= 0 || height <= 0 {
					return
				}
				kos.DrawBar(x, y, width, height, uint32(selectionColor))
			},
		)
		return
	}
	switch element.kind {
	case ElementKindButton:
		style := element.effectiveStyle()
		x, y := element.rawPosition(style)
		background, ok := resolveColor(style.Background)
		if !ok {
			background = Silver
		}
		foreground, ok := resolveColor(style.Foreground)
		if !ok {
			foreground = Black
		}
		width := element.resolvedWidth(style)
		height := element.resolvedHeight(style)
		kos.DrawButton(x, y, width, height, element.ID, background)
		element.forEachTextLine(Rect{X: x, Y: y, Width: width, Height: height}, style, func(textX, textY int, line string) {
			kos.DrawText(
				textX,
				textY,
				foreground,
				line,
			)
		})
	case ElementKindLabel:
		style := element.effectiveStyle()
		x, y := element.rawPosition(style)
		foreground, ok := resolveColor(style.Foreground)
		if !ok {
			foreground = Black
		}
		element.forEachTextLine(Rect{X: x, Y: y, Width: element.resolvedWidth(style), Height: element.resolvedHeight(style)}, style, func(textX, textY int, line string) {
			kos.DrawText(
				textX,
				textY,
				foreground,
				line,
			)
		})
	default:
		style := element.effectiveStyle()
		x, y := element.rawPosition(style)
		foreground, ok := resolveColor(style.Foreground)
		if !ok {
			foreground = Black
		}
		element.forEachTextLine(Rect{X: x, Y: y, Width: element.resolvedWidth(style), Height: element.resolvedHeight(style)}, style, func(textX, textY int, line string) {
			kos.DrawText(
				textX,
				textY,
				foreground,
				line,
			)
		})
	}
}

func (element *Element) DrawTo(canvas *Canvas) {
	if element == nil || canvas == nil {
		return
	}
	style := element.effectiveStyle()
	if display, ok := resolveDisplay(style.Display); ok && display == DisplayNone {
		return
	}
	element.updateRenderKey(style)
	if element.layoutRect.Empty() {
		element.applyLayout(canvas, style)
	}
	rect := element.layoutRect
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	if element.tryDrawFromCache(canvas, rect, style) {
		return
	}
	element.drawToRect(canvas, rect, style)
}

func (element *Element) backgroundRect(rect Rect, style Style) Rect {
	if element == nil {
		return rect
	}
	attachment := BackgroundAttachmentScroll
	if value, ok := resolveBackgroundAttachment(style.BackgroundAttachment); ok {
		attachment = value
	}
	if attachment == BackgroundAttachmentFixed {
		if element.window != nil {
			return Rect{X: 0, Y: 0, Width: element.window.client.Width, Height: element.window.client.Height}
		}
	}
	return rect
}

func (element *Element) drawToRect(canvas *Canvas, rect Rect, style Style) {
	if element == nil || canvas == nil || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	borderRadius := resolveBorderRadius(style)
	if FastNoRadius {
		borderRadius = CornerRadii{}
	}
	if !FastNoShadows {
		if shadow, ok := resolveShadow(style.Shadow); ok {
			if borderRadius.Active() {
				canvas.DrawShadowRounded(rect, *shadow, borderRadius)
			} else {
				canvas.DrawShadow(rect, *shadow)
			}
		}
	}

	gradient, gradientSet := resolveGradient(style.Gradient)
	if FastNoGradients {
		gradientSet = false
	}
	background, backgroundSet := resolveColor(style.Background)
	bgRect := rect
	if gradientSet {
		bgRect = element.backgroundRect(rect, style)
	}
	if gradientSet {
		if opacity, ok := resolveOpacity(style.Opacity); ok && opacity < 255 {
			canvas.FillRoundedRectGradientAreaAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, bgRect, opacity)
		} else {
			canvas.FillRoundedRectGradientArea(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, bgRect)
		}
	} else if backgroundSet {
		if opacity, ok := resolveOpacity(style.Opacity); ok && opacity < 255 {
			canvas.FillRoundedRectAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background, opacity)
		} else {
			canvas.FillRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background)
		}
	}

	if !FastNoBorders {
		if borderWidth, ok := resolveLength(style.BorderWidth); ok && borderWidth > 0 {
			borderColor := kos.Color(0)
			if value, ok := resolveColor(style.BorderColor); ok {
				borderColor = value
			}
			canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, borderWidth, borderColor)
		}
	}

	if element.isTextInput() {
		if FastNoText {
			return
		}
		foreground, ok := resolveColor(style.Foreground)
		if !ok {
			foreground = Black
		}
		font := fontForStyle(style)
		layout := element.textInputLayout(rect, style)
		defer layout.release()
		selectionColor := defaultSelectionBackground
		canvas.PushClip(layout.content)
		element.drawEditableTextLines(layout, style,
			func(x int, y int, text string) {
				if font != nil {
					canvas.DrawTextFont(x, y, foreground, text, font)
				} else {
					canvas.DrawText(x, y, foreground, text)
				}
			},
			func(x int, y int) {
				height := layout.lineHeight
				if height <= 0 {
					height = defaultFontHeight
				}
				canvas.FillRect(x, y, 1, height, foreground)
			},
			func(x int, y int, width int, height int) {
				if width <= 0 || height <= 0 {
					return
				}
				canvas.FillRect(x, y, width, height, selectionColor)
			},
		)
		canvas.PopClip()
		element.drawInputScrollbars(canvas, layout)
		return
	}
	if element.kind == ElementKindTinyGL {
		return
	}
	if FastNoText {
		return
	}
	text := element.text()
	if text == "" {
		return
	}
	foreground, ok := resolveColor(style.Foreground)
	if !ok {
		foreground = Black
	}
	font := fontForStyle(style)
	shadow, shadowOk := resolveTextShadow(style.TextShadow)
	if FastNoTextShadow || FastNoShadows {
		shadowOk = false
	}
	element.forEachTextLine(rect, style, func(textX, textY int, line string) {
		if shadowOk {
			if font != nil {
				canvas.DrawTextFont(textX+shadow.OffsetX, textY+shadow.OffsetY, shadow.Color, line, font)
			} else {
				canvas.DrawText(textX+shadow.OffsetX, textY+shadow.OffsetY, shadow.Color, line)
			}
		}
		if font != nil {
			canvas.DrawTextFont(textX, textY, foreground, line, font)
		} else {
			canvas.DrawText(textX, textY, foreground, line)
		}
	})
}

func (element *Element) tryDrawFromCache(canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	if FastNoCache {
		element.cache = nil
		return false
	}
	cacheable, needsAlpha, visual := element.cacheInfo(style, rect)
	if !cacheable || visual.Empty() {
		element.cache = nil
		return false
	}
	cache := element.cache
	if cache == nil {
		cache = &elementCache{}
		element.cache = cache
	}
	local := Rect{X: rect.X - visual.X, Y: rect.Y - visual.Y, Width: rect.Width, Height: rect.Height}
	if cache.canvas == nil || cache.width != visual.Width || cache.height != visual.Height ||
		cache.alpha != needsAlpha || !elementRenderKeyEqual(cache.renderKey, element.renderKey) ||
		cache.offsetX != local.X || cache.offsetY != local.Y {
		cache.width = visual.Width
		cache.height = visual.Height
		cache.offsetX = local.X
		cache.offsetY = local.Y
		cache.alpha = needsAlpha
		if cache.canvas == nil || cache.canvas.alpha != needsAlpha {
			if needsAlpha {
				cache.canvas = NewCanvasAlpha(visual.Width, visual.Height)
			} else {
				cache.canvas = NewCanvas(visual.Width, visual.Height)
			}
		} else if cache.canvas.Width() != visual.Width || cache.canvas.Height() != visual.Height {
			cache.canvas.Resize(visual.Width, visual.Height)
		}
		if needsAlpha {
			cache.canvas.ClearTransparent()
		} else {
			cache.canvas.Clear(0)
		}
		element.drawToRect(cache.canvas, local, style)
		cache.renderKey = element.renderKey
	}
	canvas.BlitFrom(cache.canvas, Rect{X: 0, Y: 0, Width: cache.width, Height: cache.height}, visual.X, visual.Y)
	return true
}

func (element *Element) cacheInfo(style Style, rect Rect) (bool, bool, Rect) {
	if element == nil {
		return false, false, Rect{}
	}
	if element.isTextInput() {
		return false, false, Rect{}
	}
	if attachment, ok := resolveBackgroundAttachment(style.BackgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
		return false, false, Rect{}
	}
	visual := element.visualBoundsFor(rect, style)
	if visual.Empty() {
		return false, false, Rect{}
	}

	text := element.text()
	hasText := text != ""
	_, backgroundSet := resolveColor(style.Background)
	_, gradientSet := resolveGradient(style.Gradient)
	borderWidth := 0
	if value, ok := resolveLength(style.BorderWidth); ok {
		borderWidth = value
	}
	shadowSet := false
	if shadow, ok := resolveShadow(style.Shadow); ok && shadow != nil {
		shadowSet = true
	}
	textShadowSet := false
	if element.kind == ElementKindLabel {
		if shadow, ok := resolveTextShadow(style.TextShadow); ok && shadow != nil {
			textShadowSet = true
		}
	}
	hasVisual := backgroundSet || gradientSet || borderWidth > 0 || hasText || shadowSet || textShadowSet
	if !hasVisual {
		return false, false, Rect{}
	}
	needsAlpha := false
	if shadowSet || textShadowSet {
		needsAlpha = true
	}
	if opacity, ok := resolveOpacity(style.Opacity); ok && opacity < 255 {
		needsAlpha = true
	}
	if radii := resolveBorderRadius(style); radii.Active() {
		needsAlpha = true
	}
	if visual != rect {
		needsAlpha = true
	}
	if hasText && !backgroundSet && !gradientSet {
		needsAlpha = true
	}
	if borderWidth > 0 && !backgroundSet && !gradientSet {
		needsAlpha = true
	}
	return true, needsAlpha, visual
}

func (element *Element) Bounds() Rect {
	if element.layoutRect.Width > 0 || element.layoutRect.Height > 0 {
		return element.layoutRect
	}
	style := element.effectiveStyle()
	x, y := element.rawPosition(style)
	return Rect{
		X:      x,
		Y:      y,
		Width:  element.resolvedWidth(style),
		Height: element.resolvedHeight(style),
	}
}

func (element *Element) VisualBounds() Rect {
	if element.visualRect.Width > 0 || element.visualRect.Height > 0 {
		return element.visualRect
	}
	return element.Bounds()
}

func (element *Element) subtreeBounds() Rect {
	if element == nil {
		return Rect{}
	}
	if element.subtreeRect.Width > 0 || element.subtreeRect.Height > 0 {
		return element.subtreeRect
	}
	if element.visualRect.Width > 0 || element.visualRect.Height > 0 {
		return element.visualRect
	}
	return element.Bounds()
}

func (element *Element) updateSubtreeRect() {
	if element == nil {
		return
	}
	rect := element.visualRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	for _, child := range element.Children {
		if child == nil {
			continue
		}
		if nodeHidden(child) {
			continue
		}
		if childEl, ok := child.(*Element); ok {
			rect = UnionRect(rect, childEl.subtreeBounds())
			continue
		}
		rect = UnionRect(rect, child.Bounds())
	}
	element.subtreeRect = rect
}

func (element *Element) visualBoundsFor(rect Rect, style Style) Rect {
	if rect.Empty() {
		return rect
	}
	visual := rect
	if shadow, ok := resolveShadow(style.Shadow); ok {
		blur := shadow.Blur
		if blur < 0 {
			blur = 0
		}
		left := visual.X
		top := visual.Y
		right := visual.X + visual.Width
		bottom := visual.Y + visual.Height
		shadowLeft := rect.X + shadow.OffsetX - blur
		shadowTop := rect.Y + shadow.OffsetY - blur
		shadowRight := rect.X + shadow.OffsetX + rect.Width + blur
		shadowBottom := rect.Y + shadow.OffsetY + rect.Height + blur
		if shadowLeft < left {
			left = shadowLeft
		}
		if shadowTop < top {
			top = shadowTop
		}
		if shadowRight > right {
			right = shadowRight
		}
		if shadowBottom > bottom {
			bottom = shadowBottom
		}
		visual = Rect{X: left, Y: top, Width: right - left, Height: bottom - top}
	}
	if element.kind == ElementKindLabel {
		if shadow, ok := resolveTextShadow(style.TextShadow); ok {
			shadowRect := Rect{
				X:      rect.X + shadow.OffsetX,
				Y:      rect.Y + shadow.OffsetY,
				Width:  rect.Width,
				Height: rect.Height,
			}
			visual = UnionRect(visual, shadowRect)
		}
	}
	return visual
}
