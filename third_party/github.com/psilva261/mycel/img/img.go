package img

import (
	"9fans.net/go/draw"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel"
	log "github.com/psilva261/mycel/logger"
	"image"
	"image/color"
	"net/url"
	"strings"

	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const SrcZero = "//:0"

func parseDataUri(addr string) (data []byte, ct mycel.ContentType, err error) {
	addr = strings.TrimPrefix(addr, "data:")
	if strings.Contains(addr, "charset=UTF-8") {
		return nil, ct, fmt.Errorf("cannot handle charset")
	}
	parts := strings.Split(addr, ",")
	if len(parts) == 0 {
		return nil, ct, fmt.Errorf("empty: %v", addr)
	}

	var ctStr string
	if strings.Contains(parts[0], ";") {
		header := strings.Split(parts[0], ";")
		ctStr = header[0]
	} else {
		ctStr = parts[0]
	}
	if ct, err = mycel.NewContentType(ctStr, nil); err != nil {
		return nil, ct, fmt.Errorf("content type: %v: %w", ctStr, err)
	}

	if len(parts) == 1 {
		return nil, ct, fmt.Errorf("empty: %v", addr)
	}
	if strings.Contains(addr, "base64") {
		e := base64.RawStdEncoding
		if strings.HasSuffix(addr, "=") {
			e = base64.StdEncoding
		}
		if data, err = e.DecodeString(parts[1]); err != nil {
			return nil, ct, fmt.Errorf("base64 decode %v src: %w", addr, err)
		}
	} else {
		out, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, ct, fmt.Errorf("url decode: %w", err)
		}
		data = []byte(out)
	}
	return
}

func quoteAttrsInTag(s string) string {
	eqs := make([]int, 0, 5)
	offset := 0

	for {
		i := strings.Index(s[offset:], "=")
		if i >= 0 {
			eqs = append(eqs, i+offset)
			offset += i + 1
		} else {
			break
		}
	}

	keyStarts := make([]int, len(eqs))
	for i, eq := range eqs {
		j := strings.LastIndex(s[:eq], " ")
		keyStarts[i] = j
	}

	valueEnds := make([]int, len(keyStarts))
	for i, _ := range keyStarts {
		if i+1 < len(keyStarts) {
			valueEnds[i] = keyStarts[i+1]
		} else {
			off := eqs[i]
			jj := strings.Index(s[off:], ">")
			valueEnds[i] = jj + off
			if s[valueEnds[i]-1:valueEnds[i]] == "/" {
				valueEnds[i]--
			}
		}
	}

	for i := len(eqs) - 1; i >= 0; i-- {
		s = s[:valueEnds[i]] + `"` + s[valueEnds[i]:]
		s = s[:eqs[i]+1] + `"` + s[eqs[i]+1:]
	}

	return s
}

func quoteAttrs(s string) string {
	s = strings.ReplaceAll(s, `'`, `"`)
	if strings.Contains(s, `"`) {
		return s
	}

	tagStarts := make([]int, 0, 5)
	tagEnds := make([]int, 0, 5)

	offset := 0
	for {
		i := strings.Index(s[offset:], "<")
		if i >= 0 {
			tagStarts = append(tagStarts, i+offset)
			offset += i + 1
		} else {
			break
		}
	}

	offset = 0
	for {
		i := strings.Index(s[offset:], ">")
		if i >= 0 {
			tagEnds = append(tagEnds, i+offset)
			offset += i + 1
		} else {
			break
		}
	}

	if len(tagStarts) != len(tagEnds) {
		log.Errorf("quoteAttrs: len(tagStarts) != len(tagEnds)")
		return s
	}

	for i := len(tagStarts) - 1; i >= 0; i-- {
		from := tagStarts[i]
		to := tagEnds[i] + 1
		q := quoteAttrsInTag(s[from:to])
		s = s[:tagStarts[i]] + q + s[tagEnds[i]+1:]
	}

	return s
}

// Svg returns the svg+xml with the sizing defined in
// viewbox unless w and h != 0
func Svg(dui *duit.DUI, data string, w, h int) (ni *draw.Image, err error) {
	rgba, err := svg(data, w, h)
	if err != nil {
		return nil, err
	}
	ni, err = dui.Display.AllocImage(rgba.Bounds(), draw.ABGR32, false, draw.White)
	if err != nil {
		return nil, fmt.Errorf("allocimage: %s", err)
	}
	_, err = ni.Load(rgba.Bounds(), rgba.Pix)
	if err != nil {
		return nil, fmt.Errorf("load image: %s", err)
	}
	return
}

func svg(data string, w, h int) (img *image.RGBA, err error) {
	if w == 0 || h == 0 {
		w = 32
		h = 32
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRGBA(rgba, color.RGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff})
	return rgba, nil
}

