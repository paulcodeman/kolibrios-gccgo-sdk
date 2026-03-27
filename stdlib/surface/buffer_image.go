package surface

func (buffer *Buffer) DrawImage(x int, y int, image *Image) {
	if image == nil {
		return
	}
	buffer.DrawImageRect(Rect{X: x, Y: y, Width: image.Width, Height: image.Height}, image)
}

func (buffer *Buffer) DrawImageRect(rect Rect, image *Image) {
	raw := rawBuffer(buffer)
	if raw == nil || image == nil || !image.Valid() || rect.Empty() {
		return
	}
	visible := IntersectRect(rect, Rect{X: 0, Y: 0, Width: buffer.Width(), Height: buffer.Height()})
	if visible.Empty() || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	srcWidth := image.Width
	srcHeight := image.Height
	for dstY := visible.Y; dstY < visible.Y+visible.Height; dstY++ {
		srcY := ((dstY - rect.Y) * srcHeight) / rect.Height
		if srcY < 0 {
			srcY = 0
		} else if srcY >= srcHeight {
			srcY = srcHeight - 1
		}
		row := srcY * srcWidth
		for dstX := visible.X; dstX < visible.X+visible.Width; dstX++ {
			srcX := ((dstX - rect.X) * srcWidth) / rect.Width
			if srcX < 0 {
				srcX = 0
			} else if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			raw.BlendPremultipliedPixelValue(dstX, dstY, image.Pixels[row+srcX])
		}
	}
}
