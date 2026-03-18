package ui

type TextMeasurer interface {
	FontForStyle(style Style) (*ttfFont, fontMetrics)
	MeasureText(text string, font *ttfFont, charWidth int) int
}

type systemTextMeasurer struct{}

func (systemTextMeasurer) FontForStyle(style Style) (*ttfFont, fontMetrics) {
	return fontAndMetricsForStyle(style)
}

func (systemTextMeasurer) MeasureText(text string, font *ttfFont, charWidth int) int {
	return textWidthWithFont(text, font, charWidth)
}

type LayoutContext struct {
	Viewport Rect
	Text     TextMeasurer
}

func DefaultLayoutContext(viewport Rect) LayoutContext {
	return LayoutContext{
		Viewport: viewport,
		Text:     systemTextMeasurer{},
	}
}

func layoutContextWithViewport(ctx LayoutContext, viewport Rect) LayoutContext {
	ctx.Viewport = viewport
	if ctx.Text == nil {
		ctx.Text = systemTextMeasurer{}
	}
	return ctx
}

func (ctx LayoutContext) textMeasurer() TextMeasurer {
	if ctx.Text != nil {
		return ctx.Text
	}
	return systemTextMeasurer{}
}

func (ctx LayoutContext) FontForStyle(style Style) (*ttfFont, fontMetrics) {
	return ctx.textMeasurer().FontForStyle(style)
}

func (ctx LayoutContext) MeasureText(text string, font *ttfFont, charWidth int) int {
	return ctx.textMeasurer().MeasureText(text, font, charWidth)
}

func layoutContextForCanvas(canvas *Canvas) LayoutContext {
	if canvas == nil {
		return DefaultLayoutContext(Rect{})
	}
	return DefaultLayoutContext(Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()})
}

func (window *Window) layoutContext() LayoutContext {
	if window == nil {
		return DefaultLayoutContext(Rect{})
	}
	if surface := window.surface(); surface != nil {
		return DefaultLayoutContext(surface.Bounds())
	}
	return DefaultLayoutContext(window.client)
}
