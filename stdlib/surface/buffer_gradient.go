package surface

func (buffer *Buffer) FillRectGradient(x int, y int, width int, height int, gradient Gradient) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRectGradient(x, y, width, height, rawGradient(gradient))
}

func (buffer *Buffer) FillRoundedRectGradient(x int, y int, width int, height int, radii CornerRadii, gradient Gradient) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradient(x, y, width, height, radii, rawGradient(gradient))
}

func (buffer *Buffer) FillRectGradientAlpha(x int, y int, width int, height int, gradient Gradient, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRectGradientAlpha(x, y, width, height, rawGradient(gradient), alpha)
}

func (buffer *Buffer) FillRoundedRectGradientAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientAlpha(x, y, width, height, radii, rawGradient(gradient), alpha)
}

func (buffer *Buffer) FillRectGradientArea(x int, y int, width int, height int, gradient Gradient, area Rect) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRectGradientArea(x, y, width, height, rawGradient(gradient), area)
}

func (buffer *Buffer) FillRoundedRectGradientArea(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientArea(x, y, width, height, radii, rawGradient(gradient), area)
}

func (buffer *Buffer) FillRectGradientAreaAlpha(x int, y int, width int, height int, gradient Gradient, area Rect, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRectGradientAreaAlpha(x, y, width, height, rawGradient(gradient), area, alpha)
}

func (buffer *Buffer) FillRoundedRectGradientAreaAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientAreaAlpha(x, y, width, height, radii, rawGradient(gradient), area, alpha)
}
