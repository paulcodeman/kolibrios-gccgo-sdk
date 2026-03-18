package ui

import "kos"

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
		if shadow, ok := resolveShadow(style.Shadow); ok {
			if borderRadius.Active() {
				canvas.DrawShadowRounded(rect, *shadow, borderRadius)
			} else {
				canvas.DrawShadow(rect, *shadow)
			}
		}
	}

	gradient, gradientSet := resolveGradient(style.Gradient)
	if FastNoGradients {
		gradientSet = false
	}
	background, backgroundSet := resolveColor(style.Background)
	if !backgroundSet && fallback != nil {
		background = *fallback
		backgroundSet = true
	}
	if gradientSet {
		if opacity, ok := resolveOpacity(style.Opacity); ok && opacity < 255 {
			canvas.FillRoundedRectGradientAreaAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, backgroundRect, opacity)
		} else {
			canvas.FillRoundedRectGradientArea(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, *gradient, backgroundRect)
		}
	} else if backgroundSet {
		if opacity, ok := resolveOpacity(style.Opacity); ok && opacity < 255 {
			canvas.FillRoundedRectAlpha(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background, opacity)
		} else {
			canvas.FillRoundedRect(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, background)
		}
	}

	if !FastNoBorders {
		if borderWidth, ok := resolveLength(style.BorderWidth); ok && borderWidth > 0 {
			borderColor := kos.Color(0)
			if value, ok := resolveColor(style.BorderColor); ok {
				borderColor = value
			}
			canvas.StrokeRoundedRectWidth(rect.X, rect.Y, rect.Width, rect.Height, borderRadius, borderWidth, borderColor)
		}
	}
}
