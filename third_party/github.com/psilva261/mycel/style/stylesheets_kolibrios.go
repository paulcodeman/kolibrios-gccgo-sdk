package style

import (
	"fmt"
	"github.com/andybalholm/cascadia"
	"github.com/mjl-/duit"
	log "github.com/psilva261/mycel/logger"
	"golang.org/x/image/colornames"
	"image"
	"kos"
	"math"
	"net/html"
	"runtime"
	"strconv"
	"strings"
)

const FontBaseSize = 11.0

var WindowWidth = 1280
var WindowHeight = 1080

var MediaValues = map[string]string{
	"type":                 "screen",
	"width":                fmt.Sprintf("%vpx", WindowWidth),
	"orientation":          "landscape",
	"prefers-color-scheme": "light",
}

const AddOnCSS = `
/* https://developer.mozilla.org/en-US/docs/Web/HTML/Inline_elements */
a, abbr, acronym, audio, b, bdi, bdo, big, br, button, canvas, cite, code, data, datalist, del, dfn, em, embed, i, iframe, img, input, ins, kbd, label, map, mark, meter, noscript, object, output, picture, progress, q, ruby, s, samp, script, select, slot, small, span, strong, sub, sup, svg, template, textarea, time, u, tt, var, video, wbr {
  display: inline;
}

/* non-HTML5 elements */
font, strike, tt {
  display: inline;
}

button, textarea, input, select {
  display: inline-block;
}

/* https://developer.mozilla.org/en-US/docs/Web/HTML/Block-level_elements */
address, article, aside, blockquote, details, dialog, dd, div, dl, dt, fieldset, figcaption, figure, footer, form, h1, h2, h3, h4, h5, h6, header, hgroup, hr, li, main, nav, ol, p, pre, section, table, ul {
  display: block;
}

html, body {
  display: block;
}

html {
  font-size: 16px;
}

body {
  margin: 8px;
}

h1 {
  font-size: 2em;
  margin: 0.67em 0;
}

h2 {
  font-size: 1.5em;
  margin: 0.75em 0;
}

h3 {
  font-size: 1.17em;
  margin: 0.83em 0;
}

h4, p, ul, ol, pre, blockquote {
  margin: 1em 0;
}

h1, h2, h3, h4, h5, h6, b, strong {
  font-weight: bold;
}

*[href] {
  color: blue;
  margin-right: 2px;
}
`

type Spacing = duit.Space

func SetViewport(width int, height int) {
	if width > 0 {
		WindowWidth = width
	}
	if height > 0 {
		WindowHeight = height
	}
	MediaValues["width"] = fmt.Sprintf("%vpx", WindowWidth)
	if WindowWidth >= WindowHeight {
		MediaValues["orientation"] = "landscape"
	} else {
		MediaValues["orientation"] = "portrait"
	}
}

func MergeNodeMaps(m, addOn map[*html.Node]Map) {
	for n, mp := range addOn {
		initial := m[n]
		m[n] = initial.ApplyChildStyle(mp, true)
	}
}

func FetchNodeMap(doc *html.Node, cssText string) (m map[*html.Node]Map, err error) {
	mr, rv, err := FetchNodeRules(doc, cssText)
	if err != nil {
		return nil, fmt.Errorf("fetch rules: %w", err)
	}
	m = make(map[*html.Node]Map)
	for n, rs := range mr {
		ds := make(map[string]Declaration)
		for _, r := range rs {
			for _, d := range r.Declarations {
				if exist, ok := ds[d.Prop]; ok && smaller(d, exist) {
					continue
				}
				if strings.HasPrefix(d.Val, "var(") {
					v := strings.TrimPrefix(d.Val, "var(")
					v = strings.TrimSuffix(v, ")")
					if vv, ok := rv[v]; ok {
						d.Val = vv
					}
				}
				ds[d.Prop] = d
			}
		}
		m[n] = Map{Declarations: ds}
	}
	return
}

func smaller(d, dd Declaration) bool {
	if dd.Important {
		return true
	} else if d.Important {
		return false
	}
	return d.Specificity.Less(dd.Specificity)
}

func compile(v string) (cs cascadia.SelectorGroup, err error) {
	return cascadia.ParseGroup(v)
}

