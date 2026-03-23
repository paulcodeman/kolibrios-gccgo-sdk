package ui

func (canvas *Canvas) FillRectGradient(x int, y int, width int, height int, gradient Gradient) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRectGradient(x, y, width, height, gradient)
}

func (canvas *Canvas) FillRoundedRectGradient(x int, y int, width int, height int, radii CornerRadii, gradient Gradient) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradient(x, y, width, height, radii, gradient)
}

func (canvas *Canvas) FillRectGradientAlpha(x int, y int, width int, height int, gradient Gradient, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRectGradientAlpha(x, y, width, height, gradient, alpha)
}

func (canvas *Canvas) FillRoundedRectGradientAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientAlpha(x, y, width, height, radii, gradient, alpha)
}

func (canvas *Canvas) FillRectGradientArea(x int, y int, width int, height int, gradient Gradient, area Rect) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRectGradientArea(x, y, width, height, gradient, area)
}

func (canvas *Canvas) FillRoundedRectGradientArea(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientArea(x, y, width, height, radii, gradient, area)
}

func (canvas *Canvas) FillRectGradientAreaAlpha(x int, y int, width int, height int, gradient Gradient, area Rect, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRectGradientAreaAlpha(x, y, width, height, gradient, area, alpha)
}

func (canvas *Canvas) FillRoundedRectGradientAreaAlpha(x int, y int, width int, height int, radii CornerRadii, gradient Gradient, area Rect, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRectGradientAreaAlpha(x, y, width, height, radii, gradient, area, alpha)
}
