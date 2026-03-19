package ui

func (element *Element) Layout(canvas *Canvas) {
	if element == nil || canvas == nil {
		return
	}
	element.LayoutWithContext(layoutContextForCanvas(canvas))
}

func (element *Element) LayoutWithContext(ctx LayoutContext) {
	if element == nil {
		return
	}
	style := element.effectiveStyle()
	element.updateRenderKey(style)
	element.applyLayoutWithContext(ctx, ctx.Viewport, style)
}

func (element *Element) LayoutDirty() bool {
	if element == nil {
		return false
	}
	style := element.effectiveStyle()
	key := element.layoutKeyFor(style, Rect{})
	return !elementLayoutKeyEqual(key, element.layoutKey)
}

func (element *Element) layoutDirtyInCurrentContainer() bool {
	if element == nil {
		return false
	}
	if element.layoutRect.Empty() {
		return true
	}
	container := element.layoutContainer()
	if container.Width == 0 && container.Height == 0 {
		return true
	}
	style := element.effectiveStyle()
	key := element.layoutKeyFor(style, container)
	return !elementLayoutKeyEqual(key, element.layoutKey)
}

func (element *Element) applyLayout(canvas *Canvas, style Style) {
	if element == nil || canvas == nil {
		return
	}
	element.applyLayoutWithContext(layoutContextForCanvas(canvas), Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}, style)
}

func (element *Element) applyLayoutIn(canvas *Canvas, container Rect, style Style) {
	if element == nil || canvas == nil {
		return
	}
	element.applyLayoutWithContext(layoutContextForCanvas(canvas), container, style)
}

func (element *Element) applyLayoutWithContext(ctx LayoutContext, container Rect, style Style) {
	if element == nil {
		return
	}
	key := element.layoutKeyForContext(ctx, style, container)
	if elementLayoutKeyEqual(key, element.layoutKey) && element.layoutRect.Width > 0 && element.layoutRect.Height > 0 {
		return
	}
	element.layoutKey = key
	rect := element.resolveRectIn(container, style)
	element.layoutRect = rect
	element.visualRect = element.visualBoundsFor(rect, style)
	element.visualRectValid = true
	element.subtreeRect = Rect{}
	element.subtreeRectValid = false
}

func (element *Element) layoutKeyFor(style Style, container Rect) elementLayoutKey {
	return element.layoutKeyForContext(DefaultLayoutContext(container), style, container)
}

func (element *Element) layoutKeyForContext(ctx LayoutContext, style Style, container Rect) elementLayoutKey {
	var position *PositionMode
	if value, ok := resolvePosition(style.position); ok {
		v := value
		position = &v
	}
	var display *DisplayMode
	if value, ok := resolveDisplay(style.display); ok {
		v := value
		display = &v
	}
	var left *int
	if value, ok := resolveLength(style.left); ok {
		v := value
		left = &v
	}
	var top *int
	if value, ok := resolveLength(style.top); ok {
		v := value
		top = &v
	}
	var right *int
	if value, ok := resolveLength(style.right); ok {
		v := value
		right = &v
	}
	var bottom *int
	if value, ok := resolveLength(style.bottom); ok {
		v := value
		bottom = &v
	}
	var styleWidth *int
	if value, ok := resolveLength(style.width); ok {
		v := value
		styleWidth = &v
	}
	var styleHeight *int
	if value, ok := resolveLength(style.height); ok {
		v := value
		styleHeight = &v
	}
	var margin *Spacing
	if value, ok := resolveSpacing(style.margin); ok {
		if value != nil {
			v := *value
			margin = &v
		}
	}
	flowSet := element.flowSet
	flowX := 0
	flowY := 0
	if flowSet {
		flowX = element.flowX
		flowY = element.flowY
	}
	return elementLayoutKey{
		kind:        element.kind,
		position:    position,
		display:     display,
		containerX:  container.X,
		containerY:  container.Y,
		containerW:  container.Width,
		containerH:  container.Height,
		left:        left,
		top:         top,
		right:       right,
		bottom:      bottom,
		width:       element.resolvedWidthInWithContext(ctx, style, container),
		height:      element.resolvedHeightInWithContext(ctx, style, container),
		styleWidth:  styleWidth,
		styleHeight: styleHeight,
		margin:      margin,
		flowSet:     flowSet,
		flowX:       flowX,
		flowY:       flowY,
	}
}

func (element *Element) layoutContainer() Rect {
	if element == nil {
		return Rect{}
	}
	key := element.layoutKey
	return Rect{
		X:      key.containerX,
		Y:      key.containerY,
		Width:  key.containerW,
		Height: key.containerH,
	}
}

func (element *Element) resolveRect(canvas *Canvas, style Style) Rect {
	if canvas == nil {
		return element.resolveRectIn(Rect{}, style)
	}
	container := Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}
	return element.resolveRectIn(container, style)
}

func (element *Element) resolveRectIn(container Rect, style Style) Rect {
	base := Rect{
		X:      0,
		Y:      0,
		Width:  element.resolvedWidthIn(style, container),
		Height: element.resolvedHeightIn(style, container),
	}
	x, y := element.basePosition(style)
	base.X = x
	base.Y = y
	if !style.HasLayout() {
		return base
	}
	return resolveRect(base, container, style)
}

