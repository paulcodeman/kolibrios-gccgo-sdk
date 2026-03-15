package ui

func (window *Window) layoutFlow() {
	if window == nil || window.canvas == nil {
		return
	}
	container := window.contentRect()
	visited := window.layoutVisited
	if visited == nil {
		visited = make(map[Node]struct{})
		window.layoutVisited = visited
	} else {
		clearVisited(visited)
	}
	window.layoutFlowIn(container, window.nodes, visited)
	window.renderListValid = false
}

func (window *Window) layoutFlowIn(container Rect, nodes []Node, visited map[Node]struct{}) {
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
		element, ok := node.(*Element)
		if !ok || element == nil {
			continue
		}
		if visited != nil {
			if _, ok := visited[element]; ok {
				continue
			}
			visited[element] = struct{}{}
		}
		style := element.effectiveStyle()
		display := DisplayInline
		if value, ok := resolveDisplay(style.Display); ok {
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
			continue
		}
		element.layoutHidden = false
		position := effectivePosition(style)
		element.layoutPosition = position
		if position == PositionAbsolute {
			element.clearFlow()
			element.applyLayoutIn(window.canvas, container, style)
			if len(element.Children) > 0 {
				childContainer := contentRectFor(element.layoutRect, style)
				window.layoutFlowIn(childContainer, element.Children, visited)
			}
			continue
		}
		width := element.resolvedWidthIn(style, container)
		var margin Spacing
		marginSet := false
		if value, ok := resolveSpacing(style.Margin); ok && value != nil {
			margin = *value
			marginSet = spacingAny(margin)
		}
		element.layoutMargin = margin
		element.layoutMarginSet = marginSet
		outerW := width
		if marginSet {
			outerW += margin.Left + margin.Right
		}
		if display == DisplayBlock {
			if cursorX > 0 {
				cursorX = 0
				cursorY += lineHeight
				lineHeight = 0
			}
		} else if cursorX > 0 && cursorX+outerW > maxWidth {
			cursorX = 0
			cursorY += lineHeight
			lineHeight = 0
		}
		x := cursorX
		y := cursorY
		if marginSet && position != PositionAbsolute {
			x += margin.Left
			y += margin.Top
		}
		absX := container.X + x
		absY := container.Y + y
		element.setFlow(absX, absY)
		element.applyLayoutIn(window.canvas, container, style)
		if len(element.Children) > 0 {
			childContainer := contentRectFor(element.layoutRect, style)
			window.layoutFlowIn(childContainer, element.Children, visited)
			window.adjustAutoWidth(element, style)
			window.adjustAutoHeight(element, style)
		}
		element.updateSubtreeRect()
		outerH := element.layoutRect.Height
		if marginSet {
			outerH += margin.Top + margin.Bottom
		}
		if display == DisplayBlock {
			cursorX = 0
			cursorY += outerH
			lineHeight = 0
			continue
		}
		cursorX += outerW
		if outerH > lineHeight {
			lineHeight = outerH
		}
	}
}

func (window *Window) adjustAutoHeight(element *Element, style Style) bool {
	if element == nil || len(element.Children) == 0 {
		return false
	}
	if _, ok := resolveLength(style.Height); ok {
		return false
	}
	paddingBottom := 0
	if padding, ok := resolveSpacingNormalized(style.Padding); ok {
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
	if _, ok := resolveLength(style.Width); ok {
		return false
	}
	display := DisplayInline
	if value, ok := resolveDisplay(style.Display); ok {
		display = value
	}
	if display != DisplayBlock && display != DisplayInlineBlock {
		return false
	}
	maxRight := maxChildRight(element)
	paddingRight := 0
	if padding, ok := resolveSpacingNormalized(style.Padding); ok {
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
	if visual, ok := node.(VisualBoundsAware); ok {
		return visual.VisualBounds()
	}
	return node.Bounds()
}

func nodeHidden(node Node) bool {
	element, ok := node.(*Element)
	if !ok || element == nil {
		return false
	}
	if display, ok := resolveDisplay(element.effectiveStyle().Display); ok && display == DisplayNone {
		return true
	}
	return false
}
