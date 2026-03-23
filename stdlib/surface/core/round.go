package core

import "sync"

type CornerRadii struct {
	TopLeft     int
	TopRight    int
	BottomRight int
	BottomLeft  int
}

func (radii CornerRadii) Active() bool {
	return radii.TopLeft != 0 ||
		radii.TopRight != 0 ||
		radii.BottomRight != 0 ||
		radii.BottomLeft != 0
}

var cornerSampleOffsets = [...]float64{0.125, 0.375, 0.625, 0.875}

var (
	cornerCoverageMu    sync.Mutex
	cornerCoverageCache = map[int][]uint8{}

	roundedShapeMu    sync.Mutex
	roundedShapeCache = map[roundedShapeKey]*roundedShapeInfo{}
)

type roundedShapeKey struct {
	width       int
	height      int
	topLeft     int
	topRight    int
	bottomRight int
	bottomLeft  int
}

type roundedRowInfo struct {
	leftWidth  int
	rightWidth int
	rightStart int
	leftAlpha  []uint8
	rightAlpha []uint8
}

type roundedShapeInfo struct {
	rows []roundedRowInfo
}

func roundedRowCoverageAlpha(row *roundedRowInfo, col int) uint8 {
	if row == nil {
		return 255
	}
	if col < row.leftWidth {
		return row.leftAlpha[col]
	}
	if col >= row.rightStart && col-row.rightStart < len(row.rightAlpha) {
		return row.rightAlpha[col-row.rightStart]
	}
	return 255
}

func normalizeRadii(width int, height int, radii CornerRadii) CornerRadii {
	if !radii.Active() || width <= 0 || height <= 0 {
		return CornerRadii{}
	}
	topLeft := radii.TopLeft
	topRight := radii.TopRight
	bottomRight := radii.BottomRight
	bottomLeft := radii.BottomLeft
	if topLeft < 0 {
		topLeft = 0
	}
	if topRight < 0 {
		topRight = 0
	}
	if bottomRight < 0 {
		bottomRight = 0
	}
	if bottomLeft < 0 {
		bottomLeft = 0
	}
	if topLeft == 0 && topRight == 0 && bottomRight == 0 && bottomLeft == 0 {
		return CornerRadii{}
	}
	scale := 1.0
	if sum := topLeft + topRight; sum > 0 && sum > width {
		scale = minFloat(scale, float64(width)/float64(sum))
	}
	if sum := bottomLeft + bottomRight; sum > 0 && sum > width {
		scale = minFloat(scale, float64(width)/float64(sum))
	}
	if sum := topLeft + bottomLeft; sum > 0 && sum > height {
		scale = minFloat(scale, float64(height)/float64(sum))
	}
	if sum := topRight + bottomRight; sum > 0 && sum > height {
		scale = minFloat(scale, float64(height)/float64(sum))
	}
	if scale < 1.0 {
		topLeft = int(float64(topLeft) * scale)
		topRight = int(float64(topRight) * scale)
		bottomRight = int(float64(bottomRight) * scale)
		bottomLeft = int(float64(bottomLeft) * scale)
	}
	maxRadius := minInt(width, height) / 2
	if topLeft > maxRadius {
		topLeft = maxRadius
	}
	if topRight > maxRadius {
		topRight = maxRadius
	}
	if bottomRight > maxRadius {
		bottomRight = maxRadius
	}
	if bottomLeft > maxRadius {
		bottomLeft = maxRadius
	}
	if topLeft == 0 && topRight == 0 && bottomRight == 0 && bottomLeft == 0 {
		return CornerRadii{}
	}
	return CornerRadii{
		TopLeft:     topLeft,
		TopRight:    topRight,
		BottomRight: bottomRight,
		BottomLeft:  bottomLeft,
	}
}

func cornerCoverageTable(radius int) []uint8 {
	if radius <= 0 {
		return nil
	}
	cornerCoverageMu.Lock()
	defer cornerCoverageMu.Unlock()
	if table, ok := cornerCoverageCache[radius]; ok {
		return table
	}
	size := radius * radius
	table := make([]uint8, size)
	r := float64(radius)
	r2 := r * r
	samples := len(cornerSampleOffsets) * len(cornerSampleOffsets)
	for row := 0; row < radius; row++ {
		for col := 0; col < radius; col++ {
			inside := 0
			for _, sx := range cornerSampleOffsets {
				for _, sy := range cornerSampleOffsets {
					dx := float64(col) + sx - r
					dy := float64(row) + sy - r
					if dx*dx+dy*dy <= r2 {
						inside++
					}
				}
			}
			alpha := (inside*255 + samples/2) / samples
			if alpha < 0 {
				alpha = 0
			} else if alpha > 255 {
				alpha = 255
			}
			table[row*radius+col] = uint8(alpha)
		}
	}
	cornerCoverageCache[radius] = table
	return table
}

