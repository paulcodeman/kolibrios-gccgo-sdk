package ui

// ElementRetainedLayers enables retained subtree layers for large static box
// containers so redraw falls back to a single blit instead of replaying every
// descendant item. CSS-like contain/will-change hints can opt large static
// boxes into the same path with lower thresholds.
var ElementRetainedLayers = true

const (
	elementRetainedLayerMinDescendants = 4
	elementRetainedLayerMinArea        = 16384
	elementContainLayerMinArea         = 4096
	elementWillChangeLayerMinArea      = 8192
	elementRetainedLayerMaxDirtyRects  = 4
)

func (element *Element) invalidateRetainedLayerChain() {
	for current := element; current != nil; current = current.Parent {
		current.subtreeLayerValid = false
		current.clearRetainedSubtreeDirty()
		current.subtreeLayerTreeKnown = false
		current.subtreeLayerTreeOK = false
		current.subtreeLayerTreeCount = 0
	}
}

func retainedLayerDirtyRectsMergeable(a Rect, b Rect) bool {
	if a.Empty() || b.Empty() {
		return false
	}
	if !IntersectRect(a, b).Empty() {
		return true
	}
	if a.Y == b.Y && a.Height == b.Height {
		aRight := a.X + a.Width
		bRight := b.X + b.Width
		if aRight == b.X || bRight == a.X {
			return true
		}
	}
	if a.X == b.X && a.Width == b.Width {
		aBottom := a.Y + a.Height
		bBottom := b.Y + b.Height
		if aBottom == b.Y || bBottom == a.Y {
			return true
		}
	}
	return false
}

func (element *Element) hasRetainedSubtreeDirty() bool {
	return element != nil && (element.subtreeLayerDirtyFull || element.subtreeLayerDirtyCount != 0)
}

func (element *Element) noteRetainedSubtreeDirty(rect Rect) {
	if element == nil || rect.Empty() {
		return
	}
	if element.subtreeLayerDirtyFull {
		return
	}
	for {
		merged := false
		for index := 0; index < element.subtreeLayerDirtyCount; index++ {
			existing := element.subtreeLayerDirty[index]
			if !retainedLayerDirtyRectsMergeable(existing, rect) {
				continue
			}
			rect = UnionRect(rect, existing)
			last := element.subtreeLayerDirtyCount - 1
			element.subtreeLayerDirty[index] = element.subtreeLayerDirty[last]
			element.subtreeLayerDirty[last] = Rect{}
			element.subtreeLayerDirtyCount--
			merged = true
			break
		}
		if !merged {
			break
		}
	}
	if element.subtreeLayerDirtyCount < elementRetainedLayerMaxDirtyRects {
		element.subtreeLayerDirty[element.subtreeLayerDirtyCount] = rect
		element.subtreeLayerDirtyCount++
		return
	}
	element.clearRetainedSubtreeDirty()
	element.subtreeLayerDirtyFull = true
}

func (element *Element) clearRetainedSubtreeDirty() {
	if element == nil {
		return
	}
	for index := range element.subtreeLayerDirty {
		element.subtreeLayerDirty[index] = Rect{}
	}
	element.subtreeLayerDirtyCount = 0
	element.subtreeLayerDirtyFull = false
}

func (element *Element) useRetainedSubtreeLayer(style Style) bool {
	if element == nil || !ElementRetainedLayers || FastNoCache {
		return false
	}
	if element.kind != ElementKindBox || len(element.Children) == 0 {
		return false
	}
	if attachment, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
		return false
	}
	visual := element.subtreeBounds()
	if visual.Empty() || visual.Width <= 0 || visual.Height <= 0 {
		return false
	}
	minDescendants, minArea := retainedLayerHintThresholds(style)
	if visual.Width*visual.Height < minArea {
		return false
	}
	descendants, ok := element.retainedSubtreeDescendants()
	if !ok || descendants < minDescendants {
		return false
	}
	return true
}

func retainedLayerHintThresholds(style Style) (int, int) {
	minDescendants := elementRetainedLayerMinDescendants
	minArea := elementRetainedLayerMinArea
	if styleContainsPaint(style) {
		minDescendants = 1
		if minArea > elementContainLayerMinArea {
			minArea = elementContainLayerMinArea
		}
	}
	if styleWillChangePromotesRetainedLayer(style) {
		minDescendants = 1
		if minArea > elementWillChangeLayerMinArea {
			minArea = elementWillChangeLayerMinArea
		}
	}
	return minDescendants, minArea
}

