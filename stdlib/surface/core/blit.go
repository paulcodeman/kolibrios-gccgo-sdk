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
		if !buffer.alpha {
			for row := 0; row < dstRect.Height; row++ {
				dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
				srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
				for col := 0; col < dstRect.Width; col++ {
					srcPixel := src.data[srcIndex+col]
					if srcPixel>>24 == 0 {
						continue
					}
					buffer.data[dstIndex+col] = blendPremultipliedOpaque(buffer.data[dstIndex+col], srcPixel)
				}
			}
			return
		}
		for row := 0; row < dstRect.Height; row++ {
			dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
			srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
			for col := 0; col < dstRect.Width; col++ {
				srcPixel := src.data[srcIndex+col]
				sa := srcPixel >> 24
				if sa == 0 {
					continue
				}
				if sa >= 255 {
					buffer.data[dstIndex+col] = srcPixel
					continue
				}
				buffer.data[dstIndex+col] = blendPremultiplied(buffer.data[dstIndex+col], srcPixel)
			}
		}
		return
	}
	for row := 0; row < dstRect.Height; row++ {
		dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
		srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
		copy32(buffer.data[dstIndex:dstIndex+dstRect.Width], src.data[srcIndex:srcIndex+dstRect.Width])
	}
}

func (buffer *Buffer) BlitSelf(srcRect Rect, dstX int, dstY int) {
	if buffer == nil || srcRect.Empty() {
		return
	}
	srcRect = IntersectRect(srcRect, buffer.Bounds())
	if srcRect.Empty() {
		return
	}
	dstRect := Rect{X: dstX, Y: dstY, Width: srcRect.Width, Height: srcRect.Height}
	dstRect = IntersectRect(dstRect, buffer.Bounds())
	if dstRect.Empty() {
		return
	}
	dx := dstRect.X - dstX
	dy := dstRect.Y - dstY
	srcRect.X += dx
	srcRect.Y += dy
	srcRect.Width = dstRect.Width
	srcRect.Height = dstRect.Height
	if srcRect == dstRect {
		return
	}
	if dstRect.Y > srcRect.Y {
		for row := dstRect.Height - 1; row >= 0; row-- {
			srcIndex := 2 + (srcRect.Y+row)*buffer.width + srcRect.X
			dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
			move32(buffer.data[dstIndex:dstIndex+dstRect.Width], buffer.data[srcIndex:srcIndex+dstRect.Width])
		}
		return
	}
	for row := 0; row < dstRect.Height; row++ {
		srcIndex := 2 + (srcRect.Y+row)*buffer.width + srcRect.X
		dstIndex := 2 + (dstRect.Y+row)*buffer.width + dstRect.X
		move32(buffer.data[dstIndex:dstIndex+dstRect.Width], buffer.data[srcIndex:srcIndex+dstRect.Width])
	}
}

func (buffer *Buffer) ScrollRectY(rect Rect, deltaY int) {
	if buffer == nil || rect.Empty() || deltaY == 0 {
		return
	}
	rect = IntersectRect(rect, buffer.Bounds())
	if rect.Empty() {
		return
	}
	if deltaY >= rect.Height || deltaY <= -rect.Height {
		return
	}
	if deltaY > 0 {
		for row := rect.Height - deltaY - 1; row >= 0; row-- {
			srcY := rect.Y + row
			dstY := srcY + deltaY
			srcIndex := 2 + srcY*buffer.width + rect.X
			dstIndex := 2 + dstY*buffer.width + rect.X
			move32(buffer.data[dstIndex:dstIndex+rect.Width], buffer.data[srcIndex:srcIndex+rect.Width])
		}
		return
	}
	shift := -deltaY
	for row := shift; row < rect.Height; row++ {
		srcY := rect.Y + row
		dstY := srcY - shift
		srcIndex := 2 + srcY*buffer.width + rect.X
		dstIndex := 2 + dstY*buffer.width + rect.X
		move32(buffer.data[dstIndex:dstIndex+rect.Width], buffer.data[srcIndex:srcIndex+rect.Width])
	}
}
