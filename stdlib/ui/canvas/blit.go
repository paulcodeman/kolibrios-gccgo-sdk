package ui

func (canvas *Canvas) BlitToWindow(x int, y int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.BlitToWindow(x, y)
}

func (canvas *Canvas) BlitFrom(src *Canvas, srcRect Rect, dstX int, dstY int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.BlitFrom(surfaceBuffer(src), srcRect, dstX, dstY)
}

func (canvas *Canvas) BlitSelf(srcRect Rect, dstX int, dstY int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.BlitSelf(srcRect, dstX, dstY)
}

func (canvas *Canvas) BlitRectToWindow(rect Rect, x int, y int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.BlitRectToWindow(rect, x, y)
}

func (canvas *Canvas) ScrollRectY(rect Rect, deltaY int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.ScrollRectY(rect, deltaY)
}