func useRetainedSubtreeLayerTiles(visual Rect) bool {
	return useRetainedLayerTiles(visual)
}

func retainedSubtreeTileCount(size int) int {
	return retainedLayerTileCount(size)
}

func (element *Element) retainedSubtreeUsesTiles() bool {
	return element != nil && len(element.subtreeLayerTiles) != 0
}

func (element *Element) retainedSubtreeTileRect(col int, row int) Rect {
	if element == nil || col < 0 || row < 0 || col >= element.subtreeLayerTileCols || row >= element.subtreeLayerTileRows {
		return Rect{}
	}
	x := col * retainedLayerTileSize
	y := row * retainedLayerTileSize
	width := retainedLayerTileSize
	height := retainedLayerTileSize
	if right := x + width; right > element.subtreeLayerWidth {
		width = element.subtreeLayerWidth - x
	}
	if bottom := y + height; bottom > element.subtreeLayerHeight {
		height = element.subtreeLayerHeight - y
	}
	if width <= 0 || height <= 0 {
		return Rect{}
	}
	return Rect{X: x, Y: y, Width: width, Height: height}
}

func (element *Element) ensureRetainedSubtreeTileBacking(width int, height int) {
	if element == nil {
		return
	}
	cols := retainedSubtreeTileCount(width)
	rows := retainedSubtreeTileCount(height)
	count := cols * rows
	if count <= 0 {
		element.subtreeLayerTiles = nil
		element.subtreeLayerTileCols = 0
		element.subtreeLayerTileRows = 0
		return
	}
	tiles := make([]*Canvas, count)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			index := row*cols + col
			rect := Rect{
				X:      col * retainedLayerTileSize,
				Y:      row * retainedLayerTileSize,
				Width:  retainedLayerTileSize,
				Height: retainedLayerTileSize,
			}
			if right := rect.X + rect.Width; right > width {
				rect.Width = width - rect.X
			}
			if bottom := rect.Y + rect.Height; bottom > height {
				rect.Height = height - rect.Y
			}
			tiles[index] = NewCanvasAlpha(rect.Width, rect.Height)
		}
	}
	element.subtreeLayer = nil
	element.subtreeLayerTiles = tiles
	element.subtreeLayerTileCols = cols
	element.subtreeLayerTileRows = rows
}

func (element *Element) retainedSubtreeDescendants() (int, bool) {
	if element == nil {
		return 0, false
	}
	if element.subtreeLayerTreeKnown {
		return element.subtreeLayerTreeCount, element.subtreeLayerTreeOK
	}
	count := 0
	okTree := true
	for _, node := range element.Children {
		child, ok := node.(*Element)
		if !ok || child == nil {
			okTree = false
			break
		}
		switch child.kind {
		case ElementKindButton, ElementKindInput, ElementKindTextarea, ElementKindTinyGL:
			okTree = false
			break
		}
		if !okTree {
			break
		}
		style := child.effectiveStyle()
		if attachment, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
			okTree = false
			break
		}
		count++
		subtreeCount, ok := child.retainedSubtreeDescendants()
		if !ok {
			okTree = false
			break
		}
		count += subtreeCount
	}
	element.subtreeLayerTreeKnown = true
	element.subtreeLayerTreeOK = okTree
	if okTree {
		element.subtreeLayerTreeCount = count
	} else {
		element.subtreeLayerTreeCount = 0
	}
	return element.subtreeLayerTreeCount, element.subtreeLayerTreeOK
}

func mergeElementLayerClip(bounds Rect, parent clipState, rect Rect, clipX bool, clipY bool) clipState {
	if !clipX && !clipY {
		return parent
	}
	base := bounds
	if parent.set {
		base = parent.rect
	}
	if clipX {
		base.X = rect.X
		base.Width = rect.Width
	}
	if clipY {
		base.Y = rect.Y
		base.Height = rect.Height
	}
	base = IntersectRect(base, bounds)
	if parent.set {
		base = IntersectRect(base, parent.rect)
	}
	return clipState{rect: base, set: true}
}

