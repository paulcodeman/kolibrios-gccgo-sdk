package ui

func (window *Window) layoutFlow() {
	if window == nil || window.canvas == nil {
		return
	}
	container := window.contentRect()
	ctx := window.layoutContext()
	gen := nextNodeGeneration(&window.layoutVisitGen)
	window.layoutFlowIn(ctx, container, window.nodes, gen)
	window.renderListValid = false
}

func (window *Window) layoutFlowIn(ctx LayoutContext, container Rect, nodes []Node, gen uint32) {
	if window == nil || window.canvas == nil {
		return
	}
	maxWidth := container.Width
	if maxWidth <= 0 {
		return
	}
	cursorX := 0
	cursorY := 0
	lineHeight := 0
	for _, node := range nodes {
		if element, ok := node.(*Element); ok && element != nil {
			cursorX, cursorY, lineHeight = window.layoutFlowElement(ctx, container, element, gen, cursorX, cursorY, lineHeight, maxWidth)
			continue
		}
		if view, ok := node.(*DocumentView); ok && view != nil {
			cursorX, cursorY, lineHeight = window.layoutFlowDocumentView(ctx, container, view, gen, cursorX, cursorY, lineHeight, maxWidth)
		}
	}
}

func (window *Window) layoutFlowElement(ctx LayoutContext, container Rect, element *Element, gen uint32, cursorX int, cursorY int, lineHeight int, maxWidth int) (int, int, int) {
	if element == nil {
		return cursorX, cursorY, lineHeight
	}
	if element.layoutVisitGen == gen {
		return cursorX, cursorY, lineHeight
	}
	element.layoutVisitGen = gen
	style := element.effectiveStyle()
	display := DisplayInline
	if value, ok := resolveDisplay(style.display); ok {
		display = value
	}
	if display == DisplayNone {
		element.layoutHidden = true
		element.layoutMarginSet = false
		element.layoutMargin = Spacing{}
		element.layoutPosition = PositionStatic
		element.clearFlow()
		element.layoutRect = Rect{}
		element.visualRect = Rect{}
		element.subtreeRect = Rect{}
		return cursorX, cursorY, lineHeight
	}
	element.layoutHidden = false
	position := effectivePosition(style)
	element.layoutPosition = position
	if position == PositionAbsolute {
		element.clearFlow()
		element.applyLayoutWithContext(ctx, container, style)
		if len(element.Children) > 0 {
			childContainer := contentRectFor(element.layoutRect, style)
			window.layoutFlowIn(ctx, childContainer, element.Children, gen)
		}
		return cursorX, cursorY, lineHeight
	}
	width := element.resolvedWidthInWithContext(ctx, style, container)
	margin, marginSet := resolvedMargin(style)
	element.layoutMargin = margin
	element.layoutMarginSet = marginSet
	outerW := width
	if marginSet {
		outerW += margin.Left + margin.Right
	}
	cursorX, cursorY, lineHeight = wrapFlowCursor(display, outerW, cursorX, cursorY, lineHeight, maxWidth)
	x := cursorX
	y := cursorY
	if marginSet {
		x += margin.Left
		y += margin.Top
	}
	element.setFlow(container.X+x, container.Y+y)
	element.applyLayoutWithContext(ctx, container, style)
	if len(element.Children) > 0 {
		childContainer := contentRectFor(element.layoutRect, style)
		window.layoutFlowIn(ctx, childContainer, element.Children, gen)
		window.adjustAutoWidth(element, style)
		window.adjustAutoHeight(element, style)
	}
	element.updateSubtreeRect()
	return advanceFlowCursor(display, outerW, element.layoutRect.Height, margin, marginSet, cursorX, cursorY, lineHeight)
}

func (window *Window) layoutFlowDocumentView(ctx LayoutContext, container Rect, view *DocumentView, gen uint32, cursorX int, cursorY int, lineHeight int, maxWidth int) (int, int, int) {
	if view == nil {
		return cursorX, cursorY, lineHeight
	}
	if view.layoutVisitGen == gen {
		return cursorX, cursorY, lineHeight
	}
	view.layoutVisitGen = gen
	style := view.effectiveStyle()
	display := DisplayBlock
	if value, ok := resolveDisplay(style.display); ok {
		display = value
	}
	if display == DisplayNone {
		view.clearFlow()
		view.layoutRect = Rect{}
		view.visualRect = Rect{}
		view.layoutDirty = false
		return cursorX, cursorY, lineHeight
	}
	position := effectivePosition(style)
	if position == PositionAbsolute {
		view.clearFlow()
		view.applyLayoutWithContext(ctx, container, style)
		return cursorX, cursorY, lineHeight
	}
	width := view.resolvedWidthIn(style, container)
	margin, marginSet := resolvedMargin(style)
	outerW := width
	if marginSet {
		outerW += margin.Left + margin.Right
	}
	cursorX, cursorY, lineHeight = wrapFlowCursor(display, outerW, cursorX, cursorY, lineHeight, maxWidth)
	x := cursorX
	y := cursorY
	if marginSet {
		x += margin.Left
		y += margin.Top
	}
	view.setFlow(container.X+x, container.Y+y)
	view.applyLayoutWithContext(ctx, container, style)
	outerH := view.layoutRect.Height
	if marginSet {
		outerH += margin.Top + margin.Bottom
	}
	if display == DisplayBlock {
		return 0, cursorY + outerH, 0
	}
	cursorX += outerW
	if outerH > lineHeight {
		lineHeight = outerH
	}
	return cursorX, cursorY, lineHeight
}