func cornerCoverageAlpha(radius int, col int, row int) uint8 {
	if radius <= 0 {
		return 255
	}
	if col < 0 || row < 0 || col >= radius || row >= radius {
		return 255
	}
	table := cornerCoverageTable(radius)
	return table[row*radius+col]
}

func roundedPixelCoverageAlpha(col int, row int, width int, height int, radii CornerRadii) uint8 {
	if !radii.Active() {
		return 255
	}
	if radii.TopLeft > 0 && col < radii.TopLeft && row < radii.TopLeft {
		return cornerCoverageAlpha(radii.TopLeft, col, row)
	}
	if radii.TopRight > 0 && col >= width-radii.TopRight && row < radii.TopRight {
		return cornerCoverageAlpha(radii.TopRight, width-1-col, row)
	}
	if radii.BottomRight > 0 && col >= width-radii.BottomRight && row >= height-radii.BottomRight {
		return cornerCoverageAlpha(radii.BottomRight, width-1-col, height-1-row)
	}
	if radii.BottomLeft > 0 && col < radii.BottomLeft && row >= height-radii.BottomLeft {
		return cornerCoverageAlpha(radii.BottomLeft, col, height-1-row)
	}
	return 255
}

func cornerWidthsForRow(row int, height int, radii CornerRadii) (int, int) {
	leftWidth := 0
	rightWidth := 0
	if radii.TopLeft > 0 && row < radii.TopLeft {
		leftWidth = radii.TopLeft
	}
	if radii.BottomLeft > 0 && row >= height-radii.BottomLeft && radii.BottomLeft > leftWidth {
		leftWidth = radii.BottomLeft
	}
	if radii.TopRight > 0 && row < radii.TopRight {
		rightWidth = radii.TopRight
	}
	if radii.BottomRight > 0 && row >= height-radii.BottomRight && radii.BottomRight > rightWidth {
		rightWidth = radii.BottomRight
	}
	return leftWidth, rightWidth
}

func roundedShapeRows(width int, height int, radii CornerRadii) *roundedShapeInfo {
	if width <= 0 || height <= 0 || !radii.Active() {
		return nil
	}
	key := roundedShapeKey{
		width:       width,
		height:      height,
		topLeft:     radii.TopLeft,
		topRight:    radii.TopRight,
		bottomRight: radii.BottomRight,
		bottomLeft:  radii.BottomLeft,
	}
	roundedShapeMu.Lock()
	rows := roundedShapeCache[key]
	roundedShapeMu.Unlock()
	if rows != nil {
		return rows
	}
	rows = buildRoundedShapeInfo(width, height, radii)
	roundedShapeMu.Lock()
	if existing := roundedShapeCache[key]; existing != nil {
		roundedShapeMu.Unlock()
		return existing
	}
	roundedShapeCache[key] = rows
	roundedShapeMu.Unlock()
	return rows
}

func buildRoundedShapeInfo(width int, height int, radii CornerRadii) *roundedShapeInfo {
	info := &roundedShapeInfo{
		rows: make([]roundedRowInfo, height),
	}
	for row := 0; row < height; row++ {
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowInfo := roundedRowInfo{
			leftWidth:  leftWidth,
			rightWidth: rightWidth,
			rightStart: width - rightWidth,
		}
		if leftWidth > 0 {
			rowInfo.leftAlpha = make([]uint8, leftWidth)
			for col := 0; col < leftWidth; col++ {
				rowInfo.leftAlpha[col] = roundedPixelCoverageAlpha(col, row, width, height, radii)
			}
		}
		if rightWidth > 0 {
			rowInfo.rightAlpha = make([]uint8, rightWidth)
			for index := 0; index < rightWidth; index++ {
				col := rowInfo.rightStart + index
				rowInfo.rightAlpha[index] = roundedPixelCoverageAlpha(col, row, width, height, radii)
			}
		}
		info.rows[row] = rowInfo
	}
	return info
}

func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
