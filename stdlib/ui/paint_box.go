package ui

import "kos"

func drawStyledBorder(canvas *Canvas, rect Rect, style Style, borderRadius CornerRadii) {
	if canvas == nil || rect.Empty() || FastNoBorders {
		return
	}
	widths := borderWidthsFor(style)
	if !borderWidthsAny(widths) {
		return
	}
	topColor, rightColor, bottomColor, leftColor, colorsSet := borderColorsFor(style)
	if !colorsSet {
		topColor = kos.Color(0)
		rightColor = topColor
		bottomColor = topColor
		leftColor = topColor
	}
	if width, color, ok := uniformBorderStyle(style); ok && width > 0 {
		canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, width, color)
		return
	}
	if widths.Top > 0 {
		canvas.FillRect(rect.X, rect.Y, rect.Width, widths.Top, topColor)
	}
	if widths.Bottom > 0 {
		canvas.FillRect(rect.X, rect.Y+rect.Height-widths.Bottom, rect.Width, widths.Bottom, bottomColor)
	}
	sideHeight := rect.Height - widths.Top - widths.Bottom
	if sideHeight < 0 {
		sideHeight = 0
	}
	if widths.Left > 0 && sideHeight > 0 {
		canvas.FillRect(rect.X, rect.Y+widths.Top, widths.Left, sideHeight, leftColor)
	}
	if widths.Right > 0 && sideHeight > 0 {
		canvas.FillRect(rect.X+rect.Width-widths.Right, rect.Y+widths.Top, widths.Right, sideHeight, rightColor)
	}
}

func outlineRadii(style Style, expand int) CornerRadii {
	if radius := outlineRadiusFor(style); radius > 0 {
		return CornerRadii{
			TopLeft:     radius,
			TopRight:    radius,
			BottomRight: radius,
			BottomLeft:  radius,
		}
	}
	radii := resolveBorderRadius(style)
	if !radii.Active() || expand <= 0 {
		return radii
	}
	return CornerRadii{
		TopLeft:     radii.TopLeft + expand,
		TopRight:    radii.TopRight + expand,
		BottomRight: radii.BottomRight + expand,
		BottomLeft:  radii.BottomLeft + expand,
	}
}

func drawStyledOutline(canvas *Canvas, rect Rect, style Style) {
	if canvas == nil || rect.Empty() {
		return
	}
	width := outlineWidthFor(style)
	if width <= 0 {
		return
	}
	color, ok := outlineColorFor(style)
	if !ok {
		if foreground, foregroundOK := resolveColor(style.foreground); foregroundOK {
			color = foreground
		} else {
			color = defaultFocusRingColor
		}
	}
	offset := outlineOffsetFor(style)
	expand := width + offset
	ring := Rect{
		X:      rect.X - expand,
		Y:      rect.Y - expand,
		Width:  rect.Width + expand*2,
		Height: rect.Height + expand*2,
	}
	canvas.StrokeRoundedRectWidth(ring.X, ring.Y, ring.Width, ring.Height, outlineRadii(style, expand), width, color)
}

func drawStyledBox(canvas *Canvas, rect Rect, style Style, backgroundRect Rect, fallback *kos.Color) {
	if canvas == nil || rect.Empty() {
		return
	}
	if backgroundRect.Empty() {
		backgroundRect = rect
	}
	borderRadius := resolveBorderRadius(style)
	if FastNoRadius {
		borderRadius = CornerRadii{}
	}
	if !FastNoShadows {
		if shadow, ok := resolveShadow(style.shadow); ok {
			if borderRadius.Active() {
				canvas.DrawShadowRounded(rect, *shadow, borderRadius)
			} else {
				canvas.DrawShadow(rect, *shadow)
			}
		}
	}

	gradient, gradientSet := resolveGradient(style.gradient)
	if FastNoGradients {
		gradientSet = false
	}
	background, backgroundSet := resolveColor(style.background)
	if !backgroundSet && fallback != nil {
		background = *fallback
		backgroundSet = true
	}
	if gradientSet {
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			canvas.FillRoundedRectGradientAreaAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, backgroundRect, opacity)
		} else {
			canvas.FillRoundedRectGradientArea(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, backgroundRect)
		}
	} else if backgroundSet {
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			canvas.FillRoundedRectAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background, opacity)
		} else {
			canvas.FillRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background)
		}
	}

	drawStyledBorder(canvas, rect, style, borderRadius)
	drawStyledOutline(canvas, rect, style)
}
