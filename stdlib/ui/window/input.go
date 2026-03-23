package ui

func (window *Window) hitTest(x int, y int) Node {
	if window == nil {
		return nil
	}
	if window.renderListValid && len(window.renderList) > 0 {
		display := window.currentDisplayList()
		if window.ensureHitGridWithDisplay(display) {
			if node, ok := window.hitGrid.find(x, y, display); ok {
				return node
			}
		}
		return display.Find(x, y)
	}
	if window.scrollEnabled() && window.scrollY != 0 {
		y += window.scrollY
	}
	return window.hitTestNodes(window.nodes, x, y, false)
}

func (window *Window) hitTestNodes(nodes []Node, x int, y int, inheritedHidden bool) Node {
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if node == nil {
			continue
		}
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
		if element, ok := node.(*Element); ok && len(element.Children) > 0 {
			if !element.subtreeBounds().Contains(x, y) {
				continue
			}
			style := element.effectiveStyle()
			clipX, clipY := paintClipAxes(style)
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
				if hit := window.hitTestNodes(element.Children, x, y, currentHidden); hit != nil {
					return hit
				}
			}
		}
		if currentHidden {
			continue
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
		if visual, ok := node.(VisualBoundsAware); ok {
			if visual.VisualBounds().Contains(x, y) {
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
