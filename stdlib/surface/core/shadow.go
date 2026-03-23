package core

import "sync"

type shadowMaskKey struct {
	width    int
	height   int
	offsetX  int
	offsetY  int
	blur     int
	alpha    uint8
	rounded  bool
	topLeft  int
	topRight int
	botRight int
	botLeft  int
}

type shadowMask struct {
	width  int
	height int
	data   []uint8
	rows   []shadowMaskRow
}

type shadowMaskRow struct {
	firstStart  int
	firstEnd    int
	secondStart int
	secondEnd   int
}

var (
	shadowMaskMu    sync.Mutex
	shadowMaskCache = map[shadowMaskKey]*shadowMask{}
)

func (buffer *Buffer) DrawShadow(rect Rect, shadow Shadow) {
	buffer.drawShadowMasked(rect, shadow, CornerRadii{})
}

func (buffer *Buffer) DrawShadowRounded(rect Rect, shadow Shadow, radii CornerRadii) {
	buffer.drawShadowMasked(rect, shadow, radii)
}

func (buffer *Buffer) drawShadowMasked(rect Rect, shadow Shadow, radii CornerRadii) {
	if buffer == nil || rect.Empty() {
		return
	}
	colorValue, colorAlpha := colorValueAndAlpha(shadow.Color)
	alpha := shadow.Alpha
	if colorAlpha < 255 {
		alpha = combineAlpha(alpha, colorAlpha)
	}
	if alpha == 0 {
		return
	}
	blur := shadow.Blur
	if blur < 0 {
		blur = 0
	}
	radii = normalizeRadii(rect.Width, rect.Height, radii)
	mask := cachedShadowMask(rect.Width, rect.Height, shadow.OffsetX, shadow.OffsetY, blur, alpha, radii)
	if mask == nil || len(mask.data) == 0 {
		return
	}
	destRect := Rect{
		X:      rect.X + shadow.OffsetX - blur,
		Y:      rect.Y + shadow.OffsetY - blur,
		Width:  mask.width,
		Height: mask.height,
	}
	drawRect := IntersectRect(destRect, buffer.Bounds())
	if buffer.clip.set {
		drawRect = IntersectRect(drawRect, buffer.clip.rect)
	}
	if drawRect.Empty() {
		return
	}
	srcX := drawRect.X - destRect.X
	srcY := drawRect.Y - destRect.Y
	if buffer.alpha {
		buffer.blitShadowMaskAlpha(drawRect, srcX, srcY, mask, colorValue)
		return
	}
	buffer.blitShadowMaskOpaque(drawRect, srcX, srcY, mask, colorValue)
}

func cachedShadowMask(width int, height int, offsetX int, offsetY int, blur int, alpha uint8, radii CornerRadii) *shadowMask {
	if width <= 0 || height <= 0 || alpha == 0 {
		return nil
	}
	key := shadowMaskKey{
		width:    width,
		height:   height,
		offsetX:  offsetX,
		offsetY:  offsetY,
		blur:     blur,
		alpha:    alpha,
		rounded:  radii.Active(),
		topLeft:  radii.TopLeft,
		topRight: radii.TopRight,
		botRight: radii.BottomRight,
		botLeft:  radii.BottomLeft,
	}
	shadowMaskMu.Lock()
	mask := shadowMaskCache[key]
	shadowMaskMu.Unlock()
	if mask != nil {
		return mask
	}
	mask = buildShadowMask(key)
	shadowMaskMu.Lock()
	if existing := shadowMaskCache[key]; existing != nil {
		shadowMaskMu.Unlock()
		return existing
	}
	shadowMaskCache[key] = mask
	shadowMaskMu.Unlock()
	return mask
}

