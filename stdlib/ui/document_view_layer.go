package ui

// DocumentViewRetainedLayer enables drawing DocumentView into its own retained
// offscreen surface before compositing it into the window canvas.
var DocumentViewRetainedLayer = true

func (view *DocumentView) useRetainedLayer(style Style) bool {
	if view == nil || !DocumentViewRetainedLayer || FastNoCache {
		return false
	}
	if view.Document == nil {
		return false
	}
	if display, ok := resolveDisplay(style.display); ok && display == DisplayNone {
		return false
	}
	return true
}

func (view *DocumentView) retainedLayerVisual(style Style, rect Rect) (Rect, Rect) {
	visual := visualBoundsForStyle(rect, style, false)
	if visual.Empty() {
		return Rect{}, Rect{}
	}
	localRect := Rect{
		X:      rect.X - visual.X,
		Y:      rect.Y - visual.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
	return visual, localRect
}

func (view *DocumentView) retainedLayerKey(style Style, localRect Rect, visual Rect) (styleVisualKey, int, int, int, int) {
	return visualKeyFor(style), visual.Width, visual.Height, localRect.X, localRect.Y
}

func (view *DocumentView) retainedLayerUsesTiles() bool {
	return view != nil && len(view.layerTiles) != 0
}

func (view *DocumentView) retainedLayerTileRect(col int, row int) Rect {
	if view == nil || col < 0 || row < 0 || col >= view.layerTileCols || row >= view.layerTileRows {
		return Rect{}
	}
	x := col * retainedLayerTileSize
	y := row * retainedLayerTileSize
	width := retainedLayerTileSize
	height := retainedLayerTileSize
	if right := x + width; right > view.layerWidth {
		width = view.layerWidth - x
	}
	if bottom := y + height; bottom > view.layerHeight {
		height = view.layerHeight - y
	}
	if width <= 0 || height <= 0 {
		return Rect{}
	}
	return Rect{X: x, Y: y, Width: width, Height: height}
}

func (view *DocumentView) ensureRetainedLayerTileBacking(width int, height int) {
	if view == nil {
		return
	}
	cols := retainedLayerTileCount(width)
	rows := retainedLayerTileCount(height)
	count := cols * rows
	if count <= 0 {
		view.layerTiles = nil
		view.layerTileCols = 0
		view.layerTileRows = 0
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
	view.layerCanvas = nil
	view.layerTiles = tiles
	view.layerTileCols = cols
	view.layerTileRows = rows
}

func (view *DocumentView) ensureRetainedLayer(style Style) (Rect, Rect, bool) {
	if view == nil || view.layoutRect.Empty() {
		return Rect{}, Rect{}, false
	}
	visual, localRect := view.retainedLayerVisual(style, view.layoutRect)
	if visual.Empty() || localRect.Empty() {
		return visual, localRect, false
	}
	key, width, height, offsetX, offsetY := view.retainedLayerKey(style, localRect, visual)
	if width <= 0 || height <= 0 {
		return visual, localRect, false
	}
	useTiles := useRetainedLayerTiles(visual)
	if view.layerWidth != width || view.layerHeight != height ||
		(useTiles != view.retainedLayerUsesTiles()) ||
		(useTiles && (view.layerTileCols != retainedLayerTileCount(width) || view.layerTileRows != retainedLayerTileCount(height))) ||
		(!useTiles && view.layerCanvas == nil) {
		if useTiles {
			view.ensureRetainedLayerTileBacking(width, height)
		} else {
			view.layerTiles = nil
			view.layerTileCols = 0
			view.layerTileRows = 0
			view.layerCanvas = NewCanvasAlpha(width, height)
		}
		view.layerValid = false
	}
	if view.layerWidth != width || view.layerHeight != height ||
		view.layerOffsetX != offsetX || view.layerOffsetY != offsetY ||
		!styleVisualKeyEqual(view.layerVisualKey, key) {
		view.layerWidth = width
		view.layerHeight = height
		view.layerOffsetX = offsetX
		view.layerOffsetY = offsetY
		view.layerVisualKey = key
		view.layerValid = false
	}
	if view.layerCanvas == nil && !view.retainedLayerUsesTiles() {
		return visual, localRect, false
	}
	if !view.layerValid {
		view.redrawRetainedLayer(style, visual, localRect)
	}
	return visual, localRect, view.layerValid
}

func (view *DocumentView) redrawRetainedLayerTile(style Style, visual Rect, localRect Rect, col int, row int, clip Rect, clipSet bool) bool {
	if view == nil || visual.Empty() || localRect.Empty() {
		return false
	}
	tileRect := view.retainedLayerTileRect(col, row)
	if tileRect.Empty() {
		return false
	}
	index := row*view.layerTileCols + col
	if index < 0 || index >= len(view.layerTiles) {
		return false
	}
	tile := view.layerTiles[index]
	if tile == nil {
		return false
	}
	tileLocalRect := Rect{
		X:      localRect.X - tileRect.X,
		Y:      localRect.Y - tileRect.Y,
		Width:  localRect.Width,
		Height: localRect.Height,
	}
	if clipSet {
		clip = IntersectRect(clip, Rect{X: 0, Y: 0, Width: tileRect.Width, Height: tileRect.Height})
		if clip.Empty() {
			return false
		}
		tile.ClearRectTransparent(clip.X, clip.Y, clip.Width, clip.Height)
		tile.PushClip(clip)
	} else {
		tile.ClearTransparent()
	}
	drawStyledBox(tile, tileLocalRect, style, tileLocalRect, nil)
	if view.Document == nil {
		if clipSet {
			tile.PopClip()
		}
		return true
	}
	viewport := view.documentViewportRectIn(tileLocalRect, style)
	if !viewport.Empty() {
		tile.PushClip(viewport)
		view.Document.PaintOffset(tile, -visual.X-tileRect.X, -visual.Y-tileRect.Y-view.scrollY)
		tile.PopClip()
	}
	view.drawDocumentScrollbar(tile, tileLocalRect, style)
	if clipSet {
		tile.PopClip()
	}
	return true
}

func (view *DocumentView) redrawRetainedLayer(style Style, visual Rect, localRect Rect) {
	if view == nil || visual.Empty() || localRect.Empty() {
		return
	}
	if view.retainedLayerUsesTiles() {
		for row := 0; row < view.layerTileRows; row++ {
			for col := 0; col < view.layerTileCols; col++ {
				view.redrawRetainedLayerTile(style, visual, localRect, col, row, Rect{}, false)
			}
		}
		view.drawnScrollY = view.scrollY
		view.layerValid = true
		return
	}
	if view.layerCanvas == nil {
		return
	}
	view.layerCanvas.ClearTransparent()
	drawStyledBox(view.layerCanvas, localRect, style, localRect, nil)
	if view.Document != nil {
		viewport := view.documentViewportRectIn(localRect, style)
		if !viewport.Empty() {
			view.layerCanvas.PushClip(viewport)
			view.Document.PaintOffset(view.layerCanvas, -visual.X, -visual.Y-view.scrollY)
			view.layerCanvas.PopClip()
		}
		view.drawDocumentScrollbar(view.layerCanvas, localRect, style)
	}
	view.drawnScrollY = view.scrollY
	view.layerValid = true
}

func (view *DocumentView) updateRetainedLayerForScroll(style Style, visual Rect, localRect Rect) bool {
	if view == nil || !view.layerValid || view.Document == nil {
		return false
	}
	viewport := view.documentViewportRectIn(localRect, style)
	if viewport.Empty() {
		return false
	}
	if !view.canUseScrollBlit(style, viewport) {
		return false
	}
	delta := view.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	if view.retainedLayerUsesTiles() {
		exposed := scrollExposeRect(viewport, delta)
		track, _, trackOK := view.documentScrollbarLayoutIn(localRect, style)
		for row := 0; row < view.layerTileRows; row++ {
			for col := 0; col < view.layerTileCols; col++ {
				tileRect := view.retainedLayerTileRect(col, row)
				if tileRect.Empty() {
					continue
				}
				index := row*view.layerTileCols + col
				if index < 0 || index >= len(view.layerTiles) {
					continue
				}
				tile := view.layerTiles[index]
				if tile == nil {
					continue
				}
				viewportPart := IntersectRect(viewport, tileRect)
				if !viewportPart.Empty() {
					localViewport := Rect{
						X:      viewportPart.X - tileRect.X,
						Y:      viewportPart.Y - tileRect.Y,
						Width:  viewportPart.Width,
						Height: viewportPart.Height,
					}
					tile.ScrollRectY(localViewport, -delta)
				}
				redrawSet := false
				redraw := Rect{}
				if part := IntersectRect(exposed, tileRect); !part.Empty() {
					redraw = Rect{
						X:      part.X - tileRect.X,
						Y:      part.Y - tileRect.Y,
						Width:  part.Width,
						Height: part.Height,
					}
					redrawSet = true
				}
				if trackOK {
					if part := IntersectRect(track, tileRect); !part.Empty() {
						part = Rect{
							X:      part.X - tileRect.X,
							Y:      part.Y - tileRect.Y,
							Width:  part.Width,
							Height: part.Height,
						}
						if redrawSet {
							redraw = UnionRect(redraw, part)
						} else {
							redraw = part
							redrawSet = true
						}
					}
				}
				if redrawSet {
					view.redrawRetainedLayerTile(style, visual, localRect, col, row, redraw, true)
				}
			}
		}
		view.drawnScrollY = view.scrollY
		return true
	}
	if view.layerCanvas == nil {
		return false
	}
	view.layerCanvas.ScrollRectY(viewport, -delta)
	exposed := scrollExposeRect(viewport, view.pendingScrollDelta())
	if !exposed.Empty() {
		view.layerCanvas.PushClip(exposed)
		view.Document.PaintOffset(view.layerCanvas, -visual.X, -visual.Y-view.scrollY)
		view.layerCanvas.PopClip()
	}
	view.drawDocumentScrollbar(view.layerCanvas, localRect, style)
	view.drawnScrollY = view.scrollY
	return true
}

func (view *DocumentView) drawRetainedLayer(canvas *Canvas, style Style, offsetY int) bool {
	if view == nil || canvas == nil || !view.useRetainedLayer(style) {
		return false
	}
	visual, localRect, ok := view.ensureRetainedLayer(style)
	if !ok || (view.layerCanvas == nil && !view.retainedLayerUsesTiles()) {
		return false
	}
	if view.pendingScrollDelta() != 0 && view.layerValid {
		if !view.updateRetainedLayerForScroll(style, visual, localRect) {
			view.redrawRetainedLayer(style, visual, localRect)
		}
	}
	targetVisual := visual
	if offsetY != 0 {
		targetVisual.Y += offsetY
	}
	if view.retainedLayerUsesTiles() {
		for row := 0; row < view.layerTileRows; row++ {
			for col := 0; col < view.layerTileCols; col++ {
				tileRect := view.retainedLayerTileRect(col, row)
				if tileRect.Empty() {
					continue
				}
				dstRect := Rect{
					X:      targetVisual.X + tileRect.X,
					Y:      targetVisual.Y + tileRect.Y,
					Width:  tileRect.Width,
					Height: tileRect.Height,
				}
				if canvas.clip.set && IntersectRect(dstRect, canvas.clip.rect).Empty() {
					continue
				}
				index := row*view.layerTileCols + col
				if index < 0 || index >= len(view.layerTiles) {
					continue
				}
				tile := view.layerTiles[index]
				if tile == nil {
					continue
				}
				canvas.BlitFrom(tile, Rect{X: 0, Y: 0, Width: tileRect.Width, Height: tileRect.Height}, dstRect.X, dstRect.Y)
			}
		}
		return true
	}
	canvas.BlitFrom(view.layerCanvas, Rect{X: 0, Y: 0, Width: view.layerWidth, Height: view.layerHeight}, targetVisual.X, targetVisual.Y)
	return true
}
