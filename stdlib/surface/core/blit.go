package core

func (buffer *Buffer) BlitFrom(src *Buffer, srcRect Rect, dstX int, dstY int) {
	if buffer == nil || src == nil || srcRect.Empty() {
		return
	}
	srcRect = IntersectRect(srcRect, src.Bounds())
	if srcRect.Empty() {
		return
	}
	dstRect := Rect{X: dstX, Y: dstY, Width: srcRect.Width, Height: srcRect.Height}
	dstBounds := buffer.Bounds()
	if buffer.clip.set {
		dstBounds = IntersectRect(dstBounds, buffer.clip.rect)
	}
	dstRect = IntersectRect(dstRect, dstBounds)
	if dstRect.Empty() {
		return
	}
	dx := dstRect.X - dstX
	dy := dstRect.Y - dstY
	srcRect.X += dx
	srcRect.Y += dy
	srcRect.Width = dstRect.Width
	srcRect.Height = dstRect.Height
	if src.alpha {
		for row := 0; row < dstRect.Height; row++ {
			dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
			srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
			for col := 0; col < dstRect.Width; col++ {
				buffer.data[dstIndex+col] = blendPremultiplied(buffer.data[dstIndex+col], src.data[srcIndex+col])
			}
		}
		return
	}
	for row := 0; row < dstRect.Height; row++ {
		dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
		srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
		copy(buffer.data[dstIndex:dstIndex+dstRect.Width], src.data[srcIndex:srcIndex+dstRect.Width])
	}
}