func buildShadowMask(key shadowMaskKey) *shadowMask {
	maskWidth := key.width + key.blur*2
	maskHeight := key.height + key.blur*2
	if maskWidth <= 0 || maskHeight <= 0 {
		return nil
	}
	maskBuffer := NewBufferAlpha(maskWidth, maskHeight)
	radii := CornerRadii{
		TopLeft:     key.topLeft,
		TopRight:    key.topRight,
		BottomRight: key.botRight,
		BottomLeft:  key.botLeft,
	}
	layers := key.blur + 1
	for i := key.blur; i >= 0; i-- {
		layerAlpha := int(key.alpha)
		if key.blur > 0 {
			layerAlpha = int(key.alpha) * (key.blur - i + 1) / layers
		}
		if layerAlpha <= 0 {
			continue
		}
		x := key.blur - i
		y := key.blur - i
		width := key.width + i*2
		height := key.height + i*2
		if key.rounded {
			maskBuffer.FillRoundedRectAlpha(x, y, width, height, CornerRadii{
				TopLeft:     radii.TopLeft + i,
				TopRight:    radii.TopRight + i,
				BottomRight: radii.BottomRight + i,
				BottomLeft:  radii.BottomLeft + i,
			}, 0xFFFFFF, uint8(layerAlpha))
			continue
		}
		maskBuffer.FillRectAlpha(x, y, width, height, 0xFFFFFF, uint8(layerAlpha))
	}
	applyShadowCutout(maskBuffer, key.blur-key.offsetX, key.blur-key.offsetY, key.width, key.height, radii)
	mask := &shadowMask{
		width:  maskWidth,
		height: maskHeight,
		data:   make([]uint8, maskWidth*maskHeight),
		rows:   make([]shadowMaskRow, maskHeight),
	}
	for row := 0; row < maskHeight; row++ {
		rowOffset := row * maskWidth
		rowInfo := shadowMaskRow{
			firstStart:  -1,
			firstEnd:    -1,
			secondStart: -1,
			secondEnd:   -1,
		}
		currentStart := -1
		runCount := 0
		for col := 0; col < maskWidth; col++ {
			alpha := uint8(maskBuffer.data[rowOffset+col+2] >> 24)
			mask.data[rowOffset+col] = alpha
			if alpha != 0 {
				if currentStart < 0 {
					currentStart = col
				}
				continue
			}
			if currentStart < 0 {
				continue
			}
			if runCount == 0 {
				rowInfo.firstStart = currentStart
				rowInfo.firstEnd = col
			} else if runCount == 1 {
				rowInfo.secondStart = currentStart
				rowInfo.secondEnd = col
			}
			runCount++
			currentStart = -1
		}
		if currentStart >= 0 {
			if runCount == 0 {
				rowInfo.firstStart = currentStart
				rowInfo.firstEnd = maskWidth
			} else if runCount == 1 {
				rowInfo.secondStart = currentStart
				rowInfo.secondEnd = maskWidth
			}
		}
		mask.rows[row] = rowInfo
	}
	return mask
}

func applyShadowCutout(buffer *Buffer, x int, y int, width int, height int, radii CornerRadii) {
	if buffer == nil || width <= 0 || height <= 0 {
		return
	}
	radii = normalizeRadii(width, height, radii)
	for row := 0; row < height; row++ {
		dstY := y + row
		if dstY < 0 || dstY >= buffer.height {
			continue
		}
		rowStart := 2 + dstY*buffer.width
		for col := 0; col < width; col++ {
			dstX := x + col
			if dstX < 0 || dstX >= buffer.width {
				continue
			}
			coverage := uint8(255)
			if radii.Active() {
				coverage = roundedPixelCoverageAlpha(col, row, width, height, radii)
				if coverage == 0 {
					continue
				}
			}
			index := rowStart + dstX
			alpha := uint8(buffer.data[index] >> 24)
			if alpha == 0 {
				continue
			}
			invCoverage := 255 - int(coverage)
			if invCoverage <= 0 {
				buffer.data[index] = 0
				continue
			}
			alpha = uint8((int(alpha)*invCoverage + 127) / 255)
			if alpha == 0 {
				buffer.data[index] = 0
				continue
			}
			value := uint32(alpha)
			buffer.data[index] = (value << 24) | (value << 16) | (value << 8) | value
		}
	}
}