func FetchNodeRules(doc *html.Node, cssText string) (m map[*html.Node][]Rule, rVars map[string]string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("fetch node rules panic: %v", rec)
		}
	}()

	m = make(map[*html.Node][]Rule)
	rVars = make(map[string]string)
	s, err := Parse(cssText, false)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	yieldCounter := 0
	maybeYield := func() {
		yieldCounter++
		if (yieldCounter & 63) == 0 {
			runtime.Gosched()
		}
	}
	processRule := func(m map[*html.Node][]Rule, r Rule) error {
		for i, sel := range r.Selectors {
			func(sel Selector, i int) {
				defer func() {
					if rec := recover(); rec != nil {
						log.Errorf("css selector panic %q: %v", sel.Val, rec)
					}
				}()

				maybeYield()
				if sel.Val == ":root" {
					for _, d := range r.Declarations {
						rVars[d.Prop] = d.Val
					}
				}
				csg, compileErr := compile(sel.Val)
				if compileErr != nil {
					log.Printf("cssSel compile %v: %v", sel.Val, compileErr)
					return
				}
				for _, cs := range csg {
					for _, el := range cascadia.QueryAll(doc, cs) {
						maybeYield()
						existing := m[el]
						var sr Rule
						sr = r
						sr.Selectors = []Selector{r.Selectors[i]}
						for j := range sr.Declarations {
							sr.Declarations[j].Specificity[0] = cs.Specificity()[0]
							sr.Declarations[j].Specificity[1] = cs.Specificity()[1]
							sr.Declarations[j].Specificity[2] = cs.Specificity()[2]
						}
						existing = append(existing, sr)
						m[el] = existing
					}
				}
			}(sel, i)
		}
		return nil
	}
	for _, r := range s.Rules {
		maybeYield()
		if err := processRule(m, r); err != nil {
			return nil, nil, fmt.Errorf("process rule: %w", err)
		}
		if strings.HasPrefix(r.Prelude, "@media") {
			p := strings.TrimSpace(strings.TrimPrefix(r.Prelude, "@media"))
			yes, err := func() (yes bool, err error) {
				defer func() {
					if rec := recover(); rec != nil {
						err = fmt.Errorf("media query panic: %v", rec)
					}
				}()
				return MatchQuery(p, MediaValues)
			}()
			if err != nil {
				log.Errorf("match query %v: %v", r.Prelude, err)
			} else if !yes {
				continue
			}
		}
		for _, rr := range r.Rules {
			maybeYield()
			if err := processRule(m, rr); err != nil {
				return nil, nil, fmt.Errorf("process embedded rule: %w", err)
			}
		}
	}
	return
}

type DomTree interface {
	Rect() image.Rectangle
	Parent() (p DomTree, ok bool)
	Style() Map
}

type Map struct {
	Declarations map[string]Declaration
	DomTree      `json:"-"`
}

func NewMap(n *html.Node) Map {
	s := Map{
		Declarations: make(map[string]Declaration),
	}
	for _, a := range n.Attr {
		switch a.Key {
		case "style":
			v := strings.TrimSpace(a.Val)
			if !strings.HasSuffix(v, ";") {
				v += ";"
			}
			st, err := Parse(v, true)
			if err != nil {
				log.Printf("could not parse '%v'", a.Val)
				continue
			}
			if len(st.Rules) == 0 {
				continue
			}
			for _, d := range st.Rules[0].Declarations {
				s.Declarations[d.Prop] = d
			}
		case "height", "width":
			v := a.Val
			if !strings.HasSuffix(v, "%") && !strings.HasSuffix(v, "px") {
				v += "px"
			}
			s.Declarations[a.Key] = Declaration{Prop: a.Key, Val: v}
		case "bgcolor":
			s.Declarations["background-color"] = Declaration{Prop: "background-color", Val: a.Val}
		}
	}
	return s
}

func (cs Map) ApplyChildStyle(ccs Map, copyAll bool) (res Map) {
	res.Declarations = make(map[string]Declaration)
	for k, v := range cs.Declarations {
		switch k {
		case "azimuth", "border-collapse", "border-spacing", "caption-side", "color", "cursor", "direction", "elevation", "empty-cells", "font-family", "font-size", "font-style", "font-variant", "font-weight", "font", "letter-spacing", "line-height", "list-style-image", "list-style-position", "list-style-type", "list-style", "orphans", "pitch-range", "pitch", "quotes", "richness", "speak-header", "speak-numeral", "speak-punctuation", "speak", "speech-rate", "stress", "text-align", "text-indent", "text-transform", "visibility", "voice-family", "volume", "white-space", "widows", "word-spacing":
		default:
			if !copyAll {
				continue
			}
		}
		res.Declarations[k] = v
	}
	for k, d := range ccs.Declarations {
		if d.Val == "inherit" {
			continue
		}
		if exist, ok := res.Declarations[k]; ok && smaller(d, exist) {
			continue
		}
		res.Declarations[k] = d
	}
	return
}

