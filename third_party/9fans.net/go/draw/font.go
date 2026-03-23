package draw

import (
	"image"
	"os"
	"strconv"
	"strings"

	"surface"
)

type Font struct {
	Display *Display
	Name    string
	Height  int
	Ascent  int
	Scale   int

	surface *surface.Font
}

func newFallbackFont(d *Display, name string) *Font {
	metrics := surface.DefaultFontMetrics()
	return &Font{
		Display: d,
		Name:    name,
		Height:  metrics.Height,
		Ascent:  metrics.Ascent,
		Scale:   1,
	}
}

func loadFontFile(d *Display, path string) *Font {
	path, size := parseFontRequest(path)
	sf := surface.GetFont(path, size)
	if sf == nil {
		return nil
	}
	metrics := sf.Metrics()
	return &Font{
		Display: d,
		Name:    path,
		Height:  metrics.Height,
		Ascent:  metrics.Ascent,
		Scale:   1,
		surface: sf,
	}
}

func parseFontRequest(name string) (path string, size int) {
	path = name
	size = surface.DefaultFontHeight
	if at := strings.LastIndex(name, "@"); at >= 0 && at+1 < len(name) {
		if parsed, err := strconv.Atoi(name[at+1:]); err == nil && parsed > 0 {
			path = name[:at]
			size = parsed
		}
	}
	return path, size
}

func defaultFontPath() string {
	for _, path := range []string{
		os.Getenv("DRAW_DEFAULT_TTF"),
		"assets/OpenSans-Regular.ttf",
		"assets/RobotoMono-Regular.ttf",
	} {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (f *Font) StringWidth(s string) int {
	if f == nil {
		return 0
	}
	if f.surface != nil {
		return f.surface.MeasureString(s)
	}
	return len(s) * surface.DefaultCharWidth
}

func (f *Font) StringSize(s string) image.Point {
	if f == nil {
		return image.ZP
	}
	return image.Pt(f.StringWidth(s), f.Height)
}