func (buffer *Buffer) blitShadowMaskOpaque(drawRect Rect, srcX int, srcY int, mask *shadowMask, colorValue uint32) {
	for row := 0; row < drawRect.Height; row++ {
		dstIndex := 2 + (drawRect.Y+row)*buffer.width + drawRect.X
		srcIndex := (srcY+row)*mask.width + srcX
		rowInfo := mask.rows[srcY+row]
		buffer.blitShadowMaskOpaqueSpan(dstIndex, srcIndex, srcX, drawRect.Width, rowInfo.firstStart, rowInfo.firstEnd, mask, colorValue)
		buffer.blitShadowMaskOpaqueSpan(dstIndex, srcIndex, srcX, drawRect.Width, rowInfo.secondStart, rowInfo.secondEnd, mask, colorValue)
	}
}

func (buffer *Buffer) blitShadowMaskAlpha(drawRect Rect, srcX int, srcY int, mask *shadowMask, colorValue uint32) {
	for row := 0; row < drawRect.Height; row++ {
		dstIndex := 2 + (drawRect.Y+row)*buffer.width + drawRect.X
		srcIndex := (srcY+row)*mask.width + srcX
		rowInfo := mask.rows[srcY+row]
		buffer.blitShadowMaskAlphaSpan(dstIndex, srcIndex, srcX, drawRect.Width, rowInfo.firstStart, rowInfo.firstEnd, mask, colorValue)
		buffer.blitShadowMaskAlphaSpan(dstIndex, srcIndex, srcX, drawRect.Width, rowInfo.secondStart, rowInfo.secondEnd, mask, colorValue)
	}
}

func (buffer *Buffer) blitShadowMaskOpaqueSpan(dstIndex int, srcIndex int, srcX int, width int, spanStart int, spanEnd int, mask *shadowMask, colorValue uint32) {
	if buffer == nil || mask == nil || spanStart < 0 || spanEnd <= spanStart || width <= 0 {
		return
	}
	clipStart := srcX
	clipEnd := srcX + width
	if spanStart < clipStart {
		spanStart = clipStart
	}
	if spanEnd > clipEnd {
		spanEnd = clipEnd
	}
	if spanEnd <= spanStart {
		return
	}
	start := spanStart - srcX
	end := spanEnd - srcX
	if colorValue == 0 {
		for col := start; col < end; col++ {
			alpha := mask.data[srcIndex+col]
			if alpha == 0 {
				continue
			}
			buffer.data[dstIndex+col] = darkenPixelValue(buffer.data[dstIndex+col], alpha)
		}
		return
	}
	for col := start; col < end; col++ {
		alpha := mask.data[srcIndex+col]
		if alpha == 0 {
			continue
		}
		buffer.data[dstIndex+col] = blendPixelValue(buffer.data[dstIndex+col], colorValue, alpha)
	}
}

func (buffer *Buffer) blitShadowMaskAlphaSpan(dstIndex int, srcIndex int, srcX int, width int, spanStart int, spanEnd int, mask *shadowMask, colorValue uint32) {
	if buffer == nil || mask == nil || spanStart < 0 || spanEnd <= spanStart || width <= 0 {
		return
	}
	clipStart := srcX
	clipEnd := srcX + width
	if spanStart < clipStart {
		spanStart = clipStart
	}
	if spanEnd > clipEnd {
		spanEnd = clipEnd
	}
	if spanEnd <= spanStart {
		return
	}
	start := spanStart - srcX
	end := spanEnd - srcX
	for col := start; col < end; col++ {
		alpha := mask.data[srcIndex+col]
		if alpha == 0 {
			continue
		}
		src := premultiplyColorValue(colorValue, alpha)
		buffer.data[dstIndex+col] = blendPremultiplied(buffer.data[dstIndex+col], src)
	}
}
