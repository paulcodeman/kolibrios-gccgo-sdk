package style

import (
	"9fans.net/go/draw"
	"fmt"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/img"
	log "github.com/psilva261/mycel/logger"
	"image"
	"strings"
)

var colorCache = make(map[draw.Color]*draw.Image)
var fetcher mycel.Fetcher

func SetFetcher(f mycel.Fetcher) {
	fetcher = f
}

var TextNode = Map{
	Declarations: map[string]Declaration{
		"display": {
			Prop: "display",
			Val:  "inline",
		},
	},
}

func (cs Map) BoxBackground() (i *draw.Image, err error) {
	if bgImg := cs.backgroundImage(); bgImg != nil {
		return bgImg, nil
	}
	if bgImg := cs.BackgroundGradient(); bgImg != nil {
		return bgImg, nil
	}
	bgColor, ok := cs.backgroundColor()
	if !ok || dui == nil || dui.Display == nil {
		return nil, nil
	}
	if i, ok = colorCache[bgColor]; ok {
		return i, nil
	}
	i, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, bgColor)
	if err != nil {
		return nil, fmt.Errorf("alloc img: %w", err)
	}
	colorCache[bgColor] = i
	return i, nil
}

func (cs Map) backgroundColor() (c draw.Color, ok bool) {
	if d, ok := cs.Declarations["background-color"]; ok {
		if h, ok := ParseColor(d.Val); ok {
			return draw.Color(h), true
		}
	}
	if d, ok := cs.Declarations["background"]; ok {
		if h, ok := ParseColor(d.Val); ok {
			return draw.Color(h), true
		}
	}
	return 0, false
}

func (cs Map) BackgroundGradient() (img *draw.Image) {
	if dui == nil || dui.Display == nil {
		return nil
	}
	c, ok := cs.backgroundGradient()
	if !ok {
		return nil
	}
	var err error
	img, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, c)
	if err != nil {
		log.Errorf("alloc img: %v", err)
		return nil
	}
	return img
}

func (cs Map) backgroundGradient() (c draw.Color, ok bool) {
	d, ok := cs.Declarations["background"]
	if !ok {
		return 0, false
	}
	v := strings.TrimSpace(d.Val)
	if strings.HasPrefix(v, "linear-gradient(") {
		v = strings.TrimPrefix(v, "linear-gradient(")
	} else {
		return 0, false
	}
	v = strings.TrimSuffix(v, ")")

	colors := make([]draw.Color, 0, 2)
	for i := 0; i < len(v); {
		m := strings.Index(v[i:], ",")
		op := strings.Index(v[i:], "(")
		cl := strings.Index(v[i:], ")")
		if m < 0 {
			break
		}
		var arg string
		if cl > 0 && op >= 0 && op < m && m < cl {
			arg = v[i : i+cl+1]
			i += cl + 1
		} else {
			arg = v[i : i+m]
			i += m + 1
		}
		arg = strings.ReplaceAll(arg, " ", "")
		if h, ok := ParseColor(arg); ok {
			colors = append(colors, draw.Color(h))
		}
	}
	if len(colors) >= 2 {
		return linearGradient(colors[0], colors[1], 0.5, 0, 1), true
	}
	return 0, false
}

func linearGradient(from, to draw.Color, x, y, xmax float64) (c draw.Color) {
	fr, fg, fb, fa := from.RGBA()
	tr, tg, tb, ta := to.RGBA()
	d := x / xmax
	r := uint32(float64(fr) + d*float64(tr-fr))
	g := uint32(float64(fg) + d*float64(tg-fg))
	b := uint32(float64(fb) + d*float64(tb-fb))
	a := uint32(float64(fa) + d*float64(ta-fa))
	cc := (r / 256) << 24
	cc |= (g / 256) << 16
	cc |= (b / 256) << 8
	cc |= a / 256
	return draw.Color(cc)
}

func backgroundImageUrl(decl Declaration) (url string, ok bool) {
	if v := decl.Val; strings.Contains(v, "url(") && strings.Contains(v, ")") {
		v = strings.ReplaceAll(v, `"`, "")
		v = strings.ReplaceAll(v, `'`, "")
		from := strings.Index(v, "url(")
		if from < 0 {
			return "", false
		}
		from += len("url(")
		imgURL := v[from:]
		to := strings.Index(imgURL, ")")
		if to < 0 {
			return "", false
		}
		return imgURL[:to], true
	}
	return "", false
}

func (cs Map) backgroundImage() (i *draw.Image) {
	if fetcher == nil || dui == nil {
		return nil
	}
	decl, ok := cs.Declarations["background-image"]
	if !ok {
		decl, ok = cs.Declarations["background"]
	}
	if !ok {
		return nil
	}
	imgURL, ok := backgroundImageUrl(decl)
	if !ok {
		return nil
	}
	w := cs.Width()
	h := cs.Height()
	i, err := img.Load(dui, fetcher, imgURL, 0, w, h, true)
	if err != nil {
		log.Errorf("bg img load %v: %v", imgURL, err)
		return nil
	}
	return i
}
