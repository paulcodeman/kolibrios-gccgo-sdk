package ui

type renderItem struct {
	node      Node
	bounds    Rect
	paint     Rect
	clip      clipState
	skipPaint bool
}

func clearRenderIndex(m map[Node]int) {
	for k := range m {
		delete(m, k)
	}
}

func clearNodeBounds(m map[Node]Rect) {
	for k := range m {
		delete(m, k)
	}
}

func clearVisited(m map[Node]struct{}) {
	for k := range m {
		delete(m, k)
	}
}

func nextNodeGeneration(gen *uint32) uint32 {
	*gen = *gen + 1
	if *gen == 0 {
		*gen = 1
	}
	return *gen
}

func (window *Window) ensureRenderList() {
	if window == nil || window.canvas == nil {
		return
	}
	if window.renderListValid {
		return
	}
	window.buildRenderList()
}

func (window *Window) buildRenderList() {
	if window == nil {
		return
	}
	window.invalidateWindowDisplayItemsState()
	window.renderList = window.renderList[:0]
	window.allNodes = window.allNodes[:0]
	window.tinyglNodes = window.tinyglNodes[:0]
	window.focusables = window.focusables[:0]
	if window.renderIndex == nil {
		window.renderIndex = make(map[Node]int)
	} else {
		clearRenderIndex(window.renderIndex)
	}
	if window.focusIndex == nil {
		window.focusIndex = make(map[Node]int)
	} else {
		clearRenderIndex(window.focusIndex)
	}
	if window.nodeBounds == nil {
		window.nodeBounds = make(map[Node]Rect)
	} else {
		clearNodeBounds(window.nodeBounds)
	}
	if window.canvas == nil {
		window.renderListValid = true
		return
	}
	gen := nextNodeGeneration(&window.renderVisitGen)
	if window.renderVisited != nil {
		clearVisited(window.renderVisited)
	}
	window.appendRenderItems(window.nodes, clipState{}, gen, false, false)
	window.noteScrollMetricsBoundsChanged()
	window.updateScrollMetrics()
	window.invalidateHitGrid()
	window.renderListValid = true
}

func (window *Window) appendRenderItems(nodes []Node, clip clipState, gen uint32, inheritedHidden bool, suppressPaint bool) {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if element, ok := node.(*Element); ok && element != nil {
			if element.renderVisitGen == gen {
				continue
			}
			element.renderVisitGen = gen
		} else {
			visited := window.renderVisited
			if visited == nil {
				visited = make(map[Node]struct{})
				window.renderVisited = visited
			}
			if _, ok := visited[node]; ok {
				continue
			}
			visited[node] = struct{}{}
		}
		window.allNodes = append(window.allNodes, node)
		element, isElement := node.(*Element)
		if nodeHidden(node) {
			continue
		}
		currentHidden := inheritedHidden
		switch current := node.(type) {
		case *Element:
			if current != nil {
				currentHidden = styleHiddenByVisibility(current.effectiveStyle(), inheritedHidden)
			}
		case *DocumentView:
			if current != nil {
				currentHidden = styleHiddenByVisibility(current.effectiveStyle(), inheritedHidden)
			}
		}
		bounds := window.nodeVisualBoundsFor(node, true)
		window.nodeBounds[node] = bounds
		if !currentHidden {
			paint := bounds
			if clip.set {
				paint = IntersectRect(paint, clip.rect)
			}
			index := len(window.renderList)
			window.renderList = append(window.renderList, renderItem{
				node:      node,
				bounds:    bounds,
				paint:     paint,
				clip:      clip,
				skipPaint: suppressPaint,
			})
			window.renderIndex[node] = index
			if WindowEnableTinyGL && isElement && element != nil && element.kind == ElementKindTinyGL {
				window.tinyglNodes = append(window.tinyglNodes, element)
			}
		}
		if !isElement || element == nil || len(element.Children) == 0 {
			continue
		}
		style := element.effectiveStyle()
		rect := element.layoutRect
		if rect.Empty() {
			rect = element.Bounds()
		}
		clipX, clipY := overflowClipAxes(style)
		childClip := clip
		if clipX || clipY {
			childClip = window.mergeClip(clip, rect, style, clipX, clipY)
		}
		childSuppress := suppressPaint || element.useRetainedSubtreeLayer(style)
		window.appendRenderItems(element.Children, childClip, gen, currentHidden, childSuppress)
	}
}

func (window *Window) mergeClip(parent clipState, rect Rect, style Style, clipX bool, clipY bool) clipState {
	if window == nil || window.canvas == nil {
		return clipState{rect: Rect{}, set: true}
	}
	if !clipX && !clipY {
		return parent
	}
	canvasBounds := Rect{X: 0, Y: 0, Width: window.canvas.Width(), Height: window.canvas.Height()}
	base := canvasBounds
	if parent.set {
		base = parent.rect
	}
	clipRect := contentRectFor(rect, style)
	if clipX {
		base.X = clipRect.X
		base.Width = clipRect.Width
	}
	if clipY {
		base.Y = clipRect.Y
		base.Height = clipRect.Height
	}
	base = IntersectRect(base, canvasBounds)
	if parent.set {
		base = IntersectRect(base, parent.rect)
	}
	return clipState{rect: base, set: true}
}