func (cs Map) FontSize() float64 {
	fs, ok := cs.Declarations["font-size"]
	if !ok || fs.Val == "" {
		if cs.DomTree != nil {
			if p, ok := cs.DomTree.Parent(); ok {
				if inherited := p.Style().FontSize(); inherited > 0 {
					return inherited
				}
			}
		}
		return FontBaseSize
	}
	value := strings.TrimSpace(fs.Val)
	parentSize := FontBaseSize
	if cs.DomTree != nil {
		if p, ok := cs.DomTree.Parent(); ok {
			if inherited := p.Style().FontSize(); inherited > 0 {
				parentSize = inherited
			}
		}
	}
	switch {
	case strings.HasSuffix(value, "px"):
		f, err := strconv.ParseFloat(strings.TrimSuffix(value, "px"), 64)
		if err == nil && f > 0 {
			return f
		}
	case strings.HasSuffix(value, "rem"):
		f, err := strconv.ParseFloat(strings.TrimSuffix(value, "rem"), 64)
		if err == nil && f > 0 {
			return f * FontBaseSize
		}
	case strings.HasSuffix(value, "em"):
		f, err := strconv.ParseFloat(strings.TrimSuffix(value, "em"), 64)
		if err == nil && f > 0 {
			return f * parentSize
		}
	case strings.HasSuffix(value, "%"):
		f, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
		if err == nil && f > 0 {
			return parentSize * f / 100.0
		}
	}
	return parentSize
}

func (cs Map) FontHeight() float64 {
	fontSize := cs.FontSize()
	if fontSize <= 0 {
		fontSize = FontBaseSize
	}
	if lh, ok := cs.Declarations["line-height"]; ok {
		value := strings.TrimSpace(strings.ToLower(lh.Val))
		switch {
		case value == "", value == "normal":
			return math.Max(1, math.Round(fontSize*1.2))
		case value == "inherit":
			if cs.DomTree != nil {
				if p, ok := cs.DomTree.Parent(); ok {
					if inherited := p.Style().FontHeight(); inherited > 0 {
						return inherited
					}
				}
			}
		case strings.HasSuffix(value, "%"):
			if f, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64); err == nil && f > 0 {
				return math.Max(1, math.Round(fontSize*f/100.0))
			}
		default:
			if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 {
				return math.Max(1, math.Round(fontSize*f))
			}
			if f, unit, err := length(&cs, value); err == nil && f > 0 && unit != "%" {
				return math.Max(1, math.Round(f))
			}
		}
	}
	return math.Max(1, math.Round(fontSize*1.2))
}

func (cs Map) FontWeight() string {
	if d, ok := cs.Declarations["font-weight"]; ok {
		return strings.ToLower(strings.TrimSpace(d.Val))
	}
	return ""
}

func (cs Map) IsFontBold() bool {
	switch cs.FontWeight() {
	case "bold", "bolder":
		return true
	}
	if v := cs.FontWeight(); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n >= 600
		}
	}
	return false
}

func (cs Map) Color() kos.Color {
	if d, ok := cs.Declarations["color"]; ok {
		if h, ok := ParseColor(d.Val); ok {
			return h
		}
	}
	return kos.Color(0x000000FF)
}

