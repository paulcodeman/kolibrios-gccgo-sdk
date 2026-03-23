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
	value := colorValue | 0xFF000000
	for row := 0; row < height; row++ {
		rowStart := 2 + (y+row)*buffer.width + x
		for col := 0; col < width; col++ {
			outerAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
			if outerAlpha == 0 {
				continue
			}
			innerAlpha := 0
			if innerW > 0 && innerH > 0 {
				ix := col - stroke
				iy := row - stroke
				if ix >= 0 && iy >= 0 && ix < innerW && iy < innerH {
					if innerRadii.Active() {
						innerAlpha = int(roundedPixelCoverageAlpha(ix, iy, innerW, innerH, innerRadii))
					} else {
						innerAlpha = 255
					}
				}
			}
			cov := int(outerAlpha) - innerAlpha
			if cov <= 0 {
				continue
			}
			effective := uint8(cov)
			if alpha < 255 {
				effective = combineAlpha(effective, alpha)
				if effective == 0 {
					continue
				}
			}
			if effective >= 255 {
				buffer.data[rowStart+col] = value
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}

func maxIntValue(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
