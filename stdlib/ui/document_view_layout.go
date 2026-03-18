package ui

func (view *DocumentView) Layout(canvas *Canvas) {
	if view == nil || canvas == nil {
		return
	}
	view.LayoutWithContext(layoutContextForCanvas(canvas))
}

func (view *DocumentView) LayoutWithContext(ctx LayoutContext) {
	if view == nil {
		return
	}
	view.applyLayoutWithContext(ctx, ctx.Viewport, view.effectiveStyle())
}

func (view *DocumentView) DrawTo(canvas *Canvas) {
	view.drawToOffset(canvas, 0)
}

func (view *DocumentView) DrawToOffset(canvas *Canvas, offsetY int) {
	view.drawToOffset(canvas, offsetY)
}

func (view *DocumentView) drawToOffset(canvas *Canvas, offsetY int) {
	if view == nil || canvas == nil {
		return
	}
	style := view.effectiveStyle()
	if display, ok := resolveDisplay(style.Display); ok && display == DisplayNone {
		return
	}
	if view.layoutRect.Empty() || view.LayoutDirty() {
		view.applyLayoutWithContext(layoutContextForCanvas(canvas), Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}, style)
	}
	if view.layoutRect.Empty() {
		return
	}
	rect := view.layoutRect
	if offsetY != 0 {
		rect.Y += offsetY
	}
	drawStyledBox(canvas, rect, style, rect, nil)
	if view.Document == nil {
		return
	}
	content := contentRectFor(rect, style)
	if content.Empty() {
		return
	}
	canvas.PushClip(content)
	if offsetY != 0 {
		view.Document.PaintOffset(canvas, 0, offsetY)
	} else {
		view.Document.Paint(canvas)
	}
	canvas.PopClip()
}

func (view *DocumentView) applyLayoutWithContext(ctx LayoutContext, container Rect, style Style) {
	if view == nil {
		return
	}
	key := view.layoutKeyFor(style, container)
	if !view.layoutDirty && !view.layoutRect.Empty() && documentViewLayoutKeyEqual(key, view.layoutKey) {
		return
	}
	view.layoutKey = key
	width := view.resolvedWidthIn(style, container)
	height, heightSet := resolveLength(style.Height)
	rect := view.resolveRectIn(container, style, width, height)
	content := contentRectFor(rect, style)
	if view.Document != nil && !content.Empty() {
		docCtx := layoutContextWithViewport(ctx, content)
		view.Document.Layout(docCtx)
		if !heightSet {
			height = view.documentContentExtentHeight(content, style)
			rect = view.resolveRectIn(container, style, width, height)
			content = contentRectFor(rect, style)
			if view.Document.Viewport() != content {
				docCtx = layoutContextWithViewport(ctx, content)
				view.Document.Layout(docCtx)
			}
		}
	}
	if !heightSet && height < 0 {
		height = 0
	}
	rect.Height = height
	view.layoutRect = rect
	view.visualRect = visualBoundsForStyle(rect, style, false)
	view.layoutDirty = false
}

func (view *DocumentView) layoutKeyFor(style Style, container Rect) documentViewLayoutKey {
	var position *PositionMode
	if value, ok := resolvePosition(style.Position); ok {
		v := value
		position = &v
	}
	var display *DisplayMode
	if value, ok := resolveDisplay(style.Display); ok {
		v := value
		display = &v
	}
	var left *int
	if value, ok := resolveLength(style.Left); ok {
		v := value
		left = &v
	}
	var top *int
	if value, ok := resolveLength(style.Top); ok {
		v := value
		top = &v
	}
	var right *int
	if value, ok := resolveLength(style.Right); ok {
		v := value
		right = &v
	}
	var bottom *int
	if value, ok := resolveLength(style.Bottom); ok {
		v := value
		bottom = &v
	}
	var styleWidth *int
	if value, ok := resolveLength(style.Width); ok {
		v := value
		styleWidth = &v
	}
	var styleHeight *int
	if value, ok := resolveLength(style.Height); ok {
		v := value
		styleHeight = &v
	}
	var margin *Spacing
	if value, ok := resolveSpacing(style.Margin); ok && value != nil {
		v := *value
		margin = &v
	}
	return documentViewLayoutKey{
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
		styleWidth:  styleWidth,
		styleHeight: styleHeight,
		margin:      margin,
		flowSet:     view.flowSet,
		flowX:       view.flowX,
		flowY:       view.flowY,
	}
}

func (view *DocumentView) resolvedWidthIn(style Style, container Rect) int {
	if value, ok := resolveLength(style.Width); ok {
		return value
	}
	if display, ok := resolveDisplay(style.Display); ok && display == DisplayBlock {
		if effectivePosition(style) != PositionAbsolute {
			width := container.Width
			if margin, ok := resolveSpacing(style.Margin); ok && margin != nil {
				width -= margin.Left + margin.Right
			}
			if width < 0 {
				width = 0
			}
			return width
		}
	}
	if container.Width > 0 {
		return container.Width
	}
	if view.Document != nil {
		insets := boxInsets(style)
		content := view.Document.ContentBounds()
		width := content.X + content.Width - view.Document.Viewport().X
		if width < 0 {
			width = 0
		}
		return width + insets.Left + insets.Right
	}
	return 0
}

func (view *DocumentView) resolveRectIn(container Rect, style Style, width int, height int) Rect {
	base := Rect{
		Width:  width,
		Height: height,
	}
	x, y := view.basePosition(style)
	base.X = x
	base.Y = y
	if !style.HasLayout() {
		return base
	}
	return resolveRect(base, container, style)
}

func (view *DocumentView) basePosition(style Style) (int, int) {
	position := effectivePosition(style)
	x := 0
	y := 0
	if position != PositionAbsolute && view.flowSet {
		x = view.flowX
		y = view.flowY
	}
	if position == PositionAbsolute {
		if value, ok := resolveLength(style.Left); ok {
			x = value
		}
		if value, ok := resolveLength(style.Top); ok {
			y = value
		}
	}
	return x, y
}

func (view *DocumentView) setFlow(x int, y int) bool {
	if view == nil {
		return false
	}
	if view.flowSet && view.flowX == x && view.flowY == y {
		return false
	}
	view.flowSet = true
	view.flowX = x
	view.flowY = y
	return true
}

func (view *DocumentView) clearFlow() bool {
	if view == nil || !view.flowSet {
		return false
	}
	view.flowSet = false
	view.flowX = 0
	view.flowY = 0
	return true
}

func (view *DocumentView) documentContentExtentHeight(content Rect, style Style) int {
	insets := boxInsets(style)
	if view == nil || view.Document == nil || content.Empty() {
		return insets.Top + insets.Bottom
	}
	bounds := view.Document.ContentBounds()
	height := bounds.Y + bounds.Height - content.Y
	if height < 0 {
		height = 0
	}
	return insets.Top + height + insets.Bottom
}
