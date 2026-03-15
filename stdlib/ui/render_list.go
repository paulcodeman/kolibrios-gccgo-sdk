package ui

type renderItem struct {
	node   Node
	bounds Rect
	paint  Rect
	clip   clipState
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
	window.renderList = window.renderList[:0]
	window.allNodes = window.allNodes[:0]
	window.tinyglNodes = window.tinyglNodes[:0]
	if window.renderIndex == nil {
		window.renderIndex = make(map[Node]int)
	} else {
		clearRenderIndex(window.renderIndex)
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
	visited := window.renderVisited
	if visited == nil {
		visited = make(map[Node]struct{})
		window.renderVisited = visited
	} else {
		clearVisited(visited)
	}
	window.appendRenderItems(window.nodes, clipState{}, visited)
	window.updateScrollMetrics()
	scrollOffset := 0
	if window.scrollEnabled() && window.scrollY != 0 {
		scrollOffset = -window.scrollY
	}
	window.hitGrid.build(window.client, window.renderList, scrollOffset)
	window.renderListValid = true
}

func (window *Window) appendRenderItems(nodes []Node, clip clipState, visited map[Node]struct{}) {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if visited != nil {
			if _, ok := visited[node]; ok {
				continue
			}
			visited[node] = struct{}{}
		}
		window.allNodes = append(window.allNodes, node)
		element, isElement := node.(*Element)
		hidden := nodeHidden(node)
		bounds := window.nodeVisualBoundsFor(node, true)
		window.nodeBounds[node] = bounds
		if !hidden {
			paint := bounds
			if clip.set {
				paint = IntersectRect(paint, clip.rect)
			}
			index := len(window.renderList)
			window.renderList = append(window.renderList, renderItem{
				node:   node,
				bounds: bounds,
				paint:  paint,
				clip:   clip,
			})
			window.renderIndex[node] = index
			if WindowEnableTinyGL && isElement && element != nil && element.kind == ElementKindTinyGL {
				window.tinyglNodes = append(window.tinyglNodes, element)
			}
		} else {
			continue
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
		window.appendRenderItems(element.Children, childClip, visited)
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
