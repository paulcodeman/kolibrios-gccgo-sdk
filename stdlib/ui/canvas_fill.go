package ui

import "kos"

func (canvas *Canvas) Clear(color kos.Color) {
	if canvas == nil || len(canvas.data) < 2 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha < 255 {
		canvas.FillRectAlpha(0, 0, canvas.width, canvas.height, kos.Color(rgb), alpha)
		return
	}
	value := rgb | 0xFF000000
	fillSlice32(canvas.data[2:], value)
}

func (canvas *Canvas) ClearTransparent() {
	if canvas == nil || len(canvas.data) < 2 {
		return
	}
	pixels := canvas.data[2:]
	fillSlice32(pixels, 0)
}

func (canvas *Canvas) ClearRectTransparent(x int, y int, width int, height int) {
	if canvas == nil || len(canvas.data) < 2 {
		return
	}
	if !canvas.alpha {
		canvas.fillRectValue(x, y, width, height, 0xFF000000)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	rowStart := 2 + y*canvas.width + x
	if x == 0 && width == canvas.width {
		fillSlice32(canvas.data[rowStart:rowStart+width*height], 0)
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*canvas.width
		fillSlice32(canvas.data[index:index+width], 0)
	}
}

func (canvas *Canvas) FillRect(x int, y int, width int, height int, color kos.Color) {
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha < 255 {
		canvas.FillRectAlpha(x, y, width, height, kos.Color(rgb), alpha)
		return
	}
	value := rgb | 0xFF000000
	rowStart := 2 + y*canvas.width + x
	if x == 0 && width == canvas.width {
		fillSlice32(canvas.data[rowStart:rowStart+width*height], value)
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*canvas.width
		fillSlice32(canvas.data[index:index+width], value)
	}
}

func (canvas *Canvas) fillRectValue(x int, y int, width int, height int, value uint32) {
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	rowStart := 2 + y*canvas.width + x
	if x == 0 && width == canvas.width {
		fillSlice32(canvas.data[rowStart:rowStart+width*height], value)
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*canvas.width
		fillSlice32(canvas.data[index:index+width], value)
	}
}

func (canvas *Canvas) FillRoundedRect(x int, y int, width int, height int, radii CornerRadii, color kos.Color) {
	if canvas == nil || width <= 0 || height <= 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha < 255 {
		canvas.FillRoundedRectAlpha(x, y, width, height, radii, kos.Color(rgb), alpha)
		return
	}
	if !radii.Active() {
		canvas.FillRect(x, y, width, height, kos.Color(rgb))
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRect(x, y, width, height, color)
		return
	}
	colorValue := rgb
	value := colorValue | 0xFF000000
	for row := 0; row < height; row++ {
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*canvas.width + x
		middleStart := leftWidth
		middleEnd := width - rightWidth
		if middleEnd > middleStart {
			fillSlice32(canvas.data[rowStart+middleStart:rowStart+middleEnd], value)
		}
		if leftWidth > 0 {
			for col := 0; col < leftWidth; col++ {
				alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if alpha == 0 {
					continue
				}
				if alpha >= 255 {
					canvas.data[rowStart+col] = value
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, alpha)
			}
		}
		if rightWidth > 0 {
			start := width - rightWidth
			if start < leftWidth {
				start = leftWidth
			}
			for col := start; col < width; col++ {
				alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if alpha == 0 {
					continue
				}
				if alpha >= 255 {
					canvas.data[rowStart+col] = value
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, alpha)
			}
		}
	}
}

func (canvas *Canvas) StrokeRect(x int, y int, width int, height int, color kos.Color) {
	if canvas == nil || width <= 0 || height <= 0 {
		return
	}
	canvas.FillRect(x, y, width, 1, color)
	canvas.FillRect(x, y+height-1, width, 1, color)
	canvas.FillRect(x, y, 1, height, color)
	canvas.FillRect(x+width-1, y, 1, height, color)
}

func (canvas *Canvas) StrokeRectWidth(x int, y int, width int, height int, stroke int, color kos.Color) {
	if canvas == nil || width <= 0 || height <= 0 || stroke <= 0 {
		return
	}
	maxStroke := width / 2
	if height/2 < maxStroke {
		maxStroke = height / 2
	}
	if stroke > maxStroke {
		stroke = maxStroke
	}
	for i := 0; i < stroke; i++ {
		canvas.StrokeRect(x+i, y+i, width-2*i, height-2*i, color)
	}
}

func (canvas *Canvas) StrokeRoundedRectWidth(x int, y int, width int, height int, radii CornerRadii, stroke int, color kos.Color) {
	if canvas == nil || width <= 0 || height <= 0 || stroke <= 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	if !radii.Active() {
		canvas.StrokeRectWidth(x, y, width, height, stroke, kos.Color(rgb))
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.StrokeRectWidth(x, y, width, height, stroke, kos.Color(rgb))
		return
	}
	maxStroke := width / 2
	if height/2 < maxStroke {
		maxStroke = height / 2
	}
	if stroke > maxStroke {
		stroke = maxStroke
	}
	if stroke <= 0 {
		return
	}
	if stroke*2 >= width || stroke*2 >= height {
		canvas.FillRoundedRect(x, y, width, height, radii, kos.Color(rgb))
		return
	}
	innerW := width - stroke*2
	innerH := height - stroke*2
	innerRadii := CornerRadii{
		TopLeft:     radii.TopLeft - stroke,
		TopRight:    radii.TopRight - stroke,
		BottomRight: radii.BottomRight - stroke,
		BottomLeft:  radii.BottomLeft - stroke,
	}
	if innerRadii.TopLeft < 0 {
		innerRadii.TopLeft = 0
	}
	if innerRadii.TopRight < 0 {
		innerRadii.TopRight = 0
	}
	if innerRadii.BottomRight < 0 {
		innerRadii.BottomRight = 0
	}
	if innerRadii.BottomLeft < 0 {
		innerRadii.BottomLeft = 0
	}
	innerRadii = normalizeRadii(innerW, innerH, innerRadii)
	colorValue := rgb
	value := colorValue | 0xFF000000
	for row := 0; row < height; row++ {
		rowStart := 2 + (y+row)*canvas.width + x
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
				canvas.data[rowStart+col] = value
				continue
			}
			dst := canvas.data[rowStart+col]
			canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
		}
	}
}
