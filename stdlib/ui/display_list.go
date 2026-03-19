package ui

import "kos"

type DisplayList struct {
	items        []renderItem
	rootClip     clipState
	scrollOffset int
}

func (list DisplayList) itemPaintState(item renderItem, full bool, dirty Rect) (Rect, Rect, bool, bool) {
	if item.node == nil || item.skipPaint {
		return Rect{}, Rect{}, false, false
	}
	paint := item.paint
	if list.scrollOffset != 0 {
		paint.Y += list.scrollOffset
	}
	if paint.Empty() {
		return Rect{}, Rect{}, false, false
	}
	actual := paint
	if list.rootClip.set {
		actual = IntersectRect(actual, list.rootClip.rect)
		if actual.Empty() {
			return Rect{}, Rect{}, false, false
		}
	}
	if !full {
		if dirty.Empty() || IntersectRect(actual, dirty).Empty() {
			return Rect{}, Rect{}, false, false
		}
	}
	clipSet := false
	clipRect := Rect{}
	if !full && !dirty.Empty() && nodeAllowsDirtyClip(item.node, dirty) {
		clipRect = dirty
		clipSet = true
		actual = IntersectRect(actual, dirty)
	}
	if item.clip.set {
		itemClip := item.clip.rect
		if list.scrollOffset != 0 {
			itemClip.Y += list.scrollOffset
		}
		if clipSet {
			clipRect = IntersectRect(clipRect, itemClip)
		} else {
			clipRect = itemClip
			clipSet = true
		}
		actual = IntersectRect(actual, itemClip)
	}
	if actual.Empty() {
		return Rect{}, Rect{}, false, false
	}
	return actual, clipRect, clipSet, true
}

func (list DisplayList) itemOpaqueCoverRect(item renderItem, full bool, dirty Rect) (Rect, bool) {
	paint, _, _, ok := list.itemPaintState(item, full, dirty)
	if !ok {
		return Rect{}, false
	}
	cover, ok := nodeOpaqueCoverRect(item.node)
	if !ok {
		return Rect{}, false
	}
	if list.scrollOffset != 0 {
		cover.Y += list.scrollOffset
	}
	cover = IntersectRect(cover, paint)
	if cover.Empty() {
		return Rect{}, false
	}
	return cover, true
}

func nodeAllowsDirtyClip(node Node, dirty Rect) bool {
	if node == nil {
		return false
	}
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return false
		}
		style := current.effectiveStyle()
		if current.isTextInput() {
			clipRect := current.textInputDirtyClipRect(style)
			return !clipRect.Empty() && rectContainsRect(clipRect, dirty)
		}
		if elementShowsDefaultFocusRing(current) {
			return false
		}
		if resolveBorderRadius(style).Active() {
			return false
		}
		if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
			return false
		}
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			return false
		}
		return true
	case *DocumentView:
		if current == nil {
			return false
		}
		style := current.effectiveStyle()
		if resolveBorderRadius(style).Active() {
			return false
		}
		if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
			return false
		}
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			return false
		}
		return true
	}
	return true
}

func (element *Element) canUseDirtyClip(style Style) bool {
	if element == nil || FastNoCache {
		return false
	}
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	cacheable, _, _ := element.cacheInfo(style, rect)
	return cacheable
}

func (window *Window) scrollPaintOffset() int {
	if window == nil {
		return 0
	}
	return window.currentFrameScrollPaintOffset()
}

func (list DisplayList) Items() []renderItem {
	return list.items
}

func (list DisplayList) ScrollOffset() int {
	return list.scrollOffset
}

func (list DisplayList) Paint(window *Window, full bool, dirty Rect, stats *FrameStats) {
	if window == nil || window.canvas == nil {
		return
	}
	var skip []bool
	if DisplayListOcclusionCulling && len(list.items) > 1 {
		skip = make([]bool, len(list.items))
		covers := make([]Rect, 0, 8)
		for i := len(list.items) - 1; i >= 0; i-- {
			item := list.items[i]
			paint, _, _, ok := list.itemPaintState(item, full, dirty)
			if !ok {
				continue
			}
			if rectCoveredByAny(paint, covers) {
				skip[i] = true
				continue
			}
			if cover, ok := list.itemOpaqueCoverRect(item, full, dirty); ok {
				covers = append(covers, cover)
			}
		}
	}
	if list.rootClip.set {
		window.canvas.PushClip(list.rootClip.rect)
		defer window.canvas.PopClip()
	}
	for i, item := range list.items {
		if item.node == nil {
			continue
		}
		if item.skipPaint {
			if aware, ok := item.node.(DirtyAware); ok {
				aware.ClearDirty()
			}
			continue
		}
		if skip != nil && skip[i] {
			if aware, ok := item.node.(DirtyAware); ok {
				aware.ClearDirty()
			}
			continue
		}
		_, clipRect, clipSet, ok := list.itemPaintState(item, full, dirty)
		if !ok {
			continue
		}
		var element *Element
		if el, ok := item.node.(*Element); ok && el != nil {
			element = el
		}
		if clipSet {
			window.canvas.PushClip(clipRect)
		}
		if stats != nil && !window.DisableNodeTiming {
			start := kos.UptimeNanoseconds()
			if list.scrollOffset != 0 {
				if offsetAware, ok := item.node.(OffsetDrawAware); ok {
					offsetAware.DrawToOffset(window.canvas, list.scrollOffset)
				} else if element != nil {
					window.drawElementWithOffset(element, list.scrollOffset)
				} else {
					item.node.DrawTo(window.canvas)
				}
			} else {
				item.node.DrawTo(window.canvas)
			}
			stats.NodesNs += kos.UptimeNanoseconds() - start
		} else if list.scrollOffset != 0 {
			if offsetAware, ok := item.node.(OffsetDrawAware); ok {
				offsetAware.DrawToOffset(window.canvas, list.scrollOffset)
			} else if element != nil {
				window.drawElementWithOffset(element, list.scrollOffset)
			} else {
				item.node.DrawTo(window.canvas)
			}
		} else {
			item.node.DrawTo(window.canvas)
		}
		if clipSet {
			window.canvas.PopClip()
		}
		if aware, ok := item.node.(DirtyAware); ok {
			aware.ClearDirty()
		}
	}
}

func (list DisplayList) Find(x int, y int) Node {
	for i := len(list.items) - 1; i >= 0; i-- {
		item := list.items[i]
		if item.node == nil {
			continue
		}
		paint := item.paint
		if list.scrollOffset != 0 {
			paint.Y += list.scrollOffset
		}
		if paint.Contains(x, y) {
			return item.node
		}
	}
	return nil
}