func (element *Element) drawRetainedSubtreeNode(canvas *Canvas, originX int, originY int, clip clipState) {
	if element == nil || canvas == nil || nodeHidden(element) {
		return
	}
	if clip.set {
		subtree := element.subtreeBounds()
		if subtree.Empty() {
			subtree = element.Bounds()
		}
		if !subtree.Empty() {
			subtree.X -= originX
			subtree.Y -= originY
			if IntersectRect(subtree, clip.rect).Empty() {
				return
			}
		}
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	if rect.Empty() {
		return
	}
	localRect := rect
	localRect.X -= originX
	localRect.Y -= originY
	pushed := false
	if clip.set {
		if clip.rect.Empty() {
			return
		}
		canvas.PushClip(clip.rect)
		pushed = true
	}
	if !element.tryDrawFromCache(canvas, localRect, style) {
		element.drawToRect(canvas, localRect, style)
	}
	childClip := clip
	if len(element.Children) > 0 {
		clipX, clipY := paintClipAxes(style)
		if clipX || clipY {
			canvasBounds := Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}
			childClip = mergeElementLayerClip(canvasBounds, clip, contentRectFor(localRect, style), clipX, clipY)
		}
		for _, node := range element.Children {
			child, ok := node.(*Element)
			if !ok || child == nil {
				continue
			}
			child.drawRetainedSubtreeNode(canvas, originX, originY, childClip)
		}
	}
	if pushed {
		canvas.PopClip()
	}
}

func (element *Element) ensureRetainedSubtreeLayer(style Style) (Rect, bool) {
	if element == nil || !element.useRetainedSubtreeLayer(style) {
		return Rect{}, false
	}
	visual := element.subtreeBounds()
	if visual.Empty() {
		return Rect{}, false
	}
	useTiles := useRetainedSubtreeLayerTiles(visual)
	if element.subtreeLayerWidth != visual.Width || element.subtreeLayerHeight != visual.Height ||
		(useTiles != element.retainedSubtreeUsesTiles()) ||
		(useTiles && (element.subtreeLayerTileCols != retainedSubtreeTileCount(visual.Width) || element.subtreeLayerTileRows != retainedSubtreeTileCount(visual.Height))) ||
		(!useTiles && element.subtreeLayer == nil) {
		element.subtreeLayerWidth = visual.Width
		element.subtreeLayerHeight = visual.Height
		if useTiles {
			element.ensureRetainedSubtreeTileBacking(visual.Width, visual.Height)
		} else {
			element.subtreeLayerTiles = nil
			element.subtreeLayerTileCols = 0
			element.subtreeLayerTileRows = 0
			element.subtreeLayer = NewCanvasAlpha(visual.Width, visual.Height)
		}
		element.subtreeLayerValid = false
		element.clearRetainedSubtreeDirty()
	}
	if element.subtreeLayer == nil && len(element.subtreeLayerTiles) == 0 {
		return Rect{}, false
	}
	if !element.subtreeLayerValid {
		element.redrawRetainedSubtreeLayer(style, visual)
	}
	return visual, true
}

func (element *Element) redrawRetainedSubtreeTile(visual Rect, col int, row int, localClip Rect, clipSet bool) bool {
	if element == nil {
		return false
	}
	tileRect := element.retainedSubtreeTileRect(col, row)
	if tileRect.Empty() {
		return false
	}
	index := row*element.subtreeLayerTileCols + col
	if index < 0 || index >= len(element.subtreeLayerTiles) {
		return false
	}
	tile := element.subtreeLayerTiles[index]
	if tile == nil {
		return false
	}
	if clipSet {
		localClip = IntersectRect(localClip, Rect{X: 0, Y: 0, Width: tileRect.Width, Height: tileRect.Height})
		if localClip.Empty() {
			return false
		}
		tile.ClearRectTransparent(localClip.X, localClip.Y, localClip.Width, localClip.Height)
		element.drawRetainedSubtreeNode(tile, visual.X+tileRect.X, visual.Y+tileRect.Y, clipState{rect: localClip, set: true})
		return true
	}
	tile.ClearTransparent()
	element.drawRetainedSubtreeNode(tile, visual.X+tileRect.X, visual.Y+tileRect.Y, clipState{})
	return true
}

func (element *Element) redrawRetainedSubtreeLayer(style Style, visual Rect) {
	if element == nil || visual.Empty() {
		return
	}
	if element.retainedSubtreeUsesTiles() {
		for row := 0; row < element.subtreeLayerTileRows; row++ {
			for col := 0; col < element.subtreeLayerTileCols; col++ {
				element.redrawRetainedSubtreeTile(visual, col, row, Rect{}, false)
			}
		}
		element.subtreeLayerValid = true
		element.clearRetainedSubtreeDirty()
		return
	}
	if element.subtreeLayer == nil {
		return
	}
	element.subtreeLayer.ClearTransparent()
	element.drawRetainedSubtreeNode(element.subtreeLayer, visual.X, visual.Y, clipState{})
	element.subtreeLayerValid = true
	element.clearRetainedSubtreeDirty()
}