func ParseColor(propVal string) (kos.Color, bool) {
	var r, g, b uint32
	propVal = strings.TrimSpace(strings.ToLower(propVal))
	if strings.HasPrefix(propVal, "rgb") {
		parseComponent := func(v string) (uint32, bool) {
			v = strings.TrimSpace(v)
			if v == "" {
				return 0, false
			}
			if strings.HasSuffix(v, "%") {
				f, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
				if err != nil {
					return 0, false
				}
				if f < 0 {
					f = 0
				} else if f > 100 {
					f = 100
				}
				return uint32(math.Round(f * 255.0 / 100.0)), true
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0, false
			}
			if f < 0 {
				f = 0
			} else if f > 255 {
				f = 255
			}
			return uint32(math.Round(f)), true
		}

		open := strings.Index(propVal, "(")
		close := strings.LastIndex(propVal, ")")
		if open < 0 || close <= open {
			return 0, false
		}
		val := strings.TrimSpace(propVal[open+1 : close])
		if alphaSep := strings.Index(val, "/"); alphaSep >= 0 {
			val = strings.TrimSpace(val[:alphaSep])
		}
		vals := strings.FieldsFunc(val, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		if len(vals) < 3 {
			return 0, false
		}
		rr, ok := parseComponent(vals[0])
		if !ok {
			return 0, false
		}
		gg, ok := parseComponent(vals[1])
		if !ok {
			return 0, false
		}
		bb, ok := parseComponent(vals[2])
		if !ok {
			return 0, false
		}
		r = rr
		g = gg
		b = bb
	} else if strings.HasPrefix(propVal, "#") {
		hexColor := propVal[1:]
		switch len(hexColor) {
		case 3:
			rr, err := strconv.ParseInt(hexColor[0:1], 16, 32)
			if err != nil {
				return 0, false
			}
			gg, err := strconv.ParseInt(hexColor[1:2], 16, 32)
			if err != nil {
				return 0, false
			}
			bb, err := strconv.ParseInt(hexColor[2:3], 16, 32)
			if err != nil {
				return 0, false
			}
			r = uint32(rr) * 0x11
			g = uint32(gg) * 0x11
			b = uint32(bb) * 0x11
		case 6:
			rr, err := strconv.ParseInt(hexColor[0:2], 16, 32)
			if err != nil {
				return 0, false
			}
			gg, err := strconv.ParseInt(hexColor[2:4], 16, 32)
			if err != nil {
				return 0, false
			}
			bb, err := strconv.ParseInt(hexColor[4:6], 16, 32)
			if err != nil {
				return 0, false
			}
			r = uint32(rr)
			g = uint32(gg)
			b = uint32(bb)
		default:
			return 0, false
		}
	} else {
		colorRGBA, ok := colornames.Map[propVal]
		if !ok {
			return 0, false
		}
		r, g, b, _ = colorRGBA.RGBA()
		r /= 257
		g /= 257
		b /= 257
	}
	return kos.Color((r << 24) | (g << 16) | (b << 8) | 0xFF), true
}

func (cs Map) IsInline() bool {
	if propVal, ok := cs.Declarations["float"]; ok {
		switch strings.TrimSpace(strings.ToLower(propVal.Val)) {
		case "left", "right":
			return true
		}
	}
	if propVal, ok := cs.Declarations["display"]; ok {
		return propVal.Val == "inline" || propVal.Val == "inline-block"
	}
	return false
}

func (cs Map) IsDisplayNone() bool {
	if propVal, ok := cs.Declarations["display"]; ok && propVal.Val == "none" {
		return true
	}
	if propVal, ok := cs.Declarations["clip"]; ok && strings.ReplaceAll(propVal.Val, " ", "") == "rect(1px,1px,1px,1px)" {
		return true
	}
	if propVal, ok := cs.Declarations["width"]; ok && propVal.Val == "1px" {
		if propVal, ok := cs.Declarations["height"]; ok && propVal.Val == "1px" {
			return true
		}
	}
	return false
}

func (cs Map) IsFlex() bool {
	propVal, ok := cs.Declarations["display"]
	return ok && propVal.Val == "flex"
}

func (cs Map) IsFlexDirectionRow() bool {
	propVal, ok := cs.Declarations["flex-direction"]
	if ok {
		switch propVal.Val {
		case "row":
			return true
		case "column":
			return false
		}
	}
	return true
}

func (cs *Map) Tlbr(key string) (s Spacing, err error) {
	if all, ok := cs.Declarations[key]; ok {
		parts := strings.Fields(all.Val)
		nums := make([]int, len(parts))
		for i, p := range parts {
			if f, _, err := length(cs, p); err == nil {
				nums[i] = int(f)
			} else {
				return s, fmt.Errorf("length: %w", err)
			}
		}
		if len(nums) > 0 {
			s.Top = nums[0]
			s.Right = s.Top
			s.Bottom = s.Top
			s.Left = s.Top
		}
		if len(nums) >= 2 {
			s.Right = nums[1]
			s.Left = s.Right
		}
		if len(nums) >= 3 {
			s.Bottom = nums[2]
		}
		if len(nums) >= 4 {
			s.Left = nums[3]
		}
	}
	if t, err := cs.CssPx(key + "-top"); err == nil {
		s.Top = t
	}
	if r, err := cs.CssPx(key + "-right"); err == nil {
		s.Right = r
	}
	if b, err := cs.CssPx(key + "-bottom"); err == nil {
		s.Bottom = b
	}
	if l, err := cs.CssPx(key + "-left"); err == nil {
		s.Left = l
	}
	if s.Top > 100 {
		s.Top = 0
	}
	if s.Bottom > 100 {
		s.Bottom = 0
	}
	return
}

func length(cs *Map, l string) (f float64, unit string, err error) {
	value := strings.TrimSpace(strings.ToLower(l))
	if value == "auto" || value == "inherit" || value == "initial" || value == "0" || value == "" {
		return 0, "px", nil
	}
	if strings.Contains(value, "calc(") {
		return 0, "", fmt.Errorf("calc not supported")
	}
	for _, suffix := range []string{"px", "%", "rem", "em", "ex", "vw", "vh", "mm"} {
		if strings.HasSuffix(value, suffix) {
			if raw := strings.TrimSuffix(value, suffix); raw != "" {
				f, err = strconv.ParseFloat(raw, 64)
				if err != nil {
					return 0, "", fmt.Errorf("error parsing '%v': %w", value, err)
				}
			}
			unit = suffix
			break
		}
	}
	if unit == "" {
		f, err = strconv.ParseFloat(value, 64)
		if err == nil {
			return f, "px", nil
		}
		return 0, "", fmt.Errorf("unknown suffix: %v", value)
	}

	switch unit {
	case "px":
	case "rem":
		f *= FontBaseSize
	case "em", "ex":
		if cs == nil {
			f *= FontBaseSize
		} else {
			f *= cs.FontSize()
			if unit == "ex" {
				f *= 0.5
			}
		}
	case "vw":
		f *= float64(WindowWidth) / 100.0
	case "vh":
		f *= float64(WindowHeight) / 100.0
	case "%":
		if cs == nil {
			return 0, "%", nil
		}
		var wp int
		if p, ok := cs.DomTree.Parent(); ok {
			wp = p.Style().baseWidth()
		}
		f *= 0.01 * float64(wp)
	case "mm":
		f *= 96.0 / 25.4
	default:
		return f, unit, fmt.Errorf("unknown suffix: %v", value)
	}
	return
}

func (cs *Map) Height() int {
	if d, ok := cs.Declarations["height"]; ok {
		f, _, err := length(cs, d.Val)
		if err != nil {
			log.Errorf("cannot parse height: %v", err)
		}
		return int(f)
	}
	return 0
}

func (cs Map) Width() int {
	w := cs.width()
	if w > 0 {
		if d, ok := cs.Declarations["max-width"]; ok {
			f, _, err := length(&cs, d.Val)
			if err != nil {
				log.Errorf("cannot parse width: %v", err)
			}
			if mw := int(f); 0 < mw && mw < w {
				return mw
			}
		}
	}
	return w
}

func (cs Map) width() int {
	if d, ok := cs.Declarations["width"]; ok {
		f, _, err := length(&cs, d.Val)
		if err != nil {
			log.Errorf("cannot parse width: %v", err)
		}
		if f > 0 {
			return int(f)
		}
	}
	if _, ok := cs.DomTree.Parent(); !ok {
		return WindowWidth
	}
	return 0
}

func (cs Map) baseWidth() int {
	if w := cs.Width(); w != 0 {
		return w
	}
	if p, ok := cs.DomTree.Parent(); ok {
		return p.Style().baseWidth()
	}
	return WindowWidth
}

func (cs Map) Css(propName string) string {
	if d, ok := cs.Declarations[propName]; ok {
		return d.Val
	}
	return ""
}

func (cs *Map) CssPx(propName string) (l int, err error) {
	d, ok := cs.Declarations[propName]
	if !ok {
		return 0, fmt.Errorf("property doesn't exist")
	}
	f, _, err := length(cs, d.Val)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

func (cs Map) SetCss(k, v string) {
	cs.Declarations[k] = Declaration{Prop: k, Val: v}
}
