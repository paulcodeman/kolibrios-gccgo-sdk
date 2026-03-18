package ui

import "kos"

func (canvas *Canvas) BlitToWindow(x int, y int) {
	if canvas == nil || canvas.width <= 0 || canvas.height <= 0 {
		return
	}
	ptr := canvas.pixelPtr(0)
	if ptr == nil {
		return
	}
	kos.PutImage32(ptr, canvas.width, canvas.height, x, y)
}

func (canvas *Canvas) BlitFrom(src *Canvas, srcRect Rect, dstX int, dstY int) {
	if canvas == nil || src == nil {
		return
	}
	if canvas.width <= 0 || canvas.height <= 0 || src.width <= 0 || src.height <= 0 {
		return
	}
	if srcRect.Empty() {
		return
	}
	srcBounds := Rect{X: 0, Y: 0, Width: src.width, Height: src.height}
	srcRect = IntersectRect(srcRect, srcBounds)
	if srcRect.Empty() {
		return
	}
	dstRect := Rect{X: dstX, Y: dstY, Width: srcRect.Width, Height: srcRect.Height}
	dstBounds := Rect{X: 0, Y: 0, Width: canvas.width, Height: canvas.height}
	if canvas.clip.set {
		dstBounds = IntersectRect(dstBounds, canvas.clip.rect)
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
			dstIndex := 2 + (dstRect.Y+row)*canvas.width + dstRect.X
			srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
			for col := 0; col < dstRect.Width; col++ {
				dst := canvas.data[dstIndex+col]
				srcPixel := src.data[srcIndex+col]
				canvas.data[dstIndex+col] = blendPremultiplied(dst, srcPixel)
			}
		}
		return
	}
	for row := 0; row < dstRect.Height; row++ {
		dstIndex := 2 + (dstRect.Y+row)*canvas.width + dstRect.X
		srcIndex := 2 + (srcRect.Y+row)*src.width + srcRect.X
		copy(canvas.data[dstIndex:dstIndex+dstRect.Width], src.data[srcIndex:srcIndex+dstRect.Width])
	}
}

func (canvas *Canvas) BlitRectToWindow(rect Rect, x int, y int) {
	if canvas == nil || canvas.width <= 0 || canvas.height <= 0 {
		return
	}
	if rect.Empty() {
		return
	}
	cx, cy, cw, ch, ok := canvas.clampRect(rect.X, rect.Y, rect.Width, rect.Height)
	if !ok {
		return
	}
	dx := cx - rect.X
	dy := cy - rect.Y
	x += dx
	y += dy
	ptr := canvas.pixelPtr(cy*canvas.width + cx)
	if ptr == nil {
		return
	}
	rowOffset := (canvas.width - cw) * 4
	kos.PutPaletteImage(ptr, cw, ch, x, y, 32, nil, rowOffset)
}

func (canvas *Canvas) ScrollRectY(rect Rect, deltaY int) {
	if canvas == nil || canvas.width <= 0 || canvas.height <= 0 || rect.Empty() || deltaY == 0 {
		return
	}
	bounds := Rect{X: 0, Y: 0, Width: canvas.width, Height: canvas.height}
	rect = IntersectRect(rect, bounds)
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
			srcIndex := 2 + srcY*canvas.width + rect.X
			dstIndex := 2 + dstY*canvas.width + rect.X
			copy(canvas.data[dstIndex:dstIndex+rect.Width], canvas.data[srcIndex:srcIndex+rect.Width])
		}
		return
	}
	shift := -deltaY
	for row := shift; row < rect.Height; row++ {
		srcY := rect.Y + row
		dstY := srcY - shift
		srcIndex := 2 + srcY*canvas.width + rect.X
		dstIndex := 2 + dstY*canvas.width + rect.X
		copy(canvas.data[dstIndex:dstIndex+rect.Width], canvas.data[srcIndex:srcIndex+rect.Width])
	}
}
