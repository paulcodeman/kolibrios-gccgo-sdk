package ui

type fontMetrics struct {
	width  int
	height int
	ascent int
}

func defaultFontMetrics() fontMetrics {
	return fontMetrics{
		width:  defaultCharWidth,
		height: defaultFontHeight,
		ascent: defaultFontHeight * 3 / 4,
	}
}

func metricsForStyle(style Style) fontMetrics {
	if font := fontForStyle(style); font != nil {
		return font.metrics
	}
	return defaultFontMetrics()
}

func fontAndMetricsForStyle(style Style) (*ttfFont, fontMetrics) {
	font := fontForStyle(style)
	if font != nil {
		return font, font.metrics
	}
	return nil, defaultFontMetrics()
}
