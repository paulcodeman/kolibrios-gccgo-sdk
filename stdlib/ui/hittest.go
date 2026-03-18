package ui

const hitTestCellSize = 32

type hitTestGrid struct {
	cellSize int
	cols     int
	rows     int
	width    int
	height   int
	cells    [][]int
	tops     []int
	offsetY  int
}

func (grid *hitTestGrid) reset() {
	grid.cols = 0
	grid.rows = 0
	grid.width = 0
	grid.height = 0
	grid.cells = nil
	grid.tops = nil
	grid.offsetY = 0
}

func (grid *hitTestGrid) build(client Rect, list DisplayList) {
	items := list.Items()
	offsetY := list.ScrollOffset()
	if grid == nil || client.Empty() || len(items) == 0 {
		if grid != nil {
			grid.reset()
		}
		return
	}
	size := hitTestCellSize
	if size <= 0 {
		size = 1
	}
	cols := (client.Width + size - 1) / size
	rows := (client.Height + size - 1) / size
	if cols <= 0 || rows <= 0 {
		grid.reset()
		return
	}
	total := cols * rows
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
	width := client.Width
	height := client.Height
	for index, item := range items {
		rect := item.paint
		if offsetY != 0 {
			rect.Y += offsetY
		}
		if rect.Empty() {
			continue
		}
		x0 := rect.X
		y0 := rect.Y
		x1 := rect.X + rect.Width - 1
		y1 := rect.Y + rect.Height - 1
		if x1 < 0 || y1 < 0 || x0 >= width || y0 >= height {
			continue
		}
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 >= width {
			x1 = width - 1
		}
		if y1 >= height {
			y1 = height - 1
		}
		startCol := x0 / size
		endCol := x1 / size
		startRow := y0 / size
		endRow := y1 / size
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
	grid.width = width
	grid.height = height
	grid.offsetY = offsetY
}

func (grid *hitTestGrid) find(x int, y int, list DisplayList) (Node, bool) {
	items := list.Items()
	if grid == nil || grid.cols == 0 || grid.rows == 0 || len(grid.cells) == 0 {
		return nil, false
	}
	if x < 0 || y < 0 || x >= grid.width || y >= grid.height {
		return nil, true
	}
	col := x / grid.cellSize
	row := y / grid.cellSize
	if col < 0 || col >= grid.cols || row < 0 || row >= grid.rows {
		return nil, true
	}
	cell := grid.cells[row*grid.cols+col]
	top := grid.tops[row*grid.cols+col]
	if top >= 0 && top < len(items) {
		item := items[top]
		paint := item.paint
		if list.scrollOffset != 0 {
			paint.Y += list.scrollOffset
		}
		if paint.Contains(x, y) {
			return item.node, true
		}
	}
	for i := len(cell) - 1; i >= 0; i-- {
		index := cell[i]
		if index < 0 || index >= len(items) {
			continue
		}
		item := items[index]
		paint := item.paint
		if list.scrollOffset != 0 {
			paint.Y += list.scrollOffset
		}
		if paint.Contains(x, y) {
			return item.node, true
		}
	}
	return nil, true
}
