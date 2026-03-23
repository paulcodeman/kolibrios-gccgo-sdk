package ui

import "kos"

const (
	controlIndicatorGap    = 8
	controlIndicatorMin    = 14
	progressTrackMinHeight = 8
	rangeTrackHeight       = 4
	rangeThumbSize         = 14
)

func controlAccentColor(style Style) kos.Color {
	if value, ok := resolveColor(style.foreground); ok {
		return value
	}
	return Blue
}

func controlBorderColor(style Style) kos.Color {
	if value, ok := resolveColor(style.borderColor); ok {
		return value
	}
	return Gray
}

func controlSurfaceColor(style Style) kos.Color {
	if value, ok := resolveColor(style.background); ok {
		return value
	}
	return White
}

func (element *Element) controlIndicatorSize(style Style) int {
	_, metrics := fontAndMetricsForStyle(style)
	return controlIndicatorSizeForLineHeight(lineHeightForStyle(style, metrics.height))
}

func (element *Element) checkableIndicatorRect(rect Rect, style Style) Rect {
	content := contentRectFor(rect, style)
	size := element.controlIndicatorSize(style)
	if size > content.Height {
		size = content.Height
	}
	if size <= 0 {
		return Rect{}
	}
	y := content.Y
	if content.Height > size {
		y += (content.Height - size) / 2
	}
	return Rect{X: content.X, Y: y, Width: size, Height: size}
}

func (element *Element) labeledControlTextRect(rect Rect, style Style) Rect {
	content := contentRectFor(rect, style)
	if element == nil || !element.isCheckable() {
		return content
	}
	indicator := element.checkableIndicatorRect(rect, style)
	content.X = indicator.X + indicator.Width + controlIndicatorGap
	content.Width -= indicator.Width + controlIndicatorGap
	if content.Width < 0 {
		content.Width = 0
	}
	return content
}

func drawCheckboxMark(canvas *Canvas, rect Rect, color kos.Color) {
	if canvas == nil || rect.Empty() {
		return
	}
	left := rect.X + rect.Width/5
	midY := rect.Y + rect.Height/2
	step := rect.Width / 5
	if step < 2 {
		step = 2
	}
	thickness := rect.Width / 6
	if thickness < 2 {
		thickness = 2
	}
	canvas.FillRect(left, midY, thickness, thickness, color)
	canvas.FillRect(left+step, midY+step/2, thickness, thickness, color)
	canvas.FillRect(left+step*2, midY-step/2, thickness, thickness, color)
	canvas.FillRect(left+step*3, midY-step, thickness, thickness, color)
}

func (element *Element) drawControlText(canvas *Canvas, rect Rect, style Style) {
	if element == nil || canvas == nil || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	text := element.text()
	if text == "" || FastNoText {
		return
	}
	foreground, ok := resolveColor(style.foreground)
	if !ok {
		foreground = Black
	}
	font, metrics := fontAndMetricsForStyle(style)
	charWidth := metrics.width
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	lines := element.wrapTextLinesCachedStyle(text, rect.Width, font, charWidth, style)
	if len(lines) == 0 {
		return
	}
	ensureTextLineMetrics(lines, font, charWidth)
	lineHeight := lineHeightForStyle(style, metrics.height)
	totalHeight := len(lines) * lineHeight
	startY := rect.Y
	if rect.Height > totalHeight {
		startY += (rect.Height - totalHeight) / 2
	}
	for index, line := range lines {
		lineRect := Rect{X: rect.X, Y: startY + index*lineHeight, Width: rect.Width, Height: lineHeight}
		textX := textLineXForWidth(lineRect, style, 0, 0, rect.Width, line.width)
		textY := lineRect.Y
		if font != nil {
			canvas.DrawTextFont(textX, textY, foreground, line.text, font)
		} else {
			canvas.DrawText(textX, textY, foreground, line.text)
		}
		drawTextDecorations(canvas, textX, textY, line.text, style, font, charWidth, foreground)
	}
}

func (element *Element) progressBarRect(rect Rect, style Style) Rect {
	content := contentRectFor(rect, style)
	if content.Empty() {
		return Rect{}
	}
	height := content.Height
	if height > 12 {
		height = 12
	}
	if height < progressTrackMinHeight {
		height = progressTrackMinHeight
	}
	if height > content.Height {
		height = content.Height
	}
	y := content.Y
	if content.Height > height {
		y += (content.Height - height) / 2
	}
	return Rect{X: content.X, Y: y, Width: content.Width, Height: height}
}

func (element *Element) rangeTrackRect(rect Rect, style Style) Rect {
	content := contentRectFor(rect, style)
	if content.Empty() {
		return Rect{}
	}
	thumb := rangeThumbSize
	if thumb > content.Height {
		thumb = content.Height
	}
	if thumb < 10 {
		thumb = 10
	}
	trackX := content.X + thumb/2
	trackW := content.Width - thumb
	if trackW < 0 {
		trackW = 0
	}
	y := content.Y + (content.Height-rangeTrackHeight)/2
	return Rect{X: trackX, Y: y, Width: trackW, Height: rangeTrackHeight}
}

