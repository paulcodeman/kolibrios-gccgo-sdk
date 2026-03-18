package ui

import "kos"

func (view *DocumentView) Focused() bool {
	if view == nil {
		return false
	}
	return view.focused
}

func (view *DocumentView) SetFocus(focus bool) bool {
	if view == nil || view.focused == focus {
		return false
	}
	view.focused = focus
	changed := false
	if !focus {
		changed = view.setFocusNode(nil, DocumentEvent{Type: EventBlur, View: view}) || changed
	}
	if !view.StyleFocus.IsZero() {
		if view.markDirtyForStyle(view.StyleFocus) {
			changed = true
		}
	}
	if !focus {
		view.scrollDrag = false
	}
	return changed
}

func (view *DocumentView) HandleTab(shift bool) bool {
	if view == nil || !view.focused || view.Document == nil {
		return false
	}
	focusables := view.focusableNodes()
	if len(focusables) == 0 {
		return false
	}
	if view.focusNode == nil {
		if shift {
			return view.setFocusNode(focusables[len(focusables)-1], DocumentEvent{Type: EventFocus, View: view})
		}
		return view.setFocusNode(focusables[0], DocumentEvent{Type: EventFocus, View: view})
	}
	index := -1
	for i, node := range focusables {
		if node == view.focusNode {
			index = i
			break
		}
	}
	if index < 0 {
		if shift {
			return view.setFocusNode(focusables[len(focusables)-1], DocumentEvent{Type: EventFocus, View: view})
		}
		return view.setFocusNode(focusables[0], DocumentEvent{Type: EventFocus, View: view})
	}
	if shift {
		index--
		if index < 0 {
			index = len(focusables) - 1
		}
	} else {
		index = (index + 1) % len(focusables)
	}
	return view.setFocusNode(focusables[index], DocumentEvent{Type: EventFocus, View: view})
}

func (view *DocumentView) HandleKey(key kos.KeyEvent) bool {
	if view == nil || !view.focused || key.Empty || key.Hotkey {
		return false
	}
	if view.focusNode != nil {
		event := DocumentEvent{
			Type:    EventKeyDown,
			Key:     key,
			ScrollX: 0,
			ScrollY: view.scrollY,
			View:    view,
			Node:    view.focusNode,
		}
		if dispatchDocumentNodeHandler(view.focusNode.OnKeyDown, view.focusNode, event) {
			return true
		}
	}
	switch {
	case key.Code == 13 || key.Code == 32:
		if view.focusNode != nil {
			event := DocumentEvent{
				Type:    EventClick,
				Button:  MouseLeft,
				Key:     key,
				ScrollY: view.scrollY,
				View:    view,
				Node:    view.focusNode,
			}
			if dispatchDocumentClick(view.focusNode, event) {
				return true
			}
		} else if key.Code == 32 {
			if kos.ControlKeysStatus().Shift() {
				return view.scrollDocumentByPage(-1)
			}
			return view.scrollDocumentByPage(1)
		}
	case key.ScanCode == 0x48:
		return view.scrollDocumentBy(-1)
	case key.ScanCode == 0x50:
		return view.scrollDocumentBy(1)
	case key.ScanCode == 0x49:
		return view.scrollDocumentByPage(-1)
	case key.ScanCode == 0x51:
		return view.scrollDocumentByPage(1)
	case key.ScanCode == 0x47:
		return view.scrollDocumentTo(0)
	case key.ScanCode == 0x4F:
		return view.scrollDocumentTo(view.scrollMaxY)
	}
	return false
}

func (view *DocumentView) HandleScroll(deltaX int, deltaY int) bool {
	if view == nil {
		return false
	}
	target := view.hoverNode
	if target == nil {
		target = view.focusNode
	}
	if target != nil {
		event := DocumentEvent{
			Type:    EventScroll,
			DeltaX:  deltaX,
			DeltaY:  deltaY,
			ScrollX: 0,
			ScrollY: view.scrollY,
			View:    view,
			Node:    target,
		}
		if dispatchDocumentNodeHandler(target.OnScroll, target, event) {
			return true
		}
	}
	if deltaY != 0 {
		return view.scrollDocumentBy(deltaY)
	}
	return false
}

