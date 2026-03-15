package ui

import "kos"

func (canvas *Canvas) FillRectAlpha(x int, y int, width int, height int, color kos.Color, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	rgb, colorAlpha := colorValueAndAlpha(color)
	if colorAlpha < 255 {
		alpha = combineAlpha(alpha, colorAlpha)
		if alpha == 0 {
			return
		}
	}
	if alpha >= 255 {
		canvas.FillRect(x, y, width, height, kos.Color(rgb))
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	colorValue := rgb
	if canvas.alpha {
		src := premultiplyColorValue(colorValue, alpha)
		sa := (src >> 24) & 0xFF
		if sa == 0 {
			return
		}
		rowStart := 2 + y*canvas.width + x
		if sa >= 255 {
			if x == 0 && width == canvas.width {
				fillSlice32(canvas.data[rowStart:rowStart+width*height], src)
				return
			}
			for row := 0; row < height; row++ {
				index := rowStart + row*canvas.width
				fillSlice32(canvas.data[index:index+width], src)
			}
			return
		}
		for row := 0; row < height; row++ {
			index := rowStart + row*canvas.width
			for col := 0; col < width; col++ {
				dst := canvas.data[index+col]
				canvas.data[index+col] = blendPremultiplied(dst, src)
			}
		}
		return
	}

	srcR := int((colorValue >> 16) & 0xFF)
	srcG := int((colorValue >> 8) & 0xFF)
	srcB := int(colorValue & 0xFF)
	alphaInt := int(alpha)
	invAlpha := 255 - alphaInt
	srcRA := srcR * alphaInt
	srcGA := srcG * alphaInt
	srcBA := srcB * alphaInt
	rowStart := 2 + y*canvas.width + x
	if x == 0 && width == canvas.width {
		total := width * height
		for i := 0; i < total; i++ {
			dst := canvas.data[rowStart+i]
			dstR := int((dst >> 16) & 0xFF)
			dstG := int((dst >> 8) & 0xFF)
			dstB := int(dst & 0xFF)
			outR := (srcRA + dstR*invAlpha + 127) / 255
			outG := (srcGA + dstG*invAlpha + 127) / 255
			outB := (srcBA + dstB*invAlpha + 127) / 255
			canvas.data[rowStart+i] = 0xFF000000 | uint32(outR<<16|outG<<8|outB)
		}
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*canvas.width
		for col := 0; col < width; col++ {
			dst := canvas.data[index+col]
			dstR := int((dst >> 16) & 0xFF)
			dstG := int((dst >> 8) & 0xFF)
			dstB := int(dst & 0xFF)
			outR := (srcRA + dstR*invAlpha + 127) / 255
			outG := (srcGA + dstG*invAlpha + 127) / 255
			outB := (srcBA + dstB*invAlpha + 127) / 255
			canvas.data[index+col] = 0xFF000000 | uint32(outR<<16|outG<<8|outB)
		}
	}
}

func (canvas *Canvas) FillRoundedRectAlpha(x int, y int, width int, height int, radii CornerRadii, color kos.Color, alpha uint8) {
	if canvas == nil || alpha == 0 {
		return
	}
	rgb, colorAlpha := colorValueAndAlpha(color)
	if colorAlpha < 255 {
		alpha = combineAlpha(alpha, colorAlpha)
		if alpha == 0 {
			return
		}
	}
	if alpha >= 255 {
		canvas.FillRoundedRect(x, y, width, height, radii, kos.Color(rgb))
		return
	}
	if !radii.Active() {
		canvas.FillRectAlpha(x, y, width, height, kos.Color(rgb), alpha)
		return
	}
	x, y, width, height, ok := canvas.clampRect(x, y, width, height)
	if !ok {
		return
	}
	radii = normalizeRadii(width, height, radii)
	if !radii.Active() {
		canvas.FillRectAlpha(x, y, width, height, kos.Color(rgb), alpha)
		return
	}
	colorValue := rgb
	for row := 0; row < height; row++ {
		leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
		rowStart := 2 + (y+row)*canvas.width + x
		middleStart := leftWidth
		middleEnd := width - rightWidth
		if middleEnd > middleStart {
			for col := middleStart; col < middleEnd; col++ {
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, alpha)
			}
		}
		if leftWidth > 0 {
			for col := 0; col < leftWidth; col++ {
				covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if covAlpha == 0 {
					continue
				}
				effective := uint8((int(alpha)*int(covAlpha) + 127) / 255)
				if effective == 0 {
					continue
				}
				if effective >= 255 {
					canvas.data[rowStart+col] = 0xFF000000 | colorValue
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
			}
		}
		if rightWidth > 0 {
			start := width - rightWidth
			if start < leftWidth {
				start = leftWidth
			}
			for col := start; col < width; col++ {
				covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
				if covAlpha == 0 {
					continue
				}
				effective := uint8((int(alpha)*int(covAlpha) + 127) / 255)
				if effective == 0 {
					continue
				}
				if effective >= 255 {
					canvas.data[rowStart+col] = 0xFF000000 | colorValue
					continue
				}
				dst := canvas.data[rowStart+col]
				canvas.data[rowStart+col] = canvas.blendPixel(dst, colorValue, effective)
			}
		}
	}
}
