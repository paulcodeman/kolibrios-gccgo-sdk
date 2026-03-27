package core

import "math"

func (buffer *Buffer) DrawImagePixels(x int, y int, width int, height int, pixels []uint32, srcOpaque bool) {
	if buffer == nil || width <= 0 || height <= 0 || len(pixels) < width*height {
		return
	}
	dstBounds := buffer.Bounds()
	if buffer.clip.set {
		dstBounds = IntersectRect(dstBounds, buffer.clip.rect)
	}
	dstRect := IntersectRect(Rect{X: x, Y: y, Width: width, Height: height}, dstBounds)
	if dstRect.Empty() {
		return
	}
	srcX := dstRect.X - x
	srcY := dstRect.Y - y
	for row := 0; row < dstRect.Height; row++ {
		dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
		srcIndex := (srcY+row)*width + srcX
		drawImageSpan(buffer, dstIndex, pixels[srcIndex:srcIndex+dstRect.Width], srcOpaque)
	}
}

func (buffer *Buffer) DrawImagePixelsRect(rect Rect, srcWidth int, srcHeight int, pixels []uint32, srcOpaque bool) {
	if buffer == nil || rect.Empty() || srcWidth <= 0 || srcHeight <= 0 || len(pixels) < srcWidth*srcHeight {
		return
	}
	if rect.Width == srcWidth && rect.Height == srcHeight {
		buffer.DrawImagePixels(rect.X, rect.Y, srcWidth, srcHeight, pixels, srcOpaque)
		return
	}
	dstBounds := buffer.Bounds()
	if buffer.clip.set {
		dstBounds = IntersectRect(dstBounds, buffer.clip.rect)
	}
	visible := IntersectRect(rect, dstBounds)
	if visible.Empty() {
		return
	}
	xMap := buffer.scratchPixels(visible.Width)
	stepX := (int64(srcWidth) << 16) / int64(rect.Width)
	srcXFixed := int64(visible.X-rect.X) * stepX
	for index := range xMap {
		srcX := int(srcXFixed >> 16)
		if srcX < 0 {
			srcX = 0
		} else if srcX >= srcWidth {
			srcX = srcWidth - 1
		}
		xMap[index] = uint32(srcX)
		srcXFixed += stepX
	}
	stepY := (int64(srcHeight) << 16) / int64(rect.Height)
	srcYFixed := int64(visible.Y-rect.Y) * stepY
	for row := 0; row < visible.Height; row++ {
		srcY := int(srcYFixed >> 16)
		if srcY < 0 {
			srcY = 0
		} else if srcY >= srcHeight {
			srcY = srcHeight - 1
		}
		srcRow := srcY * srcWidth
		dstIndex := 2 + (visible.Y+row)*buffer.width + visible.X
		drawMappedImageSpan(buffer, dstIndex, pixels[srcRow:srcRow+srcWidth], xMap, srcOpaque)
		srcYFixed += stepY
	}
}