func (view *DocumentView) HandleMouseMove(x int, y int) bool {
	if view == nil {
		return false
	}
	if view.scrollDrag {
		return view.handleDocumentScrollbarDrag(y)
	}
	event, ok := view.documentEventFor(EventMouseMove, x, y, 0)
	if !ok {
		return view.clearHoverNode()
	}
	changed := view.setHoverNode(event.Node, event)
	if event.Node != nil && dispatchDocumentNodeHandler(event.Node.OnMouseMove, event.Node, event) {
		changed = true
	}
	return changed
}

func (view *DocumentView) HandleMouseDown(x int, y int, button MouseButton) bool {
	if view == nil {
		return false
	}
	if button == MouseLeft && view.handleDocumentScrollbarMouseDown(x, y) {
		return true
	}
	event, ok := view.documentEventFor(EventMouseDown, x, y, button)
	if !ok {
		changed := view.clearHoverNode()
		if view.setActiveNode(nil, DocumentEvent{Type: EventMouseDown, Button: button, View: view}) {
			changed = true
		}
		return changed
	}
	changed := view.setHoverNode(event.Node, event)
	if view.setActiveNode(event.Node, event) {
		changed = true
	}
	if event.Node != nil && dispatchDocumentNodeHandler(event.Node.OnMouseDown, event.Node, event) {
		changed = true
	}
	return changed
}

func (view *DocumentView) HandleMouseUp(x int, y int, button MouseButton) bool {
	if view == nil {
		return false
	}
	if view.scrollDrag {
		view.scrollDrag = false
		return true
	}
	event, ok := view.documentEventFor(EventMouseUp, x, y, button)
	changed := false
	if ok {
		changed = view.setHoverNode(event.Node, event)
		if documentNodeCanFocus(event.Node) {
			if view.setFocusNode(event.Node, event) {
				changed = true
			}
		} else if view.focusNode != nil {
			if view.setFocusNode(nil, event) {
				changed = true
			}
		}
	} else {
		event = DocumentEvent{
			Type:    EventMouseUp,
			X:       x,
			Y:       y,
			Button:  button,
			ScrollY: view.scrollY,
			View:    view,
		}
		changed = view.clearHoverNode()
	}
	if view.activeNode != nil {
		activeEvent := event
		activeEvent.Node = view.activeNode
		if dispatchDocumentNodeHandler(view.activeNode.OnMouseUp, view.activeNode, activeEvent) {
			changed = true
		}
	}
	if view.setActiveNode(nil, event) {
		changed = true
	}
	return changed
}

func (view *DocumentView) focusableNodes() []*DocumentNode {
	if view == nil || view.Document == nil {
		return nil
	}
	nodes := make([]*DocumentNode, 0, 8)
	collectDocumentFocusables(view.Document.Root, &nodes)
	return nodes
}

func collectDocumentFocusables(node *DocumentNode, out *[]*DocumentNode) {
	if node == nil {
		return
	}
	if documentNodeCanFocus(node) {
		*out = append(*out, node)
	}
	for _, child := range node.Children {
		collectDocumentFocusables(child, out)
	}
}

func (view *DocumentView) setHoverNode(node *DocumentNode, event DocumentEvent) bool {
	if view == nil || view.hoverNode == node {
		return false
	}
	changed := false
	if previous := view.hoverNode; previous != nil {
		if updated, needsLayout := previous.setHover(false); updated {
			view.markDocumentNodeStateChange(previous, needsLayout)
			changed = true
		}
		leave := event
		leave.Type = EventMouseLeave
		leave.Node = previous
		if dispatchDocumentNodeHandler(previous.OnMouseLeave, previous, leave) {
			changed = true
		}
	}
	view.hoverNode = node
	if node != nil {
		if updated, needsLayout := node.setHover(true); updated {
			view.markDocumentNodeStateChange(node, needsLayout)
			changed = true
		}
		enter := event
		enter.Type = EventMouseEnter
		enter.Node = node
		if dispatchDocumentNodeHandler(node.OnMouseEnter, node, enter) {
			changed = true
		}
	}
	return changed
}

func (view *DocumentView) clearHoverNode() bool {
	if view == nil || view.hoverNode == nil {
		return false
	}
	return view.setHoverNode(nil, DocumentEvent{Type: EventMouseLeave, ScrollY: view.scrollY, View: view})
}

