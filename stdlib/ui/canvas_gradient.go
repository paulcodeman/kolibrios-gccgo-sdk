package ui

func (canvas *Canvas) FillRectGradient(x int, y int, width int, height int, gradient Gradient) {
	if canvas == nil {
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	if width == 1 && height == 1 {
		canvas.FillRect(x, y, width, height, gradient.From)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		canvas.FillRectGradientAlpha(x, y, width, height, gradient, 255)
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
			rowStart := 2 + (y+row)*canvas.width + x
			copy(canvas.data[rowStart:rowStart+width], rowColors)
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
		rowStart := 2 + (y+row)*canvas.width + x
		fillSlice32(canvas.data[rowStart:rowStart+width], value)
	}
}

func (canvas *Canvas) FillRoundedRectGradient(x int, y int, width int, height int, radii CornerRadii, gradient Gradient) {
	if canvas == nil {
		return
	}
	if !radii.Active() {
		canvas.FillRectGradient(x, y, width, height, gradient)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRectGradient(x, y, width, height, gradient)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		canvas.FillRoundedRectGradientAlpha(x, y, width, height, radii, gradient, 255)
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
			rowStart := 2 + (y+row)*canvas.width + x
			middleStart := leftWidth
			middleEnd := width - rightWidth
			if middleEnd > middleStart {
				copy(canvas.data[rowStart+middleStart:rowStart+middleEnd], rowColors[middleStart:middleEnd])
			}
			if leftWidth > 0 {
				for col := 0; col < leftWidth; col++ {
					alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if alpha == 0 {
						continue
					}
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						canvas.data[rowStart+col] = 0xFF000000 | colorValue
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
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						canvas.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					dst := canvas.data[rowStart+col]
					canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, alpha)
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

func (canvas *Canvas) FillRectGradientAlpha(x int, y int, width int, height int, gradient Gradient, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		canvas.FillRectGradient(x, y, width, height, gradient)
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
			rowStart := 2 + (y+row)*canvas.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, rowColors[col], effective)
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
		rowStart := 2 + (y+row)*canvas.width + x
		for col := 0; col < width; col++ {
			dst := canvas.data[rowStart+col]
			canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
		}
	}
}

func (canvas *Canvas) FillRoundedRectGradientAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	if !radii.Active() {
		canvas.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		canvas.FillRoundedRectGradient(x, y, width, height, radii, gradient)
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
			rowStart := 2 + (y+row)*canvas.width + x
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
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, rowColors[col], effective)
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
		rowStart := 2 + (y+row)*canvas.width + x
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
			dst := canvas.data[rowStart+col]
			canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
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

func (canvas *Canvas) FillRectGradientArea(x int, y int, width int, height int, gradient Gradient, area Rect) {
	if canvas == nil {
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		canvas.FillRectGradient(x, y, width, height, gradient)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	if width == 1 && height == 1 {
		canvas.FillRect(x, y, width, height, gradient.From)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		canvas.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, 255)
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
			rowStart := 2 + (y+row)*canvas.width + x
			copy(canvas.data[rowStart:rowStart+width], rowColors)
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
		rowStart := 2 + (y+row)*canvas.width + x
		fillSlice32(canvas.data[rowStart:rowStart+width], value)
	}
}

func (canvas *Canvas) FillRoundedRectGradientArea(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect) {
	if canvas == nil {
		return
	}
	if !radii.Active() {
		canvas.FillRectGradientArea(x, y, width, height, gradient, area)
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		canvas.FillRoundedRectGradient(x, y, width, height, radii, gradient)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRectGradientArea(x, y, width, height, gradient, area)
		return
	}
	_, fromAlpha := colorValueAndAlpha(gradient.From)
	_, toAlpha := colorValueAndAlpha(gradient.To)
	if fromAlpha < 255 || toAlpha < 255 {
		canvas.FillRoundedRectGradientAreaAlpha(x, y, width, height, radii, gradient, area, 255)
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
			rowStart := 2 + (y+row)*canvas.width + x
			middleStart := leftWidth
			middleEnd := width - rightWidth
			if middleEnd > middleStart {
				copy(canvas.data[rowStart+middleStart:rowStart+middleEnd], rowColors[middleStart:middleEnd])
			}
			if leftWidth > 0 {
				for col := 0; col < leftWidth; col++ {
					alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
					if alpha == 0 {
						continue
					}
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						canvas.data[rowStart+col] = 0xFF000000 | colorValue
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
					colorValue := rowColors[col] & 0xFFFFFF
					if alpha >= 255 {
						canvas.data[rowStart+col] = 0xFF000000 | colorValue
						continue
					}
					dst := canvas.data[rowStart+col]
					canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, alpha)
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

func (canvas *Canvas) FillRectGradientAreaAlpha(x int, y int, width int, height int, gradient Gradient, area Rect, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		canvas.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		canvas.FillRectGradientArea(x, y, width, height, gradient, area)
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
			rowStart := 2 + (y+row)*canvas.width + x
			for col := 0; col < width; col++ {
				effective := rowAlphas[col]
				if effective == 0 {
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, rowColors[col], effective)
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
		rowStart := 2 + (y+row)*canvas.width + x
		for col := 0; col < width; col++ {
			dst := canvas.data[rowStart+col]
			canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
		}
	}
}

func (canvas *Canvas) FillRoundedRectGradientAreaAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	if !radii.Active() {
		canvas.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, alpha)
		return
	}
	if area.Width <= 0 || area.Height <= 0 || (area == Rect{}) {
		canvas.FillRoundedRectGradientAlpha(x, y, width, height, radii, gradient, alpha)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, alpha)
		return
	}
	fromValue, fromAlpha := colorValueAndAlpha(gradient.From)
	toValue, toAlpha := colorValueAndAlpha(gradient.To)
	if alpha >= 255 && fromAlpha >= 255 && toAlpha >= 255 {
		canvas.FillRoundedRectGradientArea(x, y, width, height, radii, gradient, area)
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
			rowStart := 2 + (y+row)*canvas.width + x
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
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, rowColors[col], effective)
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
		rowStart := 2 + (y+row)*canvas.width + x
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
			dst := canvas.data[rowStart+col]
			canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
		}
	}
}