func (element *Element) rangeThumbRect(rect Rect, style Style) Rect {
	content := contentRectFor(rect, style)
	if content.Empty() {
		return Rect{}
	}
	thumb := rangeThumbSize
	if thumb > content.Height {
		thumb = content.Height
	}
	if thumb < 10 {
		thumb = 10
	}
	track := element.rangeTrackRect(rect, style)
	x := content.X
	if track.Width > 0 {
		x = track.X - thumb/2 + int(element.ValueFraction()*float64(track.Width))
	}
	y := content.Y + (content.Height-thumb)/2
	return Rect{X: x, Y: y, Width: thumb, Height: thumb}
}

func (element *Element) drawCheckboxOrRadio(canvas *Canvas, rect Rect, style Style) {
	indicator := element.checkableIndicatorRect(rect, style)
	if indicator.Empty() {
		return
	}
	border := controlBorderColor(style)
	background := controlSurfaceColor(style)
	accent := controlAccentColor(style)
	radius := CornerRadii{}
	if element.isRadio() {
		radius = CornerRadii{TopLeft: indicator.Width / 2, TopRight: indicator.Width / 2, BottomRight: indicator.Width / 2, BottomLeft: indicator.Width / 2}
	} else {
		radius = CornerRadii{TopLeft: 4, TopRight: 4, BottomRight: 4, BottomLeft: 4}
	}
	canvas.FillRoundedRect(indicator.X, indicator.Y, indicator.Width, indicator.Height, radius, background)
	canvas.StrokeRoundedRectWidth(indicator.X, indicator.Y, indicator.Width, indicator.Height, radius, 1, border)
	if element.checked {
		if element.isRadio() {
			inset := indicator.Width / 4
			if inset < 3 {
				inset = 3
			}
			inner := Rect{
				X:      indicator.X + inset,
				Y:      indicator.Y + inset,
				Width:  indicator.Width - inset*2,
				Height: indicator.Height - inset*2,
			}
			innerRadius := CornerRadii{TopLeft: inner.Width / 2, TopRight: inner.Width / 2, BottomRight: inner.Width / 2, BottomLeft: inner.Width / 2}
			canvas.FillRoundedRect(inner.X, inner.Y, inner.Width, inner.Height, innerRadius, accent)
		} else {
			drawCheckboxMark(canvas, indicator, accent)
		}
	}
	element.drawControlText(canvas, element.labeledControlTextRect(rect, style), style)
}

func (element *Element) drawProgress(canvas *Canvas, rect Rect, style Style) {
	bar := element.progressBarRect(rect, style)
	if bar.Empty() {
		return
	}
	track := controlSurfaceColor(style)
	border := controlBorderColor(style)
	fill := controlAccentColor(style)
	radius := CornerRadii{TopLeft: bar.Height / 2, TopRight: bar.Height / 2, BottomRight: bar.Height / 2, BottomLeft: bar.Height / 2}
	canvas.FillRoundedRect(bar.X, bar.Y, bar.Width, bar.Height, radius, track)
	canvas.StrokeRoundedRectWidth(bar.X, bar.Y, bar.Width, bar.Height, radius, 1, border)
	fillWidth := int(float64(bar.Width) * element.ValueFraction())
	if fillWidth > 0 {
		canvas.FillRoundedRect(bar.X, bar.Y, fillWidth, bar.Height, radius, fill)
	}
}

func (element *Element) drawRange(canvas *Canvas, rect Rect, style Style) {
	content := contentRectFor(rect, style)
	track := element.rangeTrackRect(rect, style)
	thumb := element.rangeThumbRect(rect, style)
	if content.Empty() || track.Empty() || thumb.Empty() {
		return
	}
	trackRadius := CornerRadii{TopLeft: track.Height / 2, TopRight: track.Height / 2, BottomRight: track.Height / 2, BottomLeft: track.Height / 2}
	accent := controlAccentColor(style)
	trackColor := Silver
	canvas.FillRoundedRect(track.X, track.Y, track.Width, track.Height, trackRadius, trackColor)
	if filledWidth := thumb.X + thumb.Width/2 - track.X; filledWidth > 0 {
		if filledWidth > track.Width {
			filledWidth = track.Width
		}
		canvas.FillRoundedRect(track.X, track.Y, filledWidth, track.Height, trackRadius, accent)
	}
	thumbRadius := CornerRadii{TopLeft: thumb.Width / 2, TopRight: thumb.Width / 2, BottomRight: thumb.Width / 2, BottomLeft: thumb.Width / 2}
	canvas.FillRoundedRect(thumb.X, thumb.Y, thumb.Width, thumb.Height, thumbRadius, White)
	canvas.StrokeRoundedRectWidth(thumb.X, thumb.Y, thumb.Width, thumb.Height, thumbRadius, 1, accent)
}

func (element *Element) drawControlToRect(canvas *Canvas, rect Rect, style Style) {
	if element == nil || canvas == nil {
		return
	}
	switch {
	case element.isCheckable():
		element.drawCheckboxOrRadio(canvas, rect, style)
	case element.isProgress():
		element.drawProgress(canvas, rect, style)
	case element.isRange():
		element.drawRange(canvas, rect, style)
	}
	if elementShowsDefaultFocusRing(element) {
		drawDefaultFocusRing(canvas, rect, style)
	}
}