func (view *DocumentView) setActiveNode(node *DocumentNode, event DocumentEvent) bool {
	if view == nil || view.activeNode == node {
		return false
	}
	changed := false
	if previous := view.activeNode; previous != nil {
		if updated, needsLayout := previous.setActive(false); updated {
			view.markDocumentNodeStateChange(previous, needsLayout)
			changed = true
		}
	}
	view.activeNode = node
	if node != nil {
		if updated, needsLayout := node.setActive(true); updated {
			view.markDocumentNodeStateChange(node, needsLayout)
			changed = true
		}
	}
	return changed
}

func (view *DocumentView) setFocusNode(node *DocumentNode, event DocumentEvent) bool {
	if view == nil || view.focusNode == node {
		return false
	}
	changed := false
	if previous := view.focusNode; previous != nil {
		if updated, needsLayout := previous.setFocus(false); updated {
			view.markDocumentNodeStateChange(previous, needsLayout)
			changed = true
		}
		blur := event
		blur.Type = EventBlur
		blur.Node = previous
		if dispatchDocumentNodeHandler(previous.OnBlur, previous, blur) {
			changed = true
		}
	}
	view.focusNode = node
	if node != nil {
		if updated, needsLayout := node.setFocus(true); updated {
			view.markDocumentNodeStateChange(node, needsLayout)
			changed = true
		}
		focus := event
		focus.Type = EventFocus
		focus.Node = node
		if dispatchDocumentNodeHandler(node.OnFocus, node, focus) {
			changed = true
		}
		if view.scrollDocumentNodeIntoView(node) {
			changed = true
		}
		if view.window != nil && view.window.scrollNodeIntoView(view) {
			changed = true
		}
	}
	return changed
}

func (view *DocumentView) documentNodeDirtyRect(node *DocumentNode) Rect {
	if view == nil || node == nil || view.Document == nil {
		return Rect{}
	}
	fragment := view.Document.FragmentForNode(node)
	if fragment == nil {
		return Rect{}
	}
	bounds := fragmentPaintBounds(fragment)
	if bounds.Empty() {
		bounds = fragment.Bounds
	}
	if bounds.Empty() {
		return Rect{}
	}
	bounds.Y -= view.scrollY
	return bounds
}

func (view *DocumentView) markDocumentNodeStateChange(node *DocumentNode, needsLayout bool) {
	if view == nil {
		return
	}
	if needsLayout {
		view.MarkLayoutDirty()
		return
	}
	view.MarkDirty()
}

func (view *DocumentView) documentEventFor(kind EventType, x int, y int, button MouseButton) (DocumentEvent, bool) {
	event := DocumentEvent{
		Type:    kind,
		X:       x,
		Y:       y,
		Button:  button,
		ScrollX: 0,
		ScrollY: view.scrollY,
		View:    view,
	}
	if view == nil || view.Document == nil {
		return event, false
	}
	viewport := view.documentViewportRect(view.effectiveStyle())
	if viewport.Empty() || !viewport.Contains(x, y) {
		return event, false
	}
	event.LocalX = x - viewport.X
	event.LocalY = y - viewport.Y
	event.DocumentX = x
	event.DocumentY = y + view.scrollY
	event.Node = view.Document.HitTest(event.DocumentX, event.DocumentY)
	return event, true
}

func (view *DocumentView) documentViewportRect(style Style) Rect {
	if view == nil {
		return Rect{}
	}
	return view.documentViewportRectIn(view.layoutRect, style)
}

func (view *DocumentView) documentViewportRectIn(rect Rect, style Style) Rect {
	viewport := contentRectFor(rect, style)
	if viewport.Empty() {
		return viewport
	}
	if !view.documentShowsScrollbar(style) {
		return viewport
	}
	scrollbar := resolveScrollbarStyle(style)
	reserve := scrollbar.width + scrollbar.padding.Left + scrollbar.padding.Right
	if reserve <= 0 {
		return viewport
	}
	viewport.Width -= reserve
	if viewport.Width < 0 {
		viewport.Width = 0
	}
	return viewport
}

