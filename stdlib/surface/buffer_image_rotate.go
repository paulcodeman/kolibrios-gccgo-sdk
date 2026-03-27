package surface

import "math"

func (buffer *Buffer) DrawImageRotated(anchorX float64, anchorY float64, image *Image, angle float64, pivotX float64, pivotY float64) {
	buffer.DrawImageRotatedScaled(anchorX, anchorY, image, angle, 1, 1, pivotX, pivotY)
}

func (buffer *Buffer) DrawImageRotatedScaled(anchorX float64, anchorY float64, image *Image, angle float64, scaleX float64, scaleY float64, pivotX float64, pivotY float64) {
	raw := rawBuffer(buffer)
	if raw == nil || image == nil || !image.Valid() || scaleX == 0 || scaleY == 0 {
		return
	}

	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	minX, minY, maxX, maxY := rotatedImageBounds(anchorX, anchorY, image.Width, image.Height, pivotX, pivotY, scaleX, scaleY, cosAngle, sinAngle)
	bounds := Rect{
		X:      int(math.Floor(minX)),
		Y:      int(math.Floor(minY)),
		Width:  int(math.Ceil(maxX)) - int(math.Floor(minX)),
		Height: int(math.Ceil(maxY)) - int(math.Floor(minY)),
	}
	visible := IntersectRect(bounds, Rect{Width: buffer.Width(), Height: buffer.Height()})
	if visible.Empty() {
		return
	}

	for dstY := visible.Y; dstY < visible.Y+visible.Height; dstY++ {
		localY := float64(dstY) + 0.5 - anchorY
		for dstX := visible.X; dstX < visible.X+visible.Width; dstX++ {
			localX := float64(dstX) + 0.5 - anchorX
			srcX := (localX*cosAngle + localY*sinAngle) / scaleX
			srcY := (-localX*sinAngle + localY*cosAngle) / scaleY
			srcX += pivotX
			srcY += pivotY
			if srcX < 0 || srcY < 0 || srcX >= float64(image.Width) || srcY >= float64(image.Height) {
				continue
			}
			sourceIndex := int(srcY)*image.Width + int(srcX)
			pixel := image.Pixels[sourceIndex]
			if pixel>>24 == 0 {
				continue
			}
			raw.BlendPremultipliedPixelValue(dstX, dstY, pixel)
		}
	}
}

func rotatedImageBounds(anchorX float64, anchorY float64, width int, height int, pivotX float64, pivotY float64, scaleX float64, scaleY float64, cosAngle float64, sinAngle float64) (float64, float64, float64, float64) {
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
