package ui

// Draw draws the element directly to the window.
// Most specs route through the same surface-backed paint path as DrawTo.
func (element *Element) Draw() {
	if element == nil {
		return
	}
	style := element.effectiveStyle()
	if display, ok := resolveDisplay(style.display); ok && display == DisplayNone {
		return
	}
	if !styleVisible(style) {
		return
	}
	if element.drawRawWithSpec(style) {
		return
	}
	drawElementViaSurfaceRaw(element, style)
}

func (element *Element) DrawTo(canvas *Canvas) {
	if element == nil || canvas == nil {
		return
	}
	style := element.effectiveStyle()
	if display, ok := resolveDisplay(style.display); ok && display == DisplayNone {
		return
	}
	if !styleVisible(style) {
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
	if element.tryDrawFromRetainedSubtreeLayer(canvas, style, 0) {
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
	if value, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok {
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
	bgRect := rect
	if style.gradient != nil && !FastNoGradients {
		bgRect = element.backgroundRect(rect, style)
	}
	drawStyledBox(canvas, rect, style, bgRect, nil)

	if element.paintWithSpec(canvas, rect, style) {
		return
	}
	if FastNoText {
		if elementShowsDefaultFocusRing(element) {
			drawDefaultFocusRing(canvas, rect, style)
		}
		return
	}
	element.drawTextToRect(canvas, rect, style)
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
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
	if attachment, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
		return false, false, Rect{}
	}
	visual := element.visualBoundsFor(rect, style)
	if visual.Empty() {
		return false, false, Rect{}
	}

	text := element.text()
	hasText := text != ""
	_, backgroundSet := resolveColor(style.background)
	_, gradientSet := resolveGradient(style.gradient)
	borderWidth := 0
	if value, ok := resolveLength(style.borderWidth); ok {
		borderWidth = value
	}
	shadowSet := false
	if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
		shadowSet = true
	}
	textShadowSet := false
	if hasText && !element.isTextInput() && !element.isTinyGL() {
		if shadow, ok := resolveTextShadow(style.textShadow); ok && shadow != nil {
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
	if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
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

func (element *Element) drawTextToRect(canvas *Canvas, rect Rect, style Style) bool {
	if element == nil || canvas == nil {
		return false
	}
	text := element.text()
	if text == "" {
		return false
	}
	foreground, ok := resolveColor(style.foreground)
	if !ok {
		foreground = Black
	}
	font, metrics := fontAndMetricsForStyle(style)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	shadow, shadowOk := resolveTextShadow(style.textShadow)
	if FastNoTextShadow || FastNoShadows {
		shadowOk = false
	}
	element.forEachTextLine(rect, style, func(textX int, textY int, line string) {
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
		drawTextDecorations(canvas, textX, textY, line, style, font, charWidth, foreground)
	})
	return true
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
	if element.visualRectValid {
		return element.visualRect
	}
	return element.Bounds()
}

func (element *Element) subtreeBounds() Rect {
	if element == nil {
		return Rect{}
	}
	if element.subtreeRectValid {
		return element.subtreeRect
	}
	if element.visualRectValid {
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
	if styleContainsPaint(element.effectiveStyle()) {
		element.subtreeRect = rect
		element.subtreeRectValid = true
		return
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
		if visual, ok := child.(VisualBoundsAware); ok {
			rect = UnionRect(rect, visual.VisualBounds())
			continue
		}
		rect = UnionRect(rect, child.Bounds())
	}
	element.subtreeRect = rect
	element.subtreeRectValid = true
}

func (element *Element) visualBoundsFor(rect Rect, style Style) Rect {
	includeTextShadow := !element.isTextInput() && !element.isTinyGL() && element.text() != ""
	visual := visualBoundsForStyle(rect, style, includeTextShadow)
	if elementShowsDefaultFocusRing(element) {
		visual = UnionRect(visual, focusRingBounds(rect))
	}
	return visual
}