func (view *DocumentView) documentShowsScrollbar(style Style) bool {
	mode := overflowModeFor(style, "y")
	switch mode {
	case OverflowScroll:
		return true
	case OverflowAuto:
		return view.scrollMaxY > 0
	default:
		return false
	}
}

func (view *DocumentView) updateDocumentScrollMetrics(viewport Rect, style Style) bool {
	if view == nil {
		return false
	}
	maxScroll := 0
	mode := overflowModeFor(style, "y")
	if (mode == OverflowScroll || mode == OverflowAuto) && view.Document != nil && !viewport.Empty() {
		extent := view.documentContentExtentHeightOnly()
		maxScroll = extent - viewport.Height
		if maxScroll < 0 {
			maxScroll = 0
		}
	}
	changed := view.scrollMaxY != maxScroll
	view.scrollMaxY = maxScroll
	if view.scrollY < 0 {
		view.scrollY = 0
		changed = true
	}
	if view.scrollY > view.scrollMaxY {
		view.scrollY = view.scrollMaxY
		changed = true
	}
	return changed
}

func (view *DocumentView) documentContentExtentHeightOnly() int {
	if view == nil || view.Document == nil {
		return 0
	}
	bounds := view.Document.ContentBounds()
	height := bounds.Y + bounds.Height - view.Document.Viewport().Y
	if height < 0 {
		height = 0
	}
	return height
}

func (view *DocumentView) documentScrollbarLayoutIn(rect Rect, style Style) (Rect, Rect, bool) {
	if view == nil || !view.documentShowsScrollbar(style) {
		return Rect{}, Rect{}, false
	}
	base := contentRectFor(rect, style)
	if base.Empty() {
		return Rect{}, Rect{}, false
	}
	scrollbar := resolveScrollbarStyle(style)
	if scrollbar.width <= 0 {
		return Rect{}, Rect{}, false
	}
	track := Rect{
		X:      base.X + base.Width - scrollbar.width - scrollbar.padding.Right,
		Y:      base.Y + scrollbar.padding.Top,
		Width:  scrollbar.width,
		Height: base.Height - scrollbar.padding.Top - scrollbar.padding.Bottom,
	}
	if track.Width <= 0 || track.Height <= 0 {
		return Rect{}, Rect{}, false
	}
	contentHeight := view.documentContentExtentHeightOnly()
	if contentHeight <= 0 {
		contentHeight = track.Height
	}
	thumbHeight := track.Height
	if contentHeight > 0 && view.scrollMaxY > 0 {
		viewport := view.documentViewportRectIn(rect, style)
		thumbHeight = track.Height * viewport.Height / contentHeight
		if thumbHeight < defaultScrollbarMinThumb {
			thumbHeight = defaultScrollbarMinThumb
		}
		if thumbHeight > track.Height {
			thumbHeight = track.Height
		}
	}
	thumbY := track.Y
	offsetRange := track.Height - thumbHeight
	if offsetRange > 0 && view.scrollMaxY > 0 {
		thumbY = track.Y + view.scrollY*offsetRange/view.scrollMaxY
	}
	thumb := Rect{
		X:      track.X,
		Y:      thumbY,
		Width:  track.Width,
		Height: thumbHeight,
	}
	return track, thumb, true
}

func (view *DocumentView) handleDocumentScrollbarMouseDown(x int, y int) bool {
	if view == nil {
		return false
	}
	track, thumb, ok := view.documentScrollbarLayoutIn(view.layoutRect, view.effectiveStyle())
	if !ok || !track.Contains(x, y) {
		return false
	}
	if thumb.Contains(x, y) {
		view.scrollDrag = true
		view.scrollDragOff = y - thumb.Y
		return true
	}
	rangeY := track.Height - thumb.Height
	if rangeY <= 0 || view.scrollMaxY <= 0 {
		return false
	}
	target := y - track.Y - thumb.Height/2
	if target < 0 {
		target = 0
	}
	if target > rangeY {
		target = rangeY
	}
	return view.scrollDocumentTo(target * view.scrollMaxY / rangeY)
}

