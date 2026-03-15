package ui

func (window *Window) hitTest(x int, y int) Node {
	if window == nil {
		return nil
	}
	if window.renderListValid && len(window.renderList) > 0 {
		if node, ok := window.hitGrid.find(x, y, window.renderList); ok {
			return node
		}
		offsetY := 0
		if window.scrollEnabled() && window.scrollY != 0 {
			offsetY = -window.scrollY
		}
		for i := len(window.renderList) - 1; i >= 0; i-- {
			item := window.renderList[i]
			if item.node == nil {
				continue
			}
			paint := item.paint
			if offsetY != 0 {
				paint.Y += offsetY
			}
			if paint.Contains(x, y) {
				return item.node
			}
		}
		return nil
	}
	if window.scrollEnabled() && window.scrollY != 0 {
		y += window.scrollY
	}
	return window.hitTestNodes(window.nodes, x, y)
}

func (window *Window) hitTestNodes(nodes []Node, x int, y int) Node {
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if node == nil {
			continue
		}
		if nodeHidden(node) {
			continue
		}
		if element, ok := node.(*Element); ok && len(element.Children) > 0 {
			if !element.subtreeBounds().Contains(x, y) {
				continue
			}
			style := element.effectiveStyle()
			clipX, clipY := overflowClipAxes(style)
			skipChildren := false
			if clipX || clipY {
				rect := element.layoutRect
				if rect.Empty() {
					rect = element.Bounds()
				}
				clipRect := contentRectFor(rect, style)
				if clipX && (x < clipRect.X || x >= clipRect.X+clipRect.Width) {
					skipChildren = true
				}
				if clipY && (y < clipRect.Y || y >= clipRect.Y+clipRect.Height) {
					skipChildren = true
				}
			}
			if !skipChildren {
				if hit := window.hitTestNodes(element.Children, x, y); hit != nil {
					return hit
				}
			}
		}
		if element, ok := node.(*Element); ok && element != nil {
			rect := element.layoutRect
			if rect.Empty() {
				rect = element.Bounds()
			}
			visual := element.visualBoundsFor(rect, element.effectiveStyle())
			if visual.Contains(x, y) {
				return node
			}
			continue
		}
		if node.Bounds().Contains(x, y) {
			return node
		}
	}
	return nil
}
