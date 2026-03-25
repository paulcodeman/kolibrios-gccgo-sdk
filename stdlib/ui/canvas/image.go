package ui

func (canvas *Canvas) DrawDocumentImage(rect Rect, image *DocumentImage) {
	if canvas == nil || image == nil || !image.Valid() || rect.Empty() {
		return
	}
	buffer := surfaceBuffer(canvas)
	if buffer == nil {
		return
	}
	buffer.DrawImageRect(rect, image)
}