func (buffer *Buffer) DrawImagePixelsRotatedScaled(anchorX float64, anchorY float64, srcWidth int, srcHeight int, pixels []uint32, srcOpaque bool, angle float64, scaleX float64, scaleY float64, pivotX float64, pivotY float64) {
	if buffer == nil || srcWidth <= 0 || srcHeight <= 0 || len(pixels) < srcWidth*srcHeight || scaleX == 0 || scaleY == 0 {
		return
	}
	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	minX, minY, maxX, maxY := rotatedPixelsBounds(anchorX, anchorY, srcWidth, srcHeight, pivotX, pivotY, scaleX, scaleY, cosAngle, sinAngle)
	bounds := Rect{
		X:      int(math.Floor(minX)),
		Y:      int(math.Floor(minY)),
		Width:  int(math.Ceil(maxX)) - int(math.Floor(minX)),
		Height: int(math.Ceil(maxY)) - int(math.Floor(minY)),
	}
	dstBounds := buffer.Bounds()
	if buffer.clip.set {
		dstBounds = IntersectRect(dstBounds, buffer.clip.rect)
	}
	visible := IntersectRect(bounds, dstBounds)
	if visible.Empty() {
		return
	}

	invScaleX := 1.0 / scaleX
	invScaleY := 1.0 / scaleY
	stepXX := cosAngle * invScaleX
	stepXY := -sinAngle * invScaleY
	stepYX := sinAngle * invScaleX
	stepYY := cosAngle * invScaleY

	localXStart := float64(visible.X) + 0.5 - anchorX
	localYStart := float64(visible.Y) + 0.5 - anchorY
	rowSrcX := localXStart*stepXX + localYStart*stepYX + pivotX
	rowSrcY := localXStart*stepXY + localYStart*stepYY + pivotY
	for dstY := visible.Y; dstY < visible.Y+visible.Height; dstY++ {
		dstIndex := 2 + dstY*buffer.width + visible.X
		srcX := rowSrcX
		srcY := rowSrcY
		for col := 0; col < visible.Width; col++ {
			sx := int(srcX)
			sy := int(srcY)
			if uint(sx) < uint(srcWidth) && uint(sy) < uint(srcHeight) {
				pixel := pixels[sy*srcWidth+sx]
				blendImagePixel(buffer, dstIndex+col, pixel, srcOpaque)
			}
			srcX += stepXX
			srcY += stepXY
		}
		rowSrcX += stepYX
		rowSrcY += stepYY
	}
}

func drawImageSpan(buffer *Buffer, dstIndex int, src []uint32, srcOpaque bool) {
	if srcOpaque {
		copy32(buffer.data[dstIndex:dstIndex+len(src)], src)
		return
	}
	for col, srcPixel := range src {
		blendImagePixel(buffer, dstIndex+col, srcPixel, false)
	}
}

func drawMappedImageSpan(buffer *Buffer, dstIndex int, srcRow []uint32, xMap []uint32, srcOpaque bool) {
	if srcOpaque {
		for col, srcX := range xMap {
			buffer.data[dstIndex+col] = srcRow[int(srcX)]
		}
		return
	}
	for col, srcX := range xMap {
		blendImagePixel(buffer, dstIndex+col, srcRow[int(srcX)], false)
	}
}

func blendImagePixel(buffer *Buffer, dstIndex int, srcPixel uint32, srcOpaque bool) {
	if srcOpaque {
		buffer.data[dstIndex] = srcPixel
		return
	}
	sa := uint8(srcPixel >> 24)
	if sa == 0 {
		return
	}
	if sa >= 255 {
		buffer.data[dstIndex] = srcPixel
		return
	}
	if buffer.alpha {
		buffer.data[dstIndex] = blendPremultiplied(buffer.data[dstIndex], srcPixel)
		return
	}
	buffer.data[dstIndex] = blendPremultipliedOpaque(buffer.data[dstIndex], srcPixel)
}

func rotatedPixelsBounds(anchorX float64, anchorY float64, width int, height int, pivotX float64, pivotY float64, scaleX float64, scaleY float64, cosAngle float64, sinAngle float64) (float64, float64, float64, float64) {
	corners := [4][2]float64{
		{-pivotX * scaleX, -pivotY * scaleY},
		{(float64(width) - pivotX) * scaleX, -pivotY * scaleY},
		{(float64(width) - pivotX) * scaleX, (float64(height) - pivotY) * scaleY},
		{-pivotX * scaleX, (float64(height) - pivotY) * scaleY},
	}

	minX := anchorX
	minY := anchorY
	maxX := anchorX
	maxY := anchorY
	for index, corner := range corners {
		rotatedX := anchorX + corner[0]*cosAngle - corner[1]*sinAngle
		rotatedY := anchorY + corner[0]*sinAngle + corner[1]*cosAngle
		if index == 0 || rotatedX < minX {
			minX = rotatedX
		}
		if index == 0 || rotatedY < minY {
			minY = rotatedY
		}
		if index == 0 || rotatedX > maxX {
			maxX = rotatedX
		}
		if index == 0 || rotatedY > maxY {
			maxY = rotatedY
		}
	}
	return minX, minY, maxX, maxY
}
