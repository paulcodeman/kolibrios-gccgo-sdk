package ui

func (window *Window) noteDirty(node Node) {
	if window == nil || node == nil {
		return
	}
	if window.dirtyCandidates == nil {
		window.dirtyCandidates = make(map[Node]struct{})
	}
	window.dirtyCandidates[node] = struct{}{}
}

func (window *Window) noteHandlerMayMutate(target Node) {
	if window == nil || target == nil {
		return
	}
	if element, ok := target.(*Element); ok {
		if element.OnClick != nil {
			window.hoverDirty = true
			window.lastMouseValid = false
		}
		return
	}
	// Unknown node types may mutate arbitrary state.
	window.hoverDirty = true
	window.lastMouseValid = false
}

func (window *Window) Invalidate(rect Rect) {
	if window == nil || rect.Empty() {
		return
	}
	window.hoverDirty = true
	window.lastMouseValid = false
	client := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	clamped := IntersectRect(rect, client)
	if clamped.Empty() {
		return
	}
	if window.dirtySet {
		window.dirty = UnionRect(window.dirty, clamped)
		return
	}
	window.dirty = clamped
	window.dirtySet = true
}

func (window *Window) InvalidateContent(rect Rect) {
	if window == nil || rect.Empty() {
		return
	}
	if window.scrollEnabled() && window.scrollY != 0 {
		rect.Y -= window.scrollY
	}
	window.Invalidate(rect)
}

func (window *Window) collectDirty() bool {
	if window == nil || window.canvas == nil {
		return false
	}
	dirty := window.dirty
	dirtySet := window.dirtySet
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	scrollOffset := 0
	if window.scrollEnabled() && window.scrollY != 0 {
		scrollOffset = -window.scrollY
	}
	if !window.backgroundOverride() {
		if window.lastBackground != window.Background {
			window.lastBackground = window.Background
			dirty = full
			dirtySet = true
		}
	}
	if dirtySet && dirty == full {
		window.dirty = dirty
		window.dirtySet = true
		return true
	}
	if window.layoutDirty {
		oldBounds := window.copyNodeBounds()
		window.layoutFlow()
		window.buildRenderList()
		if window.scrollEnabled() && window.scrollY != 0 {
			scrollOffset = -window.scrollY
		} else {
			scrollOffset = 0
		}
		dirty, dirtySet = mergeDirtyBounds(dirty, dirtySet, oldBounds, window.nodeBounds, scrollOffset)
		window.hoverDirty = true
		window.lastMouseValid = false
		window.layoutDirty = false
		if window.dirtyCandidates != nil {
			clearVisited(window.dirtyCandidates)
		}
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}
	if !window.renderListValid || window.nodeBounds == nil {
		oldBounds := window.copyNodeBounds()
		window.buildRenderList()
		if window.scrollEnabled() && window.scrollY != 0 {
			scrollOffset = -window.scrollY
		} else {
			scrollOffset = 0
		}
		dirty, dirtySet = mergeDirtyBounds(dirty, dirtySet, oldBounds, window.nodeBounds, scrollOffset)
		if window.dirtyCandidates != nil {
			clearVisited(window.dirtyCandidates)
		}
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}

	nodes := window.allNodes
	if len(nodes) == 0 {
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}

	dirtyMap := window.dirtyCandidates
	if window.ImplicitDirty {
		for _, node := range nodes {
			if node == nil {
				continue
			}
			aware, ok := node.(DirtyAware)
			if !ok || !aware.Dirty() {
				continue
			}
			if dirtyMap == nil {
				dirtyMap = make(map[Node]struct{})
				window.dirtyCandidates = dirtyMap
			}
			dirtyMap[node] = struct{}{}
		}
	}
	if len(dirtyMap) == 0 {
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}

	dirtyList := window.dirtyList[:0]
	needsLayout := false
	for node := range dirtyMap {
		if node == nil {
			continue
		}
		dirtyList = append(dirtyList, node)
		if needsLayout {
			continue
		}
		if element, ok := node.(*Element); ok {
			if element.layoutDirtyInCurrentContainer() {
				needsLayout = true
			}
			continue
		}
		if aware, ok := node.(interface{ LayoutDirty() bool }); ok && aware.LayoutDirty() {
			needsLayout = true
		}
	}
	window.dirtyList = dirtyList
	if dirtyMap != nil {
		clearVisited(dirtyMap)
	}

	if len(dirtyList) == 0 {
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}

	if needsLayout {
		oldBounds := window.copyNodeBounds()
		window.layoutFlow()
		window.buildRenderList()
		if window.scrollEnabled() && window.scrollY != 0 {
			scrollOffset = -window.scrollY
		} else {
			scrollOffset = 0
		}
		dirty, dirtySet = mergeDirtyBounds(dirty, dirtySet, oldBounds, window.nodeBounds, scrollOffset)
		window.hoverDirty = true
		window.lastMouseValid = false
		if dirtySet {
			window.dirty = dirty
			window.dirtySet = true
		}
		return window.dirtySet
	}

	for _, node := range dirtyList {
		oldBounds := window.nodeBounds[node]
		newBounds := window.nodeVisualBoundsFor(node, true)
		window.nodeBounds[node] = newBounds
		union := UnionRect(oldBounds, newBounds)
		if scrollOffset != 0 && !union.Empty() {
			union.Y += scrollOffset
		}
		if !union.Empty() {
			if dirtySet {
				dirty = UnionRect(dirty, union)
			} else {
				dirty = union
				dirtySet = true
			}
		}
		if idx, ok := window.renderIndex[node]; ok && idx >= 0 && idx < len(window.renderList) {
			item := window.renderList[idx]
			item.bounds = newBounds
			paint := newBounds
			if item.clip.set {
				paint = IntersectRect(paint, item.clip.rect)
			}
			item.paint = paint
			window.renderList[idx] = item
		}
	}
	if dirtySet {
		window.dirty = dirty
		window.dirtySet = true
	}
	return window.dirtySet
}

