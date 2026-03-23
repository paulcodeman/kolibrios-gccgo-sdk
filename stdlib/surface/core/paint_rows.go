package core

func (buffer *Buffer) blendRowValue(rowStart int, width int, colorValue uint32, alpha uint8) {
	if buffer == nil || width <= 0 || alpha == 0 {
		return
	}
	if alpha >= 255 {
		fill32(buffer.data[rowStart:rowStart+width], 0xFF000000|colorValue)
		return
	}
	if !buffer.alpha {
		srcR := int((colorValue >> 16) & 0xFF)
		srcG := int((colorValue >> 8) & 0xFF)
		srcB := int(colorValue & 0xFF)
		alphaInt := int(alpha)
		invAlpha := 255 - alphaInt
		srcRA := srcR * alphaInt
		srcGA := srcG * alphaInt
		srcBA := srcB * alphaInt
		for col := 0; col < width; col++ {
			dst := buffer.data[rowStart+col]
			dstR := int((dst >> 16) & 0xFF)
			dstG := int((dst >> 8) & 0xFF)
			dstB := int(dst & 0xFF)
			outR := (srcRA + dstR*invAlpha + 127) / 255
			outG := (srcGA + dstG*invAlpha + 127) / 255
			outB := (srcBA + dstB*invAlpha + 127) / 255
			buffer.data[rowStart+col] = 0xFF000000 | uint32(outR<<16|outG<<8|outB)
		}
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

func (buffer *Buffer) blendRowColorSamples(rowStart int, samples []uint32, alpha uint8) {
	if buffer == nil || len(samples) == 0 || alpha == 0 {
		return
	}
	if alpha >= 255 {
		for col, sample := range samples {
			buffer.data[rowStart+col] = 0xFF000000 | (sample & 0xFFFFFF)
		}
		return
	}
	if !buffer.alpha {
		alphaInt := int(alpha)
		invAlpha := 255 - alphaInt
		for col, sample := range samples {
			colorValue := sample & 0xFFFFFF
			srcR := int((colorValue >> 16) & 0xFF)
			srcG := int((colorValue >> 8) & 0xFF)
			srcB := int(colorValue & 0xFF)
			dst := buffer.data[rowStart+col]
			dstR := int((dst >> 16) & 0xFF)
			dstG := int((dst >> 8) & 0xFF)
			dstB := int(dst & 0xFF)
			outR := (srcR*alphaInt + dstR*invAlpha + 127) / 255
			outG := (srcG*alphaInt + dstG*invAlpha + 127) / 255
			outB := (srcB*alphaInt + dstB*invAlpha + 127) / 255
			buffer.data[rowStart+col] = 0xFF000000 | uint32(outR<<16|outG<<8|outB)
		}
		return
	}
	for col, sample := range samples {
		buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], sample&0xFFFFFF, alpha)
	}
}

func (buffer *Buffer) paintRoundedRowOpaqueValue(rowStart int, row int, width int, height int, radii CornerRadii, value uint32) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	shape := roundedShapeRows(width, height, radii)
	if shape == nil {
		fill32(buffer.data[rowStart:rowStart+width], value)
		return
	}
	rowInfo := shape.rows[row]
	leftWidth := rowInfo.leftWidth
	rightWidth := rowInfo.rightWidth
	middleStart := leftWidth
	middleEnd := rowInfo.rightStart
	if middleEnd > middleStart {
		fill32(buffer.data[rowStart+middleStart:rowStart+middleEnd], value)
	}
	colorValue := value & 0xFFFFFF
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			alpha := rowInfo.leftAlpha[col]
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
		start := rowInfo.rightStart
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			alpha := rowInfo.rightAlpha[col-rowInfo.rightStart]
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
	shape := roundedShapeRows(width, height, radii)
	if shape == nil {
		copy(buffer.data[rowStart:rowStart+width], samples[:width])
		return
	}
	rowInfo := shape.rows[row]
	leftWidth := rowInfo.leftWidth
	rightWidth := rowInfo.rightWidth
	middleStart := leftWidth
	middleEnd := rowInfo.rightStart
	if middleEnd > middleStart {
		copy(buffer.data[rowStart+middleStart:rowStart+middleEnd], samples[middleStart:middleEnd])
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			alpha := rowInfo.leftAlpha[col]
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
		start := rowInfo.rightStart
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			alpha := rowInfo.rightAlpha[col-rowInfo.rightStart]
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
	shape := roundedShapeRows(width, height, radii)
	if shape == nil {
		buffer.blendRowValue(rowStart, width, colorValue, alpha)
		return
	}
	rowInfo := shape.rows[row]
	leftWidth := rowInfo.leftWidth
	rightWidth := rowInfo.rightWidth
	middleStart := leftWidth
	middleEnd := rowInfo.rightStart
	if middleEnd > middleStart {
		buffer.blendRowValue(rowStart+middleStart, middleEnd-middleStart, colorValue, alpha)
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			covAlpha := rowInfo.leftAlpha[col]
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
		start := rowInfo.rightStart
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			covAlpha := rowInfo.rightAlpha[col-rowInfo.rightStart]
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
	shape := roundedShapeRows(width, height, radii)
	if shape == nil {
		buffer.blendRowSamples(rowStart, samples[:width])
		return
	}
	rowInfo := shape.rows[row]
	leftWidth := rowInfo.leftWidth
	rightWidth := rowInfo.rightWidth
	middleStart := leftWidth
	middleEnd := rowInfo.rightStart
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
			covAlpha := rowInfo.leftAlpha[col]
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
		start := rowInfo.rightStart
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			sample := samples[col]
			sampleAlpha := uint8(sample >> 24)
			if sampleAlpha == 0 {
				continue
			}
			covAlpha := rowInfo.rightAlpha[col-rowInfo.rightStart]
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

func (buffer *Buffer) paintRoundedRowAlphaColorSamples(rowStart int, row int, width int, height int, radii CornerRadii, samples []uint32, alpha uint8) {
	if buffer == nil || width <= 0 || height <= 0 || len(samples) < width || alpha == 0 {
		return
	}
	shape := roundedShapeRows(width, height, radii)
	if shape == nil {
		buffer.blendRowColorSamples(rowStart, samples[:width], alpha)
		return
	}
	rowInfo := shape.rows[row]
	leftWidth := rowInfo.leftWidth
	rightWidth := rowInfo.rightWidth
	middleStart := leftWidth
	middleEnd := rowInfo.rightStart
	if middleEnd > middleStart {
		buffer.blendRowColorSamples(rowStart+middleStart, samples[middleStart:middleEnd], alpha)
	}
	if leftWidth > 0 {
		for col := 0; col < leftWidth; col++ {
			covAlpha := rowInfo.leftAlpha[col]
			if covAlpha == 0 {
				continue
			}
			effective := combineAlpha(alpha, covAlpha)
			if effective == 0 {
				continue
			}
			colorValue := samples[col] & 0xFFFFFF
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
	if rightWidth > 0 {
		start := rowInfo.rightStart
		if start < leftWidth {
			start = leftWidth
		}
		for col := start; col < width; col++ {
			covAlpha := rowInfo.rightAlpha[col-rowInfo.rightStart]
			if covAlpha == 0 {
				continue
			}
			effective := combineAlpha(alpha, covAlpha)
			if effective == 0 {
				continue
			}
			colorValue := samples[col] & 0xFFFFFF
			if effective >= 255 {
				buffer.data[rowStart+col] = 0xFF000000 | colorValue
				continue
			}
			buffer.data[rowStart+col] = buffer.blendPixel(buffer.data[rowStart+col], colorValue, effective)
		}
	}
}
