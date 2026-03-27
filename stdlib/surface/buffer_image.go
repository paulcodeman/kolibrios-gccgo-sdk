package surface

func (buffer *Buffer) DrawImage(x int, y int, image *Image) {
	if image == nil || !image.Valid() {
		return
	}
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawImagePixels(x, y, image.Width, image.Height, image.Pixels, image.Opaque())
}

func (buffer *Buffer) DrawImageRect(rect Rect, image *Image) {
	raw := rawBuffer(buffer)
	if raw == nil || image == nil || !image.Valid() || rect.Empty() {
		return
	}
	raw.DrawImagePixelsRect(rect, image.Width, image.Height, image.Pixels, image.Opaque())
}