func (window *Window) copyNodeBounds() map[Node]Rect {
	if window == nil || len(window.nodeBounds) == 0 {
		return nil
	}
	copied := make(map[Node]Rect, len(window.nodeBounds))
	for node, bounds := range window.nodeBounds {
		copied[node] = bounds
	}
	return copied
}

func mergeDirtyBounds(dirty Rect, dirtySet bool, oldBounds map[Node]Rect, newBounds map[Node]Rect, scrollOffset int) (Rect, bool) {
	if len(newBounds) == 0 && len(oldBounds) == 0 {
		return dirty, dirtySet
	}
	if oldBounds == nil {
		oldBounds = map[Node]Rect{}
	}
	for node, bounds := range newBounds {
		if old, ok := oldBounds[node]; ok {
			if old != bounds {
				union := UnionRect(old, bounds)
				if scrollOffset != 0 && !union.Empty() {
					union.Y += scrollOffset
				}
				if !union.Empty() {
					if dirtySet {
						dirty = UnionRect(dirty, union)
					} else {
						dirty = union
						dirtySet = true
					}
				}
			}
			delete(oldBounds, node)
			continue
		}
		if bounds.Empty() {
			continue
		}
		if scrollOffset != 0 {
			bounds.Y += scrollOffset
		}
		if dirtySet {
			dirty = UnionRect(dirty, bounds)
		} else {
			dirty = bounds
			dirtySet = true
		}
	}
	for _, old := range oldBounds {
		if old.Empty() {
			continue
		}
		if scrollOffset != 0 {
			old.Y += scrollOffset
		}
		if dirtySet {
			dirty = UnionRect(dirty, old)
		} else {
			dirty = old
			dirtySet = true
		}
	}
	return dirty, dirtySet
}

type nodeState struct {
	node   Node
	bounds Rect
	dirty  bool
}

func (window *Window) collectNodeStates(nodes []Node, out *[]nodeState, recompute bool) {
	if window == nil {
		return
	}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		state := nodeState{
			node:   node,
			bounds: window.nodeVisualBoundsFor(node, recompute),
		}
		if aware, ok := node.(DirtyAware); ok {
			state.dirty = aware.Dirty()
		}
		*out = append(*out, state)
		if element, ok := node.(*Element); ok && len(element.Children) > 0 {
			window.collectNodeStates(element.Children, out, recompute)
		}
	}
}
