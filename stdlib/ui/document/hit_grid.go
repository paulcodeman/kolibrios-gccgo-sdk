package ui

// DocumentHitGridMinItems disables grid construction for small document
// display lists where a linear reverse scan is cheaper.
var DocumentHitGridMinItems = 24

// DocumentHitGridMaxCells limits document hit-grid size. Larger documents
// fall back to the existing linear reverse scan.
var DocumentHitGridMaxCells = 16384

type fragmentHitTestGrid struct {
	cellSize int
	cols     int
	rows     int
	width    int
	height   int
	originX  int
	originY  int
	cells    [][]int
	tops     []int
}

func (grid *fragmentHitTestGrid) reset() {
	grid.cellSize = 0
	grid.cols = 0
	grid.rows = 0
	grid.width = 0
	grid.height = 0
	grid.originX = 0
	grid.originY = 0
	grid.cells = nil
	grid.tops = nil
}

func (document *Document) invalidateHitGrid() {
	if document == nil {
		return
	}
	document.hitGrid.reset()
	document.hitGridValid = false
	document.hitGridVersion = 0
}

func nextDocumentDisplayVersion(version uint32) uint32 {
	version++
	if version == 0 {
		version = 1
	}
	return version
}

func (document *Document) bumpDisplayVersion() {
	if document == nil {
		return
	}
	document.displayVersion = nextDocumentDisplayVersion(document.displayVersion)
}

func (document *Document) bumpGeometryVersion() {
	if document == nil {
		return
	}
	document.geometryVersion = nextDocumentDisplayVersion(document.geometryVersion)
}

func (document *Document) shouldUseHitGrid() bool {
	if document == nil {
		return false
	}
	count := len(document.displayList.items)
	if count == 0 {
		return false
	}
	if DocumentHitGridMinItems > 0 && count < DocumentHitGridMinItems {
		return false
	}
	return true
}

func (document *Document) hitGridBounds() Rect {
	if document == nil {
		return Rect{}
	}
	bounds := document.content
	if bounds.Empty() {
		bounds = document.viewport
	}
	return bounds
}

func (document *Document) ensureHitGrid() bool {
	if document == nil {
		return false
	}
	if !document.shouldUseHitGrid() {
		document.invalidateHitGrid()
		return false
	}
	if document.hitGridValid && document.hitGridVersion == document.geometryVersion {
		return true
	}
	bounds := document.hitGridBounds()
	if bounds.Empty() {
		document.invalidateHitGrid()
		return false
	}
	if !document.hitGrid.build(bounds, document.displayList) {
		document.invalidateHitGrid()
		return false
	}
	document.hitGridValid = true
	document.hitGridVersion = document.geometryVersion
	return true
}

func (grid *fragmentHitTestGrid) build(bounds Rect, list FragmentDisplayList) bool {
	items := list.items
	if grid == nil || bounds.Empty() || len(items) == 0 {
		if grid != nil {
			grid.reset()
		}
		return false
	}
	size := hitTestCellSize
	if size <= 0 {
		size = 1
	}
	cols := (bounds.Width + size - 1) / size
	rows := (bounds.Height + size - 1) / size
	if cols <= 0 || rows <= 0 {
		grid.reset()
		return false
	}
	total := cols * rows
	if DocumentHitGridMaxCells > 0 && total > DocumentHitGridMaxCells {
		grid.reset()
		return false
	}
	if cap(grid.cells) < total {
		grid.cells = make([][]int, total)
	} else {
		grid.cells = grid.cells[:total]
	}
	if cap(grid.tops) < total {
		grid.tops = make([]int, total)
	} else {
		grid.tops = grid.tops[:total]
	}
	for i := range grid.cells {
		grid.cells[i] = grid.cells[i][:0]
		grid.tops[i] = -1
	}
	maxX := bounds.X + bounds.Width - 1
	maxY := bounds.Y + bounds.Height - 1
	for index, item := range items {
		rect := item.Bounds
		if item.ClipSet {
			rect = IntersectRect(rect, item.Clip)
		}
		if rect.Empty() {
			continue
		}
		x0 := rect.X
		y0 := rect.Y
		x1 := rect.X + rect.Width - 1
		y1 := rect.Y + rect.Height - 1
		if x1 < bounds.X || y1 < bounds.Y || x0 > maxX || y0 > maxY {
			continue
		}
		if x0 < bounds.X {
			x0 = bounds.X
		}
		if y0 < bounds.Y {
			y0 = bounds.Y
		}
		if x1 > maxX {
			x1 = maxX
		}
		if y1 > maxY {
			y1 = maxY
		}
		startCol := (x0 - bounds.X) / size
		endCol := (x1 - bounds.X) / size
		startRow := (y0 - bounds.Y) / size
		endRow := (y1 - bounds.Y) / size
		if startCol < 0 {
			startCol = 0
		}
		if startRow < 0 {
			startRow = 0
		}
		if endCol >= cols {
			endCol = cols - 1
		}
		if endRow >= rows {
			endRow = rows - 1
		}
		for row := startRow; row <= endRow; row++ {
			base := row * cols
			for col := startCol; col <= endCol; col++ {
				cell := base + col
				grid.cells[cell] = append(grid.cells[cell], index)
				grid.tops[cell] = index
			}
		}
	}
	grid.cellSize = size
	grid.cols = cols
	grid.rows = rows
	grid.width = bounds.Width
	grid.height = bounds.Height
	grid.originX = bounds.X
	grid.originY = bounds.Y
	return true
}

func (grid *fragmentHitTestGrid) find(x int, y int, list FragmentDisplayList) (*DocumentNode, bool) {
	items := list.items
	if grid == nil || grid.cols == 0 || grid.rows == 0 || len(grid.cells) == 0 {
		return nil, false
	}
	localX := x - grid.originX
	localY := y - grid.originY
	if localX < 0 || localY < 0 || localX >= grid.width || localY >= grid.height {
		return nil, true
	}
	col := localX / grid.cellSize
	row := localY / grid.cellSize
	if col < 0 || col >= grid.cols || row < 0 || row >= grid.rows {
		return nil, true
	}
	cell := grid.cells[row*grid.cols+col]
	top := grid.tops[row*grid.cols+col]
	if top >= 0 && top < len(items) {
		item := items[top]
		hit := item.Bounds
		if item.ClipSet {
			hit = IntersectRect(hit, item.Clip)
		}
		if hit.Contains(x, y) {
			return item.Fragment.Node, true
		}
	}
	for i := len(cell) - 1; i >= 0; i-- {
		index := cell[i]
		if index < 0 || index >= len(items) {
			continue
		}
		item := items[index]
		hit := item.Bounds
		if item.ClipSet {
			hit = IntersectRect(hit, item.Clip)
		}
		if hit.Contains(x, y) {
			return item.Fragment.Node, true
		}
	}
	return nil, true
}
