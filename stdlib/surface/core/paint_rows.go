package core

func (buffer *Buffer) blendRowValue(rowStart int, width int, colorValue uint32, alpha uint8) {
	if buffer == nil || width <= 0 || alpha == 0 {
		return
	}
	if alpha >= 255 {
		fill32(buffer.data[rowStart:rowStart+width], 0xFF000000|colorValue)
		return
	}
	for col := 0; col < width; col++ {
		buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
	}
}

func (buffer *Buffer) blendRowSamples(rowStart int, samples []uint32) {
	if buffer == nil || len(samples) == 0 {
		return
	}
	for col, sample := range samples {
		alpha := uint8(sample >> 24)
		if alpha == 0 {
			continue
		}
		colorValue := sample & 0xFFFFFF
		if alpha >= 255 {
			buffer.data[rowStart+col] = 0xFF000000 | colorValue
			continue
		}
		buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
	}
}

func (buffer *Buffer) paintRoundedRowOpaqueValue(rowStart int, row int, width int, height int, radii CornerRadii, value uint32) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
	middleStart := leftWidth
	middleEnd := width - rightWidth
	if middleEnd > middleStart {
		fill32(buffer.data[rowStart+middleStart:rowStart+middleEnd], value)
	}
	colorValue := value & 0xFFFFFF
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

func (buffer *Buffer) paintRoundedRowOpaqueSamples(rowStart int, row int, width int, height int, radii CornerRadii, samples []uint32) {
	if buffer == nil || width <= 0 || height <= 0 || len(samples) < width {
		return
	}
	leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
	middleStart := leftWidth
	middleEnd := width - rightWidth
	if middleEnd > middleStart {
		copy(buffer.data[rowStart+middleStart:rowStart+middleEnd], samples[middleStart:middleEnd])
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			alpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
			if alpha == 0 {
				continue
			}
			sample := samples[col]
			colorValue := sample & 0xFFFFFF
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
			sample := samples[col]
			colorValue := sample & 0xFFFFFF
			if alpha >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, alpha)
		}
	}
}

func (buffer *Buffer) paintRoundedRowAlphaValue(rowStart int, row int, width int, height int, radii CornerRadii, colorValue uint32, alpha uint8) {
	if buffer == nil || width <= 0 || height <= 0 || alpha == 0 {
		return
	}
	leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
	middleStart := leftWidth
	middleEnd := width - rightWidth
	if middleEnd > middleStart {
		buffer.blendRowValue(rowStart+middleStart, middleEnd-middleStart, colorValue, alpha)
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
			if covAlpha == 0 {
				continue
			}
			effective := combineAlpha(alpha, covAlpha)
			if effective == 0 {
				continue
			}
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
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
			effective := combineAlpha(alpha, covAlpha)
			if effective == 0 {
				continue
			}
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}

func (buffer *Buffer) paintRoundedRowAlphaSamples(rowStart int, row int, width int, height int, radii CornerRadii, samples []uint32) {
	if buffer == nil || width <= 0 || height <= 0 || len(samples) < width {
		return
	}
	leftWidth, rightWidth := cornerWidthsForRow(row, height, radii)
	middleStart := leftWidth
	middleEnd := width - rightWidth
	if middleEnd > middleStart {
		buffer.blendRowSamples(rowStart+middleStart, samples[middleStart:middleEnd])
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			sample := samples[col]
			sampleAlpha := uint8(sample >> 24)
			if sampleAlpha == 0 {
				continue
			}
			covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
			if covAlpha == 0 {
				continue
			}
			effective := combineAlpha(sampleAlpha, covAlpha)
			if effective == 0 {
				continue
			}
			colorValue := sample & 0xFFFFFF
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
	if rightWidth > 0 {
		start := width - rightWidth
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			sample := samples[col]
			sampleAlpha := uint8(sample >> 24)
			if sampleAlpha == 0 {
				continue
			}
			covAlpha := roundedPixelCoverageAlpha(col, row, width, height, radii)
			if covAlpha == 0 {
				continue
			}
			effective := combineAlpha(sampleAlpha, covAlpha)
			if effective == 0 {
				continue
			}
			colorValue := sample & 0xFFFFFF
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}
