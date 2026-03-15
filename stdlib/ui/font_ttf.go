package ui

import (
	"os"
	"sync"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type ttfFont struct {
	path    string
	size    int
	face    xfont.Face
	metrics fontMetrics
}

type fontKey struct {
	path string
	size int
}

var fontCache = struct {
	sync.Mutex
	entries map[fontKey]*ttfFont
}{
	entries: map[fontKey]*ttfFont{},
}

func fontForStyle(style Style) *ttfFont {
	path, ok := resolveFontPath(style.FontPath)
	if !ok {
		return nil
	}
	size := defaultFontHeight
	if value, ok := resolveFontSize(style.FontSize); ok {
		size = value
	}
	return getTTFFont(path, size)
}

func getTTFFont(path string, size int) *ttfFont {
	if path == "" {
		return nil
	}
	if size <= 0 {
		size = defaultFontHeight
	}
	key := fontKey{path: path, size: size}
	fontCache.Lock()
	if entry, ok := fontCache.entries[key]; ok {
		fontCache.Unlock()
		return entry
	}
	fontCache.Unlock()

	entry := loadTTFFont(path, size)
	fontCache.Lock()
	fontCache.entries[key] = entry
	fontCache.Unlock()
	return entry
}

func loadTTFFont(path string, size int) *ttfFont {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return nil
	}
	metrics := computeFontMetrics(face)
	return &ttfFont{
		path:    path,
		size:    size,
		face:    face,
		metrics: metrics,
	}
}

func computeFontMetrics(face xfont.Face) fontMetrics {
	if face == nil {
		return defaultFontMetrics()
	}
	m := face.Metrics()
	height := m.Height.Ceil()
	if height <= 0 {
		height = defaultFontHeight
	}
	ascent := m.Ascent.Ceil()
	if ascent <= 0 {
		ascent = height * 3 / 4
	}
	if ascent > height {
		ascent = height
	}
	width := glyphAdvancePixels(face, 'M')
	if width <= 0 {
		width = glyphAdvancePixels(face, '0')
	}
	if width <= 0 {
		width = glyphAdvancePixels(face, ' ')
	}
	if width <= 0 {
		width = defaultCharWidth
	}
	return fontMetrics{
		width:  width,
		height: height,
		ascent: ascent,
	}
}

func glyphAdvancePixels(face xfont.Face, r rune) int {
	if face == nil {
		return 0
	}
	advance, ok := face.GlyphAdvance(r)
	if !ok {
		return 0
	}
	width := advance.Ceil()
	if width < 0 {
		width = -width
	}
	return width
}