func (element *Element) updateRetainedSubtreeLayer(style Style, visual Rect) bool {
	if element == nil || !element.subtreeLayerValid || !element.hasRetainedSubtreeDirty() {
		return false
	}
	if element.subtreeLayer == nil && !element.retainedSubtreeUsesTiles() {
		return false
	}
	dirtyFull := element.subtreeLayerDirtyFull
	dirtyCount := element.subtreeLayerDirtyCount
	dirtyRects := element.subtreeLayerDirty
	element.clearRetainedSubtreeDirty()
	if dirtyFull {
		element.redrawRetainedSubtreeLayer(style, visual)
		return true
	}
	if dirtyCount == 0 {
		return false
	}
	if element.retainedSubtreeUsesTiles() {
		for row := 0; row < element.subtreeLayerTileRows; row++ {
			for col := 0; col < element.subtreeLayerTileCols; col++ {
				tileRect := element.retainedSubtreeTileRect(col, row)
				if tileRect.Empty() {
					continue
				}
				var tileDirty Rect
				tileDirtySet := false
				for index := 0; index < dirtyCount; index++ {
					dirty := IntersectRect(dirtyRects[index], visual)
					if dirty.Empty() {
						continue
					}
					localDirty := Rect{
						X:      dirty.X - visual.X,
						Y:      dirty.Y - visual.Y,
						Width:  dirty.Width,
						Height: dirty.Height,
					}
					inter := IntersectRect(localDirty, tileRect)
					if inter.Empty() {
						continue
					}
					inter.X -= tileRect.X
					inter.Y -= tileRect.Y
					if tileDirtySet {
						tileDirty = UnionRect(tileDirty, inter)
					} else {
						tileDirty = inter
						tileDirtySet = true
					}
				}
				if !tileDirtySet {
					continue
				}
				element.redrawRetainedSubtreeTile(visual, col, row, tileDirty, true)
			}
		}
		return true
	}
	updated := false
	for index := 0; index < dirtyCount; index++ {
		dirty := IntersectRect(dirtyRects[index], visual)
		if dirty.Empty() {
			continue
		}
		localDirty := Rect{
			X:      dirty.X - visual.X,
			Y:      dirty.Y - visual.Y,
			Width:  dirty.Width,
			Height: dirty.Height,
		}
		element.subtreeLayer.ClearRectTransparent(localDirty.X, localDirty.Y, localDirty.Width, localDirty.Height)
		element.drawRetainedSubtreeNode(element.subtreeLayer, visual.X, visual.Y, clipState{rect: localDirty, set: true})
		updated = true
	}
	if !updated {
		return true
	}
	return true
}

func (element *Element) tryDrawFromRetainedSubtreeLayer(canvas *Canvas, style Style, offsetY int) bool {
	if element == nil || canvas == nil {
		return false
	}
	visual, ok := element.ensureRetainedSubtreeLayer(style)
	if !ok || (element.subtreeLayer == nil && !element.retainedSubtreeUsesTiles()) {
		return false
	}
	if element.hasRetainedSubtreeDirty() && element.subtreeLayerValid {
		if !element.updateRetainedSubtreeLayer(style, visual) {
			element.redrawRetainedSubtreeLayer(style, visual)
		}
	}
	if offsetY != 0 {
		visual.Y += offsetY
	}
	if element.retainedSubtreeUsesTiles() {
		for row := 0; row < element.subtreeLayerTileRows; row++ {
			for col := 0; col < element.subtreeLayerTileCols; col++ {
				tileRect := element.retainedSubtreeTileRect(col, row)
				if tileRect.Empty() {
					continue
				}
				dstRect := Rect{X: visual.X + tileRect.X, Y: visual.Y + tileRect.Y, Width: tileRect.Width, Height: tileRect.Height}
				if canvas.clip.set && IntersectRect(dstRect, canvas.clip.rect).Empty() {
					continue
				}
				index := row*element.subtreeLayerTileCols + col
				if index < 0 || index >= len(element.subtreeLayerTiles) {
					continue
				}
				tile := element.subtreeLayerTiles[index]
				if tile == nil {
					continue
				}
				canvas.BlitFrom(tile, Rect{X: 0, Y: 0, Width: tileRect.Width, Height: tileRect.Height}, dstRect.X, dstRect.Y)
			}
		}
		return true
	}
	canvas.BlitFrom(element.subtreeLayer, Rect{X: 0, Y: 0, Width: element.subtreeLayerWidth, Height: element.subtreeLayerHeight}, visual.X, visual.Y)
	return true
}
