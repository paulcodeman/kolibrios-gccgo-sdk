package style

import (
	"fmt"
	"os"
	"strings"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

var (
	dui       *duit.DUI
	fontCache = make(map[string]*draw.Font)
)

func Init(d *duit.DUI) {
	dui = d
}

func (cs Map) FontFilename() (string, bool) {
	paths := []string{
		os.Getenv("DRAW_DEFAULT_TTF"),
		"assets/OpenSans-Regular.ttf",
		"assets/RobotoMono-Regular.ttf",
	}
	if cs.IsFontBold() {
		paths = append([]string{
			os.Getenv("DRAW_BOLD_TTF"),
			"assets/Go-Bold.ttf",
		}, paths...)
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func (cs Map) Font() *draw.Font {
	if dui == nil || dui.Display == nil {
		return nil
	}
	fn, ok := cs.FontFilename()
	if !ok || strings.TrimSpace(fn) == "" {
		return dui.Font(nil)
	}
	size := int(cs.FontHeight())
	if fs := int(cs.FontSize()); fs > 0 {
		size = fs
	}
	if size <= 0 {
		size = 16
	}
	key := fmt.Sprintf("%s@%d", fn, size)
	if font, ok := fontCache[key]; ok && font != nil {
		return font
	}
	font, err := dui.Display.OpenFont(key)
	if err != nil || font == nil {
		return dui.Font(nil)
	}
	fontCache[key] = font
	return font
}