func resolvedMargin(style Style) (Spacing, bool) {
	if value, ok := resolveSpacing(style.margin); ok && value != nil {
		margin := *value
		return margin, spacingAny(margin)
	}
	return Spacing{}, false
}

func wrapFlowCursor(display DisplayMode, outerW int, cursorX int, cursorY int, lineHeight int, maxWidth int) (int, int, int) {
	if display == DisplayBlock {
		if cursorX > 0 {
			return 0, cursorY + lineHeight, 0
		}
		return cursorX, cursorY, lineHeight
	}
	if cursorX > 0 && cursorX+outerW > maxWidth {
		return 0, cursorY + lineHeight, 0
	}
	return cursorX, cursorY, lineHeight
}

func advanceFlowCursor(display DisplayMode, outerW int, height int, margin Spacing, marginSet bool, cursorX int, cursorY int, lineHeight int) (int, int, int) {
	outerH := height
	if marginSet {
		outerH += margin.Top + margin.Bottom
	}
	if display == DisplayBlock {
		return 0, cursorY + outerH, 0
	}
	cursorX += outerW
	if outerH > lineHeight {
		lineHeight = outerH
	}
	return cursorX, cursorY, lineHeight
}

func (window *Window) adjustAutoHeight(element *Element, style Style) bool {
	if element == nil || len(element.Children) == 0 {
		return false
	}
	if _, ok := resolveLength(style.height); ok {
		return false
	}
	paddingBottom := 0
	if padding, ok := resolveSpacingNormalized(style.padding); ok {
		paddingBottom = padding.Bottom
	}
	borderWidth := borderWidthFor(style)
	maxBottom := maxChildBottom(element)
	desired := (maxBottom - element.layoutRect.Y) + paddingBottom + borderWidth
	if desired < 0 {
		desired = 0
	}
	if desired <= element.layoutRect.Height {
		return false
	}
	element.layoutRect.Height = desired
	element.visualRect = element.visualBoundsFor(element.layoutRect, style)
	return true
}

func (window *Window) adjustAutoWidth(element *Element, style Style) bool {
	if element == nil || len(element.Children) == 0 {
		return false
	}
	if _, ok := resolveLength(style.width); ok {
		return false
	}
	display := DisplayInline
	if value, ok := resolveDisplay(style.display); ok {
		display = value
	}
	if display != DisplayBlock && display != DisplayInlineBlock {
		return false
	}
	maxRight := maxChildRight(element)
	paddingRight := 0
	if padding, ok := resolveSpacingNormalized(style.padding); ok {
		paddingRight = padding.Right
	}
	borderWidth := borderWidthFor(style)
	desired := (maxRight - element.layoutRect.X) + paddingRight + borderWidth
	if desired < 0 {
		desired = 0
	}
	if desired <= element.layoutRect.Width {
		return false
	}
	element.layoutRect.Width = desired
	element.visualRect = element.visualBoundsFor(element.layoutRect, style)
	return true
}

func (window *Window) nodeVisualBounds(node Node) Rect {
	return window.nodeVisualBoundsFor(node, false)
}

func (window *Window) nodeVisualBoundsFor(node Node, recompute bool) Rect {
	if node == nil {
		return Rect{}
	}
	if element, ok := node.(*Element); ok && element != nil && recompute {
		rect := element.layoutRect
		if rect.Empty() {
			rect = element.Bounds()
		}
		visual := element.visualBoundsFor(rect, element.effectiveStyle())
		element.visualRect = visual
		return visual
	}
	if view, ok := node.(*DocumentView); ok && view != nil && recompute {
		rect := view.layoutRect
		if rect.Empty() {
			rect = view.Bounds()
		}
		visual := visualBoundsForStyle(rect, view.effectiveStyle(), false)
		view.visualRect = visual
		return visual
	}
	if visual, ok := node.(VisualBoundsAware); ok {
		return visual.VisualBounds()
	}
	return node.Bounds()
}

func nodeHidden(node Node) bool {
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return false
		}
		if display, ok := resolveDisplay(current.effectiveStyle().display); ok && display == DisplayNone {
			return true
		}
	case *DocumentView:
		if current == nil {
			return false
		}
		if display, ok := resolveDisplay(current.effectiveStyle().display); ok && display == DisplayNone {
			return true
		}
	}
	return false
}
