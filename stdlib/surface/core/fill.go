package core

func (buffer *Buffer) ClearRectTransparent(x int, y int, width int, height int) {
	if buffer == nil || len(buffer.data) < 2 {
		return
	}
	if !buffer.alpha {
		buffer.fillRectValue(x, y, width, height, 0xFF000000)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	rowStart := 2 + y*buffer.width + x
	for row := 0; row < height; row++ {
		index := rowStart + row*buffer.width
		fill32(buffer.data[index:index+width], 0)
	}
}

func (buffer *Buffer) FillRoundedRect(x int, y int, width int, height int, radii CornerRadii, color uint32) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha < 255 {
		buffer.FillRoundedRectAlpha(x, y, width, height, radii, rgb, alpha)
		return
	}
	if !radii.Active() {
		buffer.FillRect(x, y, width, height, rgb)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.FillRect(x, y, width, height, color)
		return
	}
	colorValue := rgb
	value := colorValue | 0xFF000000
	for row := 0; row < height; row++ {
		rowStart := 2 + (y+row)*buffer.width + x
		buffer.paintRoundedRowOpaqueValue(rowStart, row, width, height, radii, value)
	}
}

func (buffer *Buffer) StrokeRect(x int, y int, width int, height int, color uint32) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	buffer.FillRect(x, y, width, 1, color)
	buffer.FillRect(x, y+height-1, width, 1, color)
	buffer.FillRect(x, y, 1, height, color)
	buffer.FillRect(x+width-1, y, 1, height, color)
}

func (buffer *Buffer) StrokeRectWidth(x int, y int, width int, height int, stroke int, color uint32) {
	if buffer == nil || width <= 0 || height <= 0 || stroke <= 0 {
		return
	}
	maxStroke := width / 2
	if value := height / 2; value < maxStroke {
		maxStroke = value
	}
	if stroke > maxStroke {
		stroke = maxStroke
	}
	for i := 0; i < stroke; i++ {
		buffer.StrokeRect(x+i, y+i, width-2*i, height-2*i, color)
	}
}

func (buffer *Buffer) StrokeRoundedRectWidth(x int, y int, width int, height int, radii CornerRadii, stroke int, color uint32) {
	if buffer == nil || width <= 0 || height <= 0 || stroke <= 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	if !radii.Active() {
		buffer.StrokeRectWidth(x, y, width, height, stroke, rgb)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.StrokeRectWidth(x, y, width, height, stroke, rgb)
		return
	}
	maxStroke := width / 2
	if value := height / 2; value < maxStroke {
		maxStroke = value
	}
	if stroke > maxStroke {
		stroke = maxStroke
	}
	if stroke <= 0 {
		return
	}
	if stroke*2 >= width || stroke*2 >= height {
		buffer.FillRoundedRect(x, y, width, height, radii, rgb)
		return
	}
	innerW := width - stroke*2
	innerH := height - stroke*2
	innerRadii := CornerRadii{
		TopLeft:     maxIntValue(0, radii.TopLeft-stroke),
		TopRight:    maxIntValue(0, radii.TopRight-stroke),
		BottomRight: maxIntValue(0, radii.BottomRight-stroke),
		BottomLeft:  maxIntValue(0, radii.BottomLeft-stroke),
	}
	innerRadii = normalizeRadii(innerW, innerH, innerRadii)
	colorValue := rgb
	for row := 0; row < height; row++ {
		rowStart := 2 + (y+row)*buffer.width + x
		buffer.paintRoundedStrokeRow(rowStart, row, width, height, radii, stroke, innerW, innerH, innerRadii, colorValue, alpha)
	}
}

func (buffer *Buffer) paintRoundedStrokeRow(rowStart int, row int, width int, height int, radii CornerRadii, stroke int, innerW int, innerH int, innerRadii CornerRadii, colorValue uint32, alpha uint8) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	outerLeftWidth, outerRightWidth := cornerWidthsForRow(row, height, radii)
	outerMiddleStart := outerLeftWidth
	outerMiddleEnd := width - outerRightWidth
	innerLeftWidth := 0
	innerRightWidth := 0
	innerRow := row - stroke
	innerActive := innerW > 0 && innerH > 0 && innerRow >= 0 && innerRow < innerH
	if innerActive && innerRadii.Active() {
		innerLeftWidth, innerRightWidth = cornerWidthsForRow(innerRow, innerH, innerRadii)
	}
	if innerActive {
		leftSpanEnd := stroke + innerLeftWidth
		if leftSpanEnd > outerMiddleEnd {
			leftSpanEnd = outerMiddleEnd
		}
		buffer.paintStrokeSpan(rowStart, outerMiddleStart, leftSpanEnd, colorValue, alpha)
		rightSpanStart := width - stroke - innerRightWidth
		if rightSpanStart < outerMiddleStart {
			rightSpanStart = outerMiddleStart
		}
		buffer.paintStrokeSpan(rowStart, rightSpanStart, outerMiddleEnd, colorValue, alpha)
	} else {
		buffer.paintStrokeSpan(rowStart, outerMiddleStart, outerMiddleEnd, colorValue, alpha)
	}
	if outerLeftWidth > 0 {
		for col := 0; col < outerLeftWidth; col++ {
			buffer.paintRoundedStrokePixel(rowStart+col, col, row, width, height, radii, stroke, innerW, innerH, innerRadii, colorValue, alpha)
		}
	}
	if outerRightWidth > 0 {
		start := width - outerRightWidth
		if start < outerLeftWidth {
			start = outerLeftWidth
		}
		for col := start; col < width; col++ {
			buffer.paintRoundedStrokePixel(rowStart+col, col, row, width, height, radii, stroke, innerW, innerH, innerRadii, colorValue, alpha)
		}
	}
}

func (buffer *Buffer) paintStrokeSpan(rowStart int, start int, end int, colorValue uint32, alpha uint8) {
	if buffer == nil || end <= start || alpha == 0 {
		return
	}
	if alpha >= 255 {
		fill32(buffer.data[rowStart+start:rowStart+end], 0xFF000000|colorValue)
		return
	}
	buffer.blendRowValue(rowStart+start, end-start, colorValue, alpha)
}

func (buffer *Buffer) paintRoundedStrokePixel(index int, col int, row int, width int, height int, radii CornerRadii, stroke int, innerW int, innerH int, innerRadii CornerRadii, colorValue uint32, alpha uint8) {
	if buffer == nil {
		return
	}
	outerAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
	if outerAlpha == 0 {
		return
	}
	innerAlpha := 0
	ix := col - stroke
	iy := row - stroke
	if ix >= 0 && iy >= 0 && ix < innerW && iy < innerH {
		if innerRadii.Active() {
			innerAlpha = int(roundedPixelCoverageAlpha(ix, iy, innerW, innerH, innerRadii))
		} else {
			innerAlpha = 255
		}
	}
	coverage := int(outerAlpha) - innerAlpha
	if coverage <= 0 {
		return
	}
	effective := uint8(coverage)
	if alpha < 255 {
		effective = combineAlpha(effective, alpha)
		if effective == 0 {
			return
		}
	}
	if effective >= 255 {
		buffer.data[index] = 0xFF000000 | colorValue
		return
	}
	buffer.data[index] = buffer.blendPixel(buffer.data[index], colorValue, effective)
}

func maxIntValue(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