// Load and resize to w and h if != 0
func Load(dui *duit.DUI, f mycel.Fetcher, src string, maxW, w, h int, forceSync bool) (ni *draw.Image, err error) {
	log.Printf("Load(..., %v, maxW=%v, w=%v, h=%v, ...)", src, maxW, w, h)
	ch := make(chan image.Image, 1)
	var bounds draw.Rectangle
	if w != 0 && h != 0 && !forceSync {
		bounds = draw.Rect(0, 0, w, h)
		go func() {
			log.Printf("load async %v...", src)
			drawImg, err := load(f, src, maxW, w, h)
			if err != nil {
				log.Errorf("load %v: %v", src, err)
				close(ch)
				return
			}
			ch <- drawImg
			log.Printf("loaded async %v", src)
		}()
	} else {
		drawImg, err := load(f, src, maxW, w, h)
		if err != nil {
			return nil, err
		}
		bounds = drawImg.Bounds()
		ch <- drawImg
	}

	ni, err = dui.Display.AllocImage(bounds, draw.ABGR32, false, draw.White)
	if err != nil {
		return nil, fmt.Errorf("allocimage: %s", err)
	}

	go func() {
		drawImg, ok := <-ch
		if !ok {
			log.Errorf("could not load image %v", src)
			return
		}
		dui.Call <- func() {
			// Stolen from duit.ReadImage
			var rgba *image.RGBA
			switch i := drawImg.(type) {
			case *image.RGBA:
				rgba = i
			default:
				b := drawImg.Bounds()
				rgba = image.NewRGBA(image.Rectangle{image.ZP, b.Size()})
				copyToRGBA(rgba, drawImg)
			}
			_, err = ni.Load(rgba.Bounds(), rgba.Pix)
			if err != nil {
				log.Errorf("load image: %s", err)
			}
			if dui != nil {
				dui.MarkDraw(nil)
				dui.Render()
			}
			log.Printf("copied image %v", src)
		}
	}()
	return
}

func load(f mycel.Fetcher, src string, maxW, w, h int) (img image.Image, err error) {
	var imgUrl *url.URL
	var data []byte
	var contentType mycel.ContentType

	if strings.HasPrefix(src, "data:") {
		if data, contentType, err = parseDataUri(src); err != nil {
			return nil, fmt.Errorf("parse data uri %v: %w", src, err)
		}
	} else {
		if imgUrl, err = f.LinkedUrl(src); err != nil {
			return nil, err
		}
		if data, contentType, err = f.Get(imgUrl); err != nil {
			return nil, fmt.Errorf("get %v: %w", imgUrl, err)
		}
	}

	if contentType.IsSvg() {
		img, err := svg(string(data), w, h)
		if err != nil {
			return nil, fmt.Errorf("svg: %v", err)
		}
		return img, nil
	} else {
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("decode %v: %w", imgUrl, err)
		}
		if maxW != 0 || w != 0 || h != 0 {
			dx := img.Bounds().Max.X
			dy := img.Bounds().Max.Y
			log.Printf("dx,dy=%v,%v", dx, dy)
			if w == 0 && h == 0 && 0 < maxW && maxW < dx {
				w = maxW
			}

			newX, newY, skip := newSizes(dx, dy, w, h)

			if !skip {
				log.Printf("resize image to %v x %v", newX, newY)
				dst := image.NewRGBA(image.Rect(0, 0, newX, newY))
				scaleNearest(img, dst)
				img = dst
			} else {
				log.Printf("skip resizing")
			}
		}
	}

	return img, nil
}

func newSizes(oldX, oldY, wantedX, wantedY int) (newX, newY int, skip bool) {
	if oldX == 0 || oldY == 0 || (wantedX == 0 && wantedY == 0) {
		return oldX, oldY, true
	}
	if wantedX == 0 {
		newX = int(float64(oldX) * float64(wantedY) / float64(oldY))
		newY = wantedY
	} else {
		newX = wantedX
		newY = int(float64(oldY) * float64(wantedX) / float64(oldX))
	}

	if newX > 2000 || newY > 2000 {
		return oldX, oldY, true
	}

	r := float64(newX) / float64(oldX)
	if 0.8 <= r && r <= 1.2 {
		return oldX, oldY, true
	}

	return
}

func scaleNearest(src image.Image, dst *image.RGBA) {
	if src == nil || dst == nil {
		return
	}
	sb := src.Bounds()
	db := dst.Bounds()
	if sb.Dx() <= 0 || sb.Dy() <= 0 || db.Dx() <= 0 || db.Dy() <= 0 {
		return
	}
	for y := 0; y < db.Dy(); y++ {
		sy := sb.Min.Y + (y*sb.Dy())/db.Dy()
		for x := 0; x < db.Dx(); x++ {
			sx := sb.Min.X + (x*sb.Dx())/db.Dx()
			c := color.RGBAModel.Convert(src.At(sx, sy))
			dst.Set(db.Min.X+x, db.Min.Y+y, c)
		}
	}
}

func fillRGBA(dst *image.RGBA, c color.RGBA) {
	if dst == nil {
		return
	}
	b := dst.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
}

func copyToRGBA(dst *image.RGBA, src image.Image) {
	if dst == nil || src == nil {
		return
	}
	db := dst.Bounds()
	sb := src.Bounds()
	for y := 0; y < db.Dy(); y++ {
		sy := sb.Min.Y + y
		if sy >= sb.Max.Y {
			break
		}
		for x := 0; x < db.Dx(); x++ {
			sx := sb.Min.X + x
			if sx >= sb.Max.X {
				break
			}
			c := color.RGBAModel.Convert(src.At(sx, sy)).(color.RGBA)
			dst.SetRGBA(db.Min.X+x, db.Min.Y+y, c)
		}
	}
}
