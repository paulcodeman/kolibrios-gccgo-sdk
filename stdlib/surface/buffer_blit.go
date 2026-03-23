package surface

func (buffer *Buffer) BlitFrom(src *Buffer, srcRect Rect, dstX int, dstY int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.BlitFrom(rawBuffer(src), srcRect, dstX, dstY)
}

func (buffer *Buffer) BlitSelf(srcRect Rect, dstX int, dstY int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.BlitSelf(srcRect, dstX, dstY)
}

func (buffer *Buffer) ScrollRectY(rect Rect, deltaY int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.ScrollRectY(rect, deltaY)
}

func (buffer *Buffer) BlitToWindow(x int, y int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.BlitToWindow(x, y)
}

func (buffer *Buffer) BlitRectToWindow(rect Rect, x int, y int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.BlitRectToWindow(rect, x, y)
}