func (view *DocumentView) handleDocumentScrollbarDrag(y int) bool {
	if view == nil || !view.scrollDrag {
		return false
	}
	track, thumb, ok := view.documentScrollbarLayoutIn(view.layoutRect, view.effectiveStyle())
	if !ok {
		return false
	}
	rangeY := track.Height - thumb.Height
	if rangeY <= 0 || view.scrollMaxY <= 0 {
		return false
	}
	target := y - view.scrollDragOff - track.Y
	if target < 0 {
		target = 0
	}
	if target > rangeY {
		target = rangeY
	}
	return view.scrollDocumentTo(target * view.scrollMaxY / rangeY)
}

func (view *DocumentView) drawDocumentScrollbar(canvas *Canvas, rect Rect, style Style) {
	if view == nil || canvas == nil {
		return
	}
	track, thumb, ok := view.documentScrollbarLayoutIn(rect, style)
	if !ok {
		return
	}
	scrollbar := resolveScrollbarStyle(style)
	radii := scrollBarRadii(scrollbar.radius)
	canvas.FillRoundedRect(track.X, track.Y, track.Width, track.Height, radii, scrollbar.track)
	canvas.FillRoundedRect(thumb.X, thumb.Y, thumb.Width, thumb.Height, radii, scrollbar.thumb)
}

func (view *DocumentView) scrollDocumentBy(deltaY int) bool {
	if view == nil || deltaY == 0 {
		return false
	}
	step := metricsForStyle(view.effectiveStyle()).height * 3
	if step < defaultFontHeight {
		step = defaultFontHeight
	}
	return view.scrollDocumentTo(view.scrollY + deltaY*step)
}

func (view *DocumentView) scrollDocumentByPage(delta int) bool {
	if view == nil || delta == 0 {
		return false
	}
	viewport := view.documentViewportRect(view.effectiveStyle())
	step := viewport.Height - defaultFontHeight
	if step < defaultFontHeight {
		step = defaultFontHeight
	}
	return view.scrollDocumentTo(view.scrollY + delta*step)
}

func (view *DocumentView) noteDocumentScrollChanged() {
	if view == nil {
		return
	}
	view.dirty = true
	if view.window == nil || view.layoutRect.Empty() {
		view.MarkDirty()
		return
	}
	style := view.effectiveStyle()
	viewport := view.documentViewportRect(style)
	if viewport.Empty() {
		view.MarkDirty()
		return
	}
	dirty := viewport
	if view.canUseScrollBlit(style, viewport) {
		exposed := scrollExposeRect(viewport, view.pendingScrollDelta())
		if !exposed.Empty() {
			dirty = exposed
		}
		view.window.markPresentRect(viewport)
	}
	if track, _, ok := view.documentScrollbarLayoutIn(view.layoutRect, style); ok {
		dirty = UnionRect(dirty, track)
	}
	view.window.Invalidate(dirty)
}

func (view *DocumentView) scrollDocumentTo(next int) bool {
	if view == nil || view.scrollMaxY <= 0 {
		if view != nil && view.scrollY != 0 {
			view.scrollY = 0
			view.noteDocumentScrollChanged()
			return true
		}
		return false
	}
	if next < 0 {
		next = 0
	}
	if next > view.scrollMaxY {
		next = view.scrollMaxY
	}
	if next == view.scrollY {
		return false
	}
	view.scrollY = next
	view.noteDocumentScrollChanged()
	return true
}

func (view *DocumentView) scrollDocumentNodeIntoView(node *DocumentNode) bool {
	if view == nil || node == nil || view.Document == nil {
		return false
	}
	viewport := view.documentViewportRect(view.effectiveStyle())
	if viewport.Empty() {
		return false
	}
	fragment := view.Document.FragmentForNode(node)
	if fragment == nil {
		return false
	}
	bounds := fragmentPaintBounds(fragment)
	if bounds.Empty() {
		bounds = fragment.Bounds
	}
	if bounds.Empty() {
		return false
	}
	start := bounds.Y - viewport.Y
	next := scrollRevealNearest(view.scrollY, viewport.Height, start, bounds.Height)
	maxScroll := view.scrollMaxY
	required := start + bounds.Height - viewport.Height
	if required > maxScroll {
		maxScroll = required
	}
	if maxScroll > view.scrollMaxY {
		view.scrollMaxY = maxScroll
	}
	return view.scrollDocumentTo(next)
}