func (element *Element) basePosition(style Style) (int, int) {
	position := effectivePosition(style)
	x := 0
	y := 0
	if position != PositionAbsolute && element.flowSet {
		x = element.flowX
		y = element.flowY
	}
	if position == PositionAbsolute {
		if value, ok := resolveLength(style.left); ok {
			x = value
		}
		if value, ok := resolveLength(style.top); ok {
			y = value
		}
	}
	return x, y
}

func (element *Element) rawPosition(style Style) (int, int) {
	x, y := element.basePosition(style)
	if effectivePosition(style) == PositionRelative {
		if value, ok := resolveLength(style.left); ok {
			x += value
		}
		if value, ok := resolveLength(style.right); ok {
			x -= value
		}
		if value, ok := resolveLength(style.top); ok {
			y += value
		}
		if value, ok := resolveLength(style.bottom); ok {
			y -= value
		}
	}
	return x, y
}

func (element *Element) setFlow(x int, y int) bool {
	if element == nil {
		return false
	}
	if element.flowSet && element.flowX == x && element.flowY == y {
		return false
	}
	element.flowSet = true
	element.flowX = x
	element.flowY = y
	return true
}

func (element *Element) clearFlow() bool {
	if element == nil || !element.flowSet {
		return false
	}
	element.flowSet = false
	element.flowX = 0
	element.flowY = 0
	return true
}

func (element *Element) resolvedWidth(style Style) int {
	return element.resolvedWidthWithContext(DefaultLayoutContext(Rect{}), style)
}

func (element *Element) resolvedWidthWithContext(ctx LayoutContext, style Style) int {
	if value, ok := explicitOuterWidth(style); ok {
		return clampWidthForStyle(style, value)
	}
	text := element.text()
	font, metrics := ctx.FontForStyle(style)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	textWidth := ctx.MeasureText(text, font, charWidth)
	insets := boxInsets(style)
	baseWidth := textWidth + insets.Left + insets.Right
	switch element.kind {
	case ElementKindButton:
		minWidth := textWidth + defaultButtonWidthPadding + insets.Left + insets.Right
		if baseWidth < minWidth {
			baseWidth = minWidth
		}
	}
	return clampWidthForStyle(style, baseWidth)
}

func (element *Element) resolvedWidthIn(style Style, container Rect) int {
	return element.resolvedWidthInWithContext(DefaultLayoutContext(container), style, container)
}

func (element *Element) resolvedWidthInWithContext(ctx LayoutContext, style Style, container Rect) int {
	if value, ok := explicitOuterWidth(style); ok {
		return clampWidthForStyle(style, value)
	}
	if display, ok := resolveDisplay(style.display); ok && display == DisplayBlock {
		if effectivePosition(style) != PositionAbsolute {
			width := container.Width
			if margin, ok := resolveSpacing(style.margin); ok && margin != nil {
				width -= margin.Left + margin.Right
			}
			if width < 0 {
				width = 0
			}
			return clampWidthForStyle(style, width)
		}
	}
	return clampWidthForStyle(style, element.resolvedWidthWithContext(ctx, style))
}

func (element *Element) resolvedHeight(style Style) int {
	return element.resolvedHeightWithContext(DefaultLayoutContext(Rect{}), style)
}

func (element *Element) resolvedHeightWithContext(ctx LayoutContext, style Style) int {
	return element.resolvedHeightInWithContext(ctx, style, Rect{})
}

func (element *Element) resolvedHeightIn(style Style, container Rect) int {
	return element.resolvedHeightInWithContext(DefaultLayoutContext(container), style, container)
}

func (element *Element) resolvedHeightInWithContext(ctx LayoutContext, style Style, container Rect) int {
	if value, ok := explicitOuterHeight(style); ok {
		return clampHeightForStyle(style, value)
	}
	text := element.text()
	textHeight := 0
	font, metrics := ctx.FontForStyle(style)
	lineHeight := lineHeightForStyle(style, metrics.height)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if text != "" {
		width := element.resolvedWidthWithContext(ctx, style)
		if container.Width > 0 || container.Height > 0 {
			width = element.resolvedWidthInWithContext(ctx, style, container)
		}
		insets := boxInsets(style)
		availableW := width - insets.Left - insets.Right
		if availableW < 0 {
			availableW = 0
		}
		if element.kind == ElementKindInput {
			textHeight = lineHeight
		} else if element.kind == ElementKindTextarea {
			lines := element.wrapTextPreserveCached(text, availableW, true, font, charWidth)
			if len(lines) > 0 {
				textHeight = len(lines) * lineHeight
			}
		} else {
			lines := element.wrapTextLinesCachedStyle(text, availableW, font, charWidth, style)
			if len(lines) > 0 {
				textHeight = len(lines) * lineHeight
			}
		}
		baseHeight := textHeight + insets.Top + insets.Bottom
		switch element.kind {
		case ElementKindButton:
			if baseHeight < defaultButtonHeight {
				baseHeight = defaultButtonHeight
			}
		}
		return clampHeightForStyle(style, baseHeight)
	}
	if text == "" && element.isTextInput() {
		textHeight = lineHeight
	}
	insets := boxInsets(style)
	baseHeight := textHeight + insets.Top + insets.Bottom
	switch element.kind {
	case ElementKindButton:
		if baseHeight < defaultButtonHeight {
			baseHeight = defaultButtonHeight
		}
	}
	return clampHeightForStyle(style, baseHeight)
}
