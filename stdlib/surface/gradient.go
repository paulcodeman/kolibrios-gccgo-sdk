package surface

func (buffer *Buffer) FillRectGradient(x int, y int, width int, height int, gradient Gradient) {
	if buffer == nil {
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	if width == 1 && height == 1 {
		buffer.FillRect(x, y, width, height, gradient.From)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		buffer.FillRectGradientAlpha(x, y, width, height, gradient, 255)
		return
	}
	fromR, fromG, fromB := colorToRGB(gradient.From)
	toR, toG, toB := colorToRGB(gradient.To)
	if gradient.Direction == GradientHorizontal {
		den := width - 1
		if den < 1 {
			den = 1
		}
		rowColors := make([]uint32, width)
		for col := 0; col < width; col++ {
			r := (fromR*(den-col) + toR*col) / den
			g := (fromG*(den-col) + toG*col) / den
			b := (fromB*(den-col) + toB*col) / den
			rowColors[col] = 0xFF000000 | uint32(r<<16|g<<8|b)
		}
		for row := 0; row < height; row++ {
			rowStart := 2 + (y+row)*buffer.width + x
			copy(buffer.data[rowStart:rowStart+width], rowColors)
		}
		return
	}
	den := height - 1
	if den < 1 {
		den = 1
	}
	for row := 0; row < height; row++ {
		r := (fromR*(den-row) + toR*row) / den
		g := (fromG*(den-row) + toG*row) / den
		b := (fromB*(den-row) + toB*row) / den
		value := 0xFF000000 | uint32(r<<16|g<<8|b)
		rowStart := 2 + (y+row)*buffer.width + x
		fill32(buffer.data[rowStart:rowStart+width], value)
	}
}

func (buffer *Buffer) FillRoundedRectGradient(x int, y int, width int, height int, radii CornerRadii, gradient Gradient) {
	if buffer == nil {
		return
	}
	if !radii.Active() {
		buffer.FillRectGradient(x, y, width, height, gradient)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.FillRectGradient(x, y, width, height, gradient)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		buffer.FillRoundedRectGradientAlpha(x, y, width, height, radii, gradient, 255)
		return
	}
	fromR, fromG, fromB := colorToRGB(gradient.From)
	toR, toG, toB := colorToRGB(gradient.To)
	if gradient.Direction == GradientHorizontal {
		den := width - 1
		if den < 1 {
			den = 1
		}
		rowColors := make([]uint32, width)
		for col := 0; col < width; col++ {
			r := (fromR*(den-col) + toR*col) / den
			g := (fromG*(den-col) + toG*col) / den
			b := (fromB*(den-col) + toB*col) / den
			rowColors[col] = 0xFF000000 | uint32(r<<16|g<<8|b)
		}
		for row := 0; row < height; row++ {
			leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
			rowStart := 2 + (y+row)*buffer.width + x
			middleStart := leftWidth
			middleEnd := width - rightWidth
			if middleEnd > middleStart {
				copy(buffer.data[rowStart+middleStart:rowStart+middleEnd], rowColors[middleStart:middleEnd])
			}
			if leftWidth > 0 {
				for col := 0; col < leftWidth; col++ {
					alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if alpha == 0 {
						continue
					}
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						buffer.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
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
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						buffer.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
				}
			}
		}
		return
	}
	den := height - 1
	if den < 1 {
		den = 1
	}
	for row := 0; row < height; row++ {
		r := (fromR*(den-row) + toR*row) / den
		g := (fromG*(den-row) + toG*row) / den
		b := (fromB*(den-row) + toB*row) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		value := 0xFF000000 | colorValue
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*buffer.width + x
		middleStart := leftWidth
		middleEnd := width - rightWidth
		if middleEnd > middleStart {
			fill32(buffer.data[rowStart+middleStart:rowStart+middleEnd], value)
		}
		if leftWidth > 0 {
			for col := 0; col < leftWidth; col++ {
				alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if alpha == 0 {
					continue
				}
				if alpha >= 255 {
					buffer.data[rowStart+col] = value
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
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
					buffer.data[rowStart+col] = value
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
			}
		}
	}
}

func (buffer *Buffer) FillRectGradientAlpha(x int, y int, width int, height int, gradient Gradient, alpha uint8) {
	if buffer == nil || alpha == 0 {
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		buffer.FillRectGradient(x, y, width, height, gradient)
		return
	}
	fromR := int((fromValue >> 16) & 0xFF)
	fromG := int((fromValue >> 8) & 0xFF)
	fromB := int(fromValue & 0xFF)
	toR := int((toValue >> 16) & 0xFF)
	toG := int((toValue >> 8) & 0xFF)
	toB := int(toValue & 0xFF)
	if gradient.Direction == GradientHorizontal {
		den := width - 1
		if den < 1 {
			den = 1
		}
		rowColors := make([]uint32, width)
		rowAlphas := make([]uint8, width)
		for col := 0; col < width; col++ {
			r := (fromR*(den-col) + toR*col) / den
			g := (fromG*(den-col) + toG*col) / den
			b := (fromB*(den-col) + toB*col) / den
			rowColors[col] = uint32(r<<16 | g<<8 | b)
			a := (int(fromAlpha)*(den-col) + int(toAlpha)*col) / den
			rowAlphas[col] = combineAlpha(alpha, uint8(a))
		}
		for row := 0; row < height; row++ {
			rowStart := 2 + (y+row)*buffer.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], rowColors[col], effective)
			}
		}
		return
	}
	den := height - 1
	if den < 1 {
		den = 1
	}
	for row := 0; row < height; row++ {
		r := (fromR*(den-row) + toR*row) / den
		g := (fromG*(den-row) + toG*row) / den
		b := (fromB*(den-row) + toB*row) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		a := (int(fromAlpha)*(den-row) + int(toAlpha)*row) / den
		effective := combineAlpha(alpha, uint8(a))
		if effective == 0 {
			continue
		}
		rowStart := 2 + (y+row)*buffer.width + x
		for col := 0; col < width; col++ {
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}

func (buffer *Buffer) FillRoundedRectGradientAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, alpha uint8) {
	if buffer == nil || alpha == 0 {
		return
	}
	if !radii.Active() {
		buffer.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		buffer.FillRoundedRectGradient(x, y, width, height, radii, gradient)
		return
	}
	fromR := int((fromValue >> 16) & 0xFF)
	fromG := int((fromValue >> 8) & 0xFF)
	fromB := int(fromValue & 0xFF)
	toR := int((toValue >> 16) & 0xFF)
	toG := int((toValue >> 8) & 0xFF)
	toB := int(toValue & 0xFF)
	if gradient.Direction == GradientHorizontal {
		den := width - 1
		if den < 1 {
			den = 1
		}
		rowColors := make([]uint32, width)
		rowAlphas := make([]uint8, width)
		for col := 0; col < width; col++ {
			r := (fromR*(den-col) + toR*col) / den
			g := (fromG*(den-col) + toG*col) / den
			b := (fromB*(den-col) + toB*col) / den
			rowColors[col] = uint32(r<<16 | g<<8 | b)
			a := (int(fromAlpha)*(den-col) + int(toAlpha)*col) / den
			rowAlphas[col] = combineAlpha(alpha, uint8(a))
		}
		for row := 0; row < height; row++ {
			leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
			rowStart := 2 + (y+row)*buffer.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				if col < leftWidth || col >= width-rightWidth {
					covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if covAlpha == 0 {
						continue
					}
					effective = combineAlpha(effective, covAlpha)
					if effective == 0 {
						continue
					}
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], rowColors[col], effective)
			}
		}
		return
	}
	den := height - 1
	if den < 1 {
		den = 1
	}
	for row := 0; row < height; row++ {
		r := (fromR*(den-row) + toR*row) / den
		g := (fromG*(den-row) + toG*row) / den
		b := (fromB*(den-row) + toB*row) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		a := (int(fromAlpha)*(den-row) + int(toAlpha)*row) / den
		rowAlpha := combineAlpha(alpha, uint8(a))
		if rowAlpha == 0 {
			continue
		}
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*buffer.width + x
		for col := 0; col < width; col++ {
			effective := rowAlpha
			if col < leftWidth || col >= width-rightWidth {
				covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if covAlpha == 0 {
					continue
				}
				effective = combineAlpha(effective, covAlpha)
				if effective == 0 {
					continue
				}
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}

func clampGradientPos(pos int, length int) int {
	if length <= 1 {
		return 0
	}
	if pos < 0 {
		return 0
	}
	if pos >= length {
		return length - 1
	}
	return pos
}

func gradientDen(length int) int {
	if length <= 1 {
		return 1
	}
	return length - 1
}

func (buffer *Buffer) FillRectGradientArea(x int, y int, width int, height int, gradient Gradient, area Rect) {
	if buffer == nil {
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		buffer.FillRectGradient(x, y, width, height, gradient)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	if width == 1 && height == 1 {
		buffer.FillRect(x, y, width, height, gradient.From)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		buffer.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, 255)
		return
	}
	fromR, fromG, fromB := colorToRGB(gradient.From)
	toR, toG, toB := colorToRGB(gradient.To)
	if gradient.Direction == GradientHorizontal {
		den := gradientDen(area.Width)
		rowColors := make([]uint32, width)
		for col := 0; col < width; col++ {
			pos := clampGradientPos(x+col-area.X, area.Width)
			r := (fromR*(den-pos) + toR*pos) / den
			g := (fromG*(den-pos) + toG*pos) / den
			b := (fromB*(den-pos) + toB*pos) / den
			rowColors[col] = 0xFF000000 | uint32(r<<16|g<<8|b)
		}
		for row := 0; row < height; row++ {
			rowStart := 2 + (y+row)*buffer.width + x
			copy(buffer.data[rowStart:rowStart+width], rowColors)
		}
		return
	}
	den := gradientDen(area.Height)
	for row := 0; row < height; row++ {
		pos := clampGradientPos(y+row-area.Y, area.Height)
		r := (fromR*(den-pos) + toR*pos) / den
		g := (fromG*(den-pos) + toG*pos) / den
		b := (fromB*(den-pos) + toB*pos) / den
		value := 0xFF000000 | uint32(r<<16|g<<8|b)
		rowStart := 2 + (y+row)*buffer.width + x
		fill32(buffer.data[rowStart:rowStart+width], value)
	}
}

func (buffer *Buffer) FillRoundedRectGradientArea(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect) {
	if buffer == nil {
		return
	}
	if !radii.Active() {
		buffer.FillRectGradientArea(x, y, width, height, gradient, area)
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		buffer.FillRoundedRectGradient(x, y, width, height, radii, gradient)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.FillRectGradientArea(x, y, width, height, gradient, area)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		buffer.FillRoundedRectGradientAreaAlpha(x, y, width, height, radii, gradient, area, 255)
		return
	}
	fromR, fromG, fromB := colorToRGB(gradient.From)
	toR, toG, toB := colorToRGB(gradient.To)
	if gradient.Direction == GradientHorizontal {
		den := gradientDen(area.Width)
		rowColors := make([]uint32, width)
		for col := 0; col < width; col++ {
			pos := clampGradientPos(x+col-area.X, area.Width)
			r := (fromR*(den-pos) + toR*pos) / den
			g := (fromG*(den-pos) + toG*pos) / den
			b := (fromB*(den-pos) + toB*pos) / den
			rowColors[col] = 0xFF000000 | uint32(r<<16|g<<8|b)
		}
		for row := 0; row < height; row++ {
			leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
			rowStart := 2 + (y+row)*buffer.width + x
			middleStart := leftWidth
			middleEnd := width - rightWidth
			if middleEnd > middleStart {
				copy(buffer.data[rowStart+middleStart:rowStart+middleEnd], rowColors[middleStart:middleEnd])
			}
			if leftWidth > 0 {
				for col := 0; col < leftWidth; col++ {
					alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if alpha == 0 {
						continue
					}
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						buffer.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
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
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						buffer.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
				}
			}
		}
		return
	}
	den := gradientDen(area.Height)
	for row := 0; row < height; row++ {
		pos := clampGradientPos(y+row-area.Y, area.Height)
		r := (fromR*(den-pos) + toR*pos) / den
		g := (fromG*(den-pos) + toG*pos) / den
		b := (fromB*(den-pos) + toB*pos) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		value := 0xFF000000 | colorValue
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*buffer.width + x
		middleStart := leftWidth
		middleEnd := width - rightWidth
		if middleEnd > middleStart {
			fill32(buffer.data[rowStart+middleStart:rowStart+middleEnd], value)
		}
		if leftWidth > 0 {
			for col := 0; col < leftWidth; col++ {
				alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if alpha == 0 {
					continue
				}
				if alpha >= 255 {
					buffer.data[rowStart+col] = value
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
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
					buffer.data[rowStart+col] = value
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
			}
		}
	}
}

func (buffer *Buffer) FillRectGradientAreaAlpha(x int, y int, width int, height int, gradient Gradient, area Rect, alpha uint8) {
	if buffer == nil || alpha == 0 {
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		buffer.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		buffer.FillRectGradientArea(x, y, width, height, gradient, area)
		return
	}
	fromR := int((fromValue >> 16) & 0xFF)
	fromG := int((fromValue >> 8) & 0xFF)
	fromB := int(fromValue & 0xFF)
	toR := int((toValue >> 16) & 0xFF)
	toG := int((toValue >> 8) & 0xFF)
	toB := int(toValue & 0xFF)
	if gradient.Direction == GradientHorizontal {
		den := gradientDen(area.Width)
		rowColors := make([]uint32, width)
		rowAlphas := make([]uint8, width)
		for col := 0; col < width; col++ {
			pos := clampGradientPos(x+col-area.X, area.Width)
			r := (fromR*(den-pos) + toR*pos) / den
			g := (fromG*(den-pos) + toG*pos) / den
			b := (fromB*(den-pos) + toB*pos) / den
			rowColors[col] = uint32(r<<16 | g<<8 | b)
			a := (int(fromAlpha)*(den-pos) + int(toAlpha)*pos) / den
			rowAlphas[col] = combineAlpha(alpha, uint8(a))
		}
		for row := 0; row < height; row++ {
			rowStart := 2 + (y+row)*buffer.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], rowColors[col], effective)
			}
		}
		return
	}
	den := gradientDen(area.Height)
	for row := 0; row < height; row++ {
		pos := clampGradientPos(y+row-area.Y, area.Height)
		r := (fromR*(den-pos) + toR*pos) / den
		g := (fromG*(den-pos) + toG*pos) / den
		b := (fromB*(den-pos) + toB*pos) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		a := (int(fromAlpha)*(den-pos) + int(toAlpha)*pos) / den
		effective := combineAlpha(alpha, uint8(a))
		if effective == 0 {
			continue
		}
		rowStart := 2 + (y+row)*buffer.width + x
		for col := 0; col < width; col++ {
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}

func (buffer *Buffer) FillRoundedRectGradientAreaAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect, alpha uint8) {
	if buffer == nil || alpha == 0 {
		return
	}
	if !radii.Active() {
		buffer.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, alpha)
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		buffer.FillRoundedRectGradientAlpha(x, y, width, height, radii, gradient, alpha)
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		buffer.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, alpha)
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		buffer.FillRoundedRectGradientArea(x, y, width, height, radii, gradient, area)
		return
	}
	fromR := int((fromValue >> 16) & 0xFF)
	fromG := int((fromValue >> 8) & 0xFF)
	fromB := int(fromValue & 0xFF)
	toR := int((toValue >> 16) & 0xFF)
	toG := int((toValue >> 8) & 0xFF)
	toB := int(toValue & 0xFF)
	if gradient.Direction == GradientHorizontal {
		den := gradientDen(area.Width)
		rowColors := make([]uint32, width)
		rowAlphas := make([]uint8, width)
		for col := 0; col < width; col++ {
			pos := clampGradientPos(x+col-area.X, area.Width)
			r := (fromR*(den-pos) + toR*pos) / den
			g := (fromG*(den-pos) + toG*pos) / den
			b := (fromB*(den-pos) + toB*pos) / den
			rowColors[col] = uint32(r<<16 | g<<8 | b)
			a := (int(fromAlpha)*(den-pos) + int(toAlpha)*pos) / den
			rowAlphas[col] = combineAlpha(alpha, uint8(a))
		}
		for row := 0; row < height; row++ {
			leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
			rowStart := 2 + (y+row)*buffer.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				if col < leftWidth || col >= width-rightWidth {
					covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if covAlpha == 0 {
						continue
					}
					effective = combineAlpha(effective, covAlpha)
					if effective == 0 {
						continue
					}
				}
				buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], rowColors[col], effective)
			}
		}
		return
	}
	den := gradientDen(area.Height)
	for row := 0; row < height; row++ {
		pos := clampGradientPos(y+row-area.Y, area.Height)
		r := (fromR*(den-pos) + toR*pos) / den
		g := (fromG*(den-pos) + toG*pos) / den
		b := (fromB*(den-pos) + toB*pos) / den
		colorValue := uint32(r<<16 | g<<8 | b)
		a := (int(fromAlpha)*(den-pos) + int(toAlpha)*pos) / den
		rowAlpha := combineAlpha(alpha, uint8(a))
		if rowAlpha == 0 {
			continue
		}
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*buffer.width + x
		for col := 0; col < width; col++ {
			effective := rowAlpha
			if col < leftWidth || col >= width-rightWidth {
				covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if covAlpha == 0 {
					continue
				}
				effective = combineAlpha(effective, covAlpha)
				if effective == 0 {
					continue
				}
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}
