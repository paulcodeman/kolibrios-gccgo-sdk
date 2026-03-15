package ui

import "sync"

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

type cornerKind uint8

const (
	cornerTopLeft cornerKind = iota
	cornerTopRight
	cornerBottomRight
	cornerBottomLeft
)

var cornerSampleOffsets = [...]float64{0.125, 0.375, 0.625, 0.875}

var (
	cornerCoverageMu    sync.Mutex
	cornerCoverageCache = map[int][]uint8{}
)

func cornerCoverageTable(radius int) []uint8 {
	if radius <= 0 {
		return nil
	}
	table, ok := cornerCoverageCache[radius]
	if ok {
		return table
	}
	size := radius * radius
	table = make([]uint8, size)
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
	if radii.BottomLeft > 0 && row >= height-radii.BottomLeft {
		if radii.BottomLeft > leftWidth {
			leftWidth = radii.BottomLeft
		}
	}
	if radii.TopRight > 0 && row < radii.TopRight {
		rightWidth = radii.TopRight
	}
	if radii.BottomRight > 0 && row >= height-radii.BottomRight {
		if radii.BottomRight > rightWidth {
			rightWidth = radii.BottomRight
		}
	}
	if leftWidth < 0 {
		leftWidth = 0
	}
	if rightWidth < 0 {
		rightWidth = 0
	}
	return leftWidth, rightWidth
}

func roundedPixelCoverage(col int, row int, width int, height int, radii CornerRadii) float64 {
	if !radii.Active() {
		return 1.0
	}
	if radii.TopLeft > 0 && col < radii.TopLeft && row < radii.TopLeft {
		return cornerCoverage(col, row, width, height, radii.TopLeft, cornerTopLeft)
	}
	if radii.TopRight > 0 && col >= width-radii.TopRight && row < radii.TopRight {
		return cornerCoverage(col, row, width, height, radii.TopRight, cornerTopRight)
	}
	if radii.BottomRight > 0 && col >= width-radii.BottomRight && row >= height-radii.BottomRight {
		return cornerCoverage(col, row, width, height, radii.BottomRight, cornerBottomRight)
	}
	if radii.BottomLeft > 0 && col < radii.BottomLeft && row >= height-radii.BottomLeft {
		return cornerCoverage(col, row, width, height, radii.BottomLeft, cornerBottomLeft)
	}
	return 1.0
}

func cornerCoverage(col int, row int, width int, height int, radius int, corner cornerKind) float64 {
	if radius <= 0 {
		return 1.0
	}
	var cx float64
	var cy float64
	switch corner {
	case cornerTopLeft:
		cx = float64(radius)
		cy = float64(radius)
	case cornerTopRight:
		cx = float64(width - radius)
		cy = float64(radius)
	case cornerBottomRight:
		cx = float64(width - radius)
		cy = float64(height - radius)
	case cornerBottomLeft:
		cx = float64(radius)
		cy = float64(height - radius)
	}
	r2 := float64(radius * radius)
	inside := 0
	for _, sx := range cornerSampleOffsets {
		for _, sy := range cornerSampleOffsets {
			px := float64(col) + sx
			py := float64(row) + sy
			dx := px - cx
			dy := py - cy
			if dx*dx+dy*dy <= r2 {
				inside++
			}
		}
	}
	return float64(inside) / float64(len(cornerSampleOffsets)*len(cornerSampleOffsets))
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
