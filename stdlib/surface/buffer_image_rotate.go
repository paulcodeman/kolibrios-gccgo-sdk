package surface

func (buffer *Buffer) DrawImageRotated(anchorX float64, anchorY float64, image *Image, angle float64, pivotX float64, pivotY float64) {
	buffer.DrawImageRotatedScaled(anchorX, anchorY, image, angle, 1, 1, pivotX, pivotY)
}

func (buffer *Buffer) DrawImageRotatedScaled(anchorX float64, anchorY float64, image *Image, angle float64, scaleX float64, scaleY float64, pivotX float64, pivotY float64) {
	raw := rawBuffer(buffer)
	if raw == nil || image == nil || !image.Valid() || scaleX == 0 || scaleY == 0 {
		return
	}
	raw.DrawImagePixelsRotatedScaled(anchorX, anchorY, image.Width, image.Height, image.Pixels, image.Opaque(), angle, scaleX, scaleY, pivotX, pivotY)
}
