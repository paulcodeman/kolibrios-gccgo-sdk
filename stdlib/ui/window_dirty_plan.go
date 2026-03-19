package ui

type windowDirtyPlanMode uint8

const (
	windowDirtyPlanNone windowDirtyPlanMode = iota
	windowDirtyPlanLayout
	windowDirtyPlanRebuild
	windowDirtyPlanNodeUpdate
)

type windowDirtyPlan struct {
	mode         windowDirtyPlanMode
	dirty        Rect
	dirtySet     bool
	full         Rect
	scrollOffset int
	dirtyNodes   []Node
}

func (window *Window) initialDirtyPlan() windowDirtyPlan {
	plan := windowDirtyPlan{}
	if window == nil {
		return plan
	}
	plan.dirty = window.dirty
	plan.dirtySet = window.dirtySet
	plan.full = Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	plan.scrollOffset = window.currentFrameScrollPaintOffset()
	return plan
}

func (window *Window) copyDirtyPlanNodes(nodes []Node) []Node {
	if window == nil || len(nodes) == 0 {
		if window != nil {
			window.dirtyPlanNodes = window.dirtyPlanNodes[:0]
		}
		return nil
	}
	window.dirtyPlanNodes = append(window.dirtyPlanNodes[:0], nodes...)
	return window.dirtyPlanNodes
}

func (window *Window) dirtyNodesNeedLayout(nodes []Node) bool {
	if window == nil {
		return false
	}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if element, ok := node.(*Element); ok {
			if element.layoutDirtyInCurrentContainer() {
				return true
			}
			continue
		}
		if aware, ok := node.(LayoutDirtyAware); ok && aware.LayoutDirty() {
			return true
		}
	}
	return false
}

func (window *Window) buildDirtyPlan() windowDirtyPlan {
	plan := window.initialDirtyPlan()
	if window == nil || window.canvas == nil {
		return plan
	}
	window.resetTranslateBlits()
	if !window.backgroundOverride() && window.lastBackground != window.Background {
		window.lastBackground = window.Background
		plan.dirty = plan.full
		plan.dirtySet = true
	}
	if plan.dirtySet && plan.dirty == plan.full {
		return plan
	}
	if window.layoutDirty {
		plan.mode = windowDirtyPlanLayout
		return plan
	}
	if !window.renderListValid || window.nodeBounds == nil {
		plan.mode = windowDirtyPlanRebuild
		return plan
	}
	if len(window.allNodes) == 0 {
		window.resetDirtyQueue()
		return plan
	}
	if window.ImplicitDirty {
		for _, node := range window.allNodes {
			if node == nil {
				continue
			}
			aware, ok := node.(DirtyAware)
			if !ok || !aware.Dirty() {
				continue
			}
			window.noteDirty(node)
		}
	}
	if len(window.dirtyList) == 0 {
		return plan
	}
	plan.dirtyNodes = window.copyDirtyPlanNodes(window.dirtyList)
	window.resetDirtyQueue()
	if window.dirtyNodesNeedLayout(plan.dirtyNodes) {
		plan.mode = windowDirtyPlanLayout
		return plan
	}
	plan.mode = windowDirtyPlanNodeUpdate
	return plan
}

func (window *Window) applyDirtyPlan(plan windowDirtyPlan) bool {
	if window == nil {
		return false
	}
	dirty := plan.dirty
	dirtySet := plan.dirtySet
	switch plan.mode {
	case windowDirtyPlanNone:
		// No structural work needed; keep current dirty state.
	case windowDirtyPlanLayout:
		oldBounds := window.copyNodeBounds()
		oldKeys := copyElementRenderKeys(oldBounds)
		window.layoutFlow()
		window.buildRenderList()
		dirty, dirtySet = window.mergeDirtyBounds(dirty, dirtySet, oldBounds, oldKeys, window.nodeBounds, plan.scrollOffset)
		window.hoverDirty = true
		window.lastMouseValid = false
		window.layoutDirty = false
		window.resetDirtyQueue()
	case windowDirtyPlanRebuild:
		oldBounds := window.copyNodeBounds()
		oldKeys := copyElementRenderKeys(oldBounds)
		window.buildRenderList()
		dirty, dirtySet = window.mergeDirtyBounds(dirty, dirtySet, oldBounds, oldKeys, window.nodeBounds, plan.scrollOffset)
		window.resetDirtyQueue()
	case windowDirtyPlanNodeUpdate:
		for _, node := range plan.dirtyNodes {
			oldBounds := window.nodeBounds[node]
			newBounds := window.nodeVisualBoundsFor(node, true)
			window.nodeBounds[node] = newBounds
			window.noteRetainedLayerDirtyBounds(node, oldBounds, newBounds)
			rawUpdated := UnionRect(oldBounds, newBounds)
			updated := rawUpdated
			if exposed, ok := window.tryTranslateBlit(node, oldBounds, newBounds, nil, plan.scrollOffset); ok {
				updated = exposed
			} else if plan.scrollOffset != 0 && !updated.Empty() {
				updated.Y += plan.scrollOffset
			}
			if !updated.Empty() {
				if dirtySet {
					dirty = UnionRect(dirty, updated)
				} else {
					dirty = updated
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
	}
	window.dirty = dirty
	window.dirtySet = dirtySet
	return dirtySet
}
