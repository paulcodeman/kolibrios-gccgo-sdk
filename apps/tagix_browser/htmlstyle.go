package main

import (
	"kos"
	"strconv"
	"strings"
	"ui"
)

var htmlNamedColors = map[string]kos.Color{
	"black":       ui.Black,
	"gray":        ui.Gray,
	"grey":        ui.Gray,
	"silver":      ui.Silver,
	"white":       ui.White,
	"fuchsia":     ui.Fuchsia,
	"purple":      ui.Purple,
	"red":         ui.Red,
	"maroon":      ui.Maroon,
	"yellow":      ui.Yellow,
	"olive":       ui.Olive,
	"lime":        ui.Lime,
	"green":       ui.Green,
	"aqua":        ui.Aqua,
	"teal":        ui.Teal,
	"blue":        ui.Blue,
	"navy":        ui.Navy,
	"orange":      0xFFA500,
	"cyan":        ui.Aqua,
	"magenta":     ui.Fuchsia,
	"transparent": ui.White,
}

var currentDocumentFontFamilies []fontFamilyEntry
var bundledDocumentFontFamilies []fontFamilyEntry
var bundledDocumentFontFamiliesLoaded bool

func setCurrentDocumentFontFamilies(registry []fontFamilyEntry) {
	currentDocumentFontFamilies = registry
}

func lookupFontFamilyPath(registry []fontFamilyEntry, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, entry := range registry {
		if entry.key != key {
			continue
		}
		if path := strings.TrimSpace(entry.path); path != "" {
			return path
		}
	}
	return ""
}

func lookupBundledFontFamilyPath(key string) string {
	if !bundledDocumentFontFamiliesLoaded {
		bundledDocumentFontFamilies = collectBundledFontFamilies()
		bundledDocumentFontFamiliesLoaded = true
	}
	return lookupFontFamilyPath(bundledDocumentFontFamilies, key)
}

func normalizeCSSFontFamilyName(value string) string {
	value = strings.Trim(strings.TrimSpace(strings.ToLower(value)), `"'`)
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	builder.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func isCSSWideKeyword(value string) bool {
	switch value {
	case "inherit", "initial", "unset", "revert", "revertlayer":
		return true
	default:
		return false
	}
}

func applyNodeInlineStyle(style *ui.Style, node *Node) {
	if style == nil || node == nil {
		return
	}
	applyInlineStyle(style, attrValue(node, "style"))
}

func applyInlineStyle(style *ui.Style, declarations string) {
	if style == nil {
		return
	}
	for _, chunk := range strings.Split(declarations, ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		colon := strings.IndexByte(chunk, ':')
		if colon <= 0 || colon+1 >= len(chunk) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(chunk[:colon]))
		value := strings.TrimSpace(chunk[colon+1:])
		applyInlineStyleRule(style, name, value)
	}
}

func applyInlineStyleRule(style *ui.Style, name string, value string) {
	if style == nil {
		return
	}
	switch name {
	case "color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetForeground(color)
		}
	case "background", "background-color":
		applyHTMLBackground(style, value)
	case "background-image":
		if gradient, ok := parseHTMLLinearGradient(value); ok {
			style.SetGradient(gradient)
		}
	case "background-attachment":
		if attachment, ok := ui.ParseBackgroundAttachment(value); ok {
			style.SetBackgroundAttachment(attachment)
		}
	case "display":
		style.SetDisplayString(value)
	case "align-items":
		style.SetAlignItemsString(value)
	case "position":
		if strings.EqualFold(strings.TrimSpace(value), "fixed") {
			style.SetPosition(ui.PositionAbsolute)
		} else {
			style.SetPositionString(value)
		}
	case "text-align":
		style.SetTextAlignString(value)
	case "text-decoration":
		style.SetTextDecorationString(value)
	case "white-space":
		style.SetWhiteSpaceString(value)
	case "overflow":
		style.SetOverflowString(value)
	case "overflow-x":
		style.SetOverflowXString(value)
	case "overflow-y":
		style.SetOverflowYString(value)
	case "contain":
		style.SetContainString(value)
	case "will-change":
		style.SetWillChangeString(value)
	case "opacity":
		if parsed, ok := parseHTMLOpacity(value); ok {
			style.SetOpacityFloat(parsed)
		}
	case "box-shadow":
		if shadow, ok := parseHTMLBoxShadow(value); ok {
			style.SetShadow(shadow)
		}
	case "text-shadow":
		if shadow, ok := parseHTMLTextShadow(value); ok {
			style.SetTextShadow(shadow)
		}
	case "filter":
		if shadow, ok := parseHTMLDropShadow(value); ok {
			style.SetShadow(shadow)
		}
	case "width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetWidth(parsed)
		}
	case "flex-grow":
		style.SetFlexGrowString(value)
	case "height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetHeight(parsed)
		}
	case "min-width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMinWidth(parsed)
		}
	case "max-width":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMaxWidth(parsed)
		}
	case "min-height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMinHeight(parsed)
		}
	case "max-height":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMaxHeight(parsed)
		}
	case "margin":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetMargin(values...)
		}
	case "margin-top":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginTop(parsed)
		}
	case "margin-right":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginRight(parsed)
		}
	case "margin-bottom":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginBottom(parsed)
		}
	case "margin-left":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetMarginLeft(parsed)
		}
	case "padding":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetPadding(values...)
		}
	case "padding-top":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingTop(parsed)
		}
	case "padding-right":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingRight(parsed)
		}
	case "padding-bottom":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingBottom(parsed)
		}
	case "padding-left":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetPaddingLeft(parsed)
		}
	case "border":
		applyHTMLBorder(style, value, "")
	case "border-top":
		applyHTMLBorder(style, value, "top")
	case "border-right":
		applyHTMLBorder(style, value, "right")
	case "border-bottom":
		applyHTMLBorder(style, value, "bottom")
	case "border-left":
		applyHTMLBorder(style, value, "left")
	case "border-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderColor(color)
		}
	case "border-top-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderTopColor(color)
		}
	case "border-right-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderRightColor(color)
		}
	case "border-bottom-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderBottomColor(color)
		}
	case "border-left-color":
		if color, ok := parseHTMLColor(value); ok {
			style.SetBorderLeftColor(color)
		}
	case "border-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderColor()
			style.SetBorder(parsed, color)
		}
	case "border-top-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderTopColor()
			style.SetBorderTop(parsed, color)
		}
	case "border-right-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderRightColor()
			style.SetBorderRight(parsed, color)
		}
	case "border-bottom-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderBottomColor()
			style.SetBorderBottom(parsed, color)
		}
	case "border-left-width":
		if parsed, ok := parseHTMLLength(value); ok {
			color, _ := style.GetBorderLeftColor()
			style.SetBorderLeft(parsed, color)
		}
	case "border-radius":
		if values, ok := parseHTMLBoxLengths(value); ok {
			style.SetBorderRadius(values...)
		}
	case "font-family":
		if path := parseHTMLFontPath(value); path != "" {
			style.SetFontPath(path)
		}
	case "font-size":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetFontSize(parsed)
		}
	case "line-height":
		if parsed, ok := parseHTMLLineHeight(style, value); ok {
			style.SetLineHeight(parsed)
		}
	case "outline":
		if width, color, ok := parseHTMLBorder(value); ok {
			style.SetOutline(width, color)
		}
	case "outline-offset":
		if parsed, ok := parseHTMLLength(value); ok {
			style.SetOutlineOffset(parsed)
		}
	}
}

func applyHTMLBorder(style *ui.Style, value string, side string) {
	if style == nil {
		return
	}
	width, color, ok := parseHTMLBorder(value)
	if !ok {
		return
	}
	switch side {
	case "top":
		style.SetBorderTop(width, color)
	case "right":
		style.SetBorderRight(width, color)
	case "bottom":
		style.SetBorderBottom(width, color)
	case "left":
		style.SetBorderLeft(width, color)
	default:
		style.SetBorder(width, color)
	}
}

func parseHTMLBorder(value string) (int, kos.Color, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, 0, false
	}
	if value == "none" {
		return 0, 0, true
	}
	width := 1
	color := kos.Color(0xC9CFD5)
	seen := false
	for _, token := range strings.Fields(value) {
		if parsed, ok := parseHTMLLength(token); ok {
			width = parsed
			seen = true
			continue
		}
		if token == "solid" || token == "none" {
			seen = true
			continue
		}
		if parsed, ok := parseHTMLColor(token); ok {
			color = parsed
			seen = true
		}
	}
	return width, color, seen
}

func parseHTMLBoxLengths(value string) ([]int, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return nil, false
	}
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		if strings.EqualFold(part, "auto") {
			values = append(values, 0)
			continue
		}
		parsed, ok := parseHTMLLength(part)
		if !ok {
			return nil, false
		}
		values = append(values, parsed)
	}
	return values, true
}

func parseHTMLLength(value string) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, false
	}
	switch value {
	case "0", "0px", "none":
		return 0, true
	case "auto", "100%":
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	if strings.Contains(value, ".") {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return int(floatValue + 0.5), true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func parseHTMLLineHeight(style *ui.Style, value string) (int, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "normal" {
		return 0, false
	}
	if strings.HasSuffix(value, "%") {
		percent, ok := parseHTMLPercent(value)
		if !ok {
			return 0, false
		}
		fontSize := defaultPageFontSize
		if style != nil {
			if current, ok := style.GetFontSize(); ok && current > 0 {
				fontSize = current
			}
		}
		return int(float64(fontSize)*percent + 0.5), true
	}
	if strings.ContainsAny(value, "abcdefghijklmnopqrstuvwxyz") {
		return parseHTMLLength(value)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	fontSize := defaultPageFontSize
	if style != nil {
		if current, ok := style.GetFontSize(); ok && current > 0 {
			fontSize = current
		}
	}
	return int(float64(fontSize)*parsed + 0.5), true
}

func parseHTMLOpacity(value string) (float64, bool) {
	raw := strings.TrimSpace(value)
	isPercent := strings.HasSuffix(raw, "%")
	value = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if isPercent {
		parsed = parsed / 100
	}
	return parsed, true
}

func parseHTMLColor(value string) (kos.Color, bool) {
	r, g, b, a, ok := parseHTMLColorComponents(value)
	if !ok {
		return 0, false
	}
	if a < 255 {
		alpha := int(a)
		r = uint8((int(r)*alpha + 255*(255-alpha)) / 255)
		g = uint8((int(g)*alpha + 255*(255-alpha)) / 255)
		b = uint8((int(b)*alpha + 255*(255-alpha)) / 255)
	}
	return htmlRGBColor(r, g, b), true
}

func parseHTMLColorWithAlpha(value string) (kos.Color, uint8, bool) {
	r, g, b, a, ok := parseHTMLColorComponents(value)
	if !ok {
		return 0, 0, false
	}
	return htmlRGBColor(r, g, b), a, true
}

func parseHTMLColorComponents(value string) (uint8, uint8, uint8, uint8, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, 0, 0, 0, false
	}
	if named, ok := htmlNamedColors[value]; ok {
		if value == "transparent" {
			return 255, 255, 255, 0, true
		}
		return uint8(uint32(named) >> 16), uint8(uint32(named) >> 8), uint8(named), 255, true
	}
	if strings.HasPrefix(value, "#") {
		hex := value[1:]
		switch len(hex) {
		case 3:
			r, okR := parseHTMLHexNibble(hex[0])
			g, okG := parseHTMLHexNibble(hex[1])
			b, okB := parseHTMLHexNibble(hex[2])
			if !okR || !okG || !okB {
				return 0, 0, 0, 0, false
			}
			return r * 17, g * 17, b * 17, 255, true
		case 6:
			parsed, err := strconv.ParseUint(hex, 16, 32)
			if err != nil {
				return 0, 0, 0, 0, false
			}
			return uint8(parsed >> 16), uint8(parsed >> 8), uint8(parsed), 255, true
		}
	}
	if strings.HasPrefix(value, "rgb(") || strings.HasPrefix(value, "rgba(") {
		args, ok := htmlFunctionArgs(value)
		if !ok {
			return 0, 0, 0, 0, false
		}
		parts, alphaValue, ok := splitHTMLColorFunctionArgs(args)
		if !ok || len(parts) != 3 {
			return 0, 0, 0, 0, false
		}
		r, okR := parseHTMLColorChannel(parts[0])
		g, okG := parseHTMLColorChannel(parts[1])
		b, okB := parseHTMLColorChannel(parts[2])
		if !okR || !okG || !okB {
			return 0, 0, 0, 0, false
		}
		alpha := uint8(255)
		if alphaValue != "" {
			value, ok := parseHTMLAlphaChannel(alphaValue)
			if !ok {
				return 0, 0, 0, 0, false
			}
			alpha = value
		}
		return r, g, b, alpha, true
	}
	if strings.HasPrefix(value, "hsl(") || strings.HasPrefix(value, "hsla(") {
		args, ok := htmlFunctionArgs(value)
		if !ok {
			return 0, 0, 0, 0, false
		}
		parts, alphaValue, ok := splitHTMLColorFunctionArgs(args)
		if !ok || len(parts) != 3 {
			return 0, 0, 0, 0, false
		}
		h, okH := parseHTMLHue(parts[0])
		s, okS := parseHTMLPercent(parts[1])
		l, okL := parseHTMLPercent(parts[2])
		if !okH || !okS || !okL {
			return 0, 0, 0, 0, false
		}
		r, g, b := htmlHSLToRGB(h, s, l)
		alpha := uint8(255)
		if alphaValue != "" {
			value, ok := parseHTMLAlphaChannel(alphaValue)
			if !ok {
				return 0, 0, 0, 0, false
			}
			alpha = value
		}
		return r, g, b, alpha, true
	}
	return 0, 0, 0, 0, false
}

func parseHTMLHexNibble(value byte) (uint8, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func htmlRGBColor(r uint8, g uint8, b uint8) kos.Color {
	return kos.Color(uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func parseHTMLColorChannel(value string) (uint8, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if strings.HasSuffix(value, "%") {
		percent, ok := parseHTMLPercent(value)
		if !ok {
			return 0, false
		}
		return uint8(percent*255 + 0.5), true
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if parsed < 0 {
		parsed = 0
	}
	if parsed > 255 {
		parsed = 255
	}
	return uint8(parsed + 0.5), true
}

func parseHTMLAlphaChannel(value string) (uint8, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if strings.HasSuffix(value, "%") {
		percent, ok := parseHTMLPercent(value)
		if !ok {
			return 0, false
		}
		return uint8(percent*255 + 0.5), true
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if parsed < 0 {
		parsed = 0
	}
	if parsed > 1 {
		if parsed > 255 {
			parsed = 255
		}
		return uint8(parsed + 0.5), true
	}
	return uint8(parsed*255 + 0.5), true
}

func parseHTMLPercent(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if parsed < 0 {
		parsed = 0
	}
	if parsed > 100 {
		parsed = 100
	}
	return parsed / 100, true
}

func parseHTMLHue(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(value), "deg"))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	for parsed < 0 {
		parsed += 360
	}
	for parsed >= 360 {
		parsed -= 360
	}
	return parsed / 360, true
}

func htmlHSLToRGB(h float64, s float64, l float64) (uint8, uint8, uint8) {
	if s <= 0 {
		value := uint8(l*255 + 0.5)
		return value, value, value
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r := htmlHueToRGB(p, q, h+1.0/3.0)
	g := htmlHueToRGB(p, q, h)
	b := htmlHueToRGB(p, q, h-1.0/3.0)
	return uint8(r*255 + 0.5), uint8(g*255 + 0.5), uint8(b*255 + 0.5)
}

func htmlHueToRGB(p float64, q float64, t float64) float64 {
	for t < 0 {
		t += 1
	}
	for t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

func htmlFunctionArgs(value string) (string, bool) {
	start := strings.IndexByte(value, '(')
	end := strings.LastIndexByte(value, ')')
	if start <= 0 || end <= start {
		return "", false
	}
	return strings.TrimSpace(value[start+1 : end]), true
}

func splitStyleValueTokens(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	tokens := make([]string, 0, 8)
	start := -1
	depth := 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '(':
			if start < 0 {
				start = index
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ' ', '\t', '\r', '\n':
			if depth == 0 && start >= 0 {
				tokens = append(tokens, strings.TrimSpace(value[start:index]))
				start = -1
			}
			continue
		}
		if start < 0 {
			start = index
		}
	}
	if start >= 0 {
		tokens = append(tokens, strings.TrimSpace(value[start:]))
	}
	return tokens
}

func splitStyleFunctionArgs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := make([]string, 0, 4)
	start := 0
	depth := 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	return parts
}

func normalizeCSSCommentWhitespace(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.Contains(value, "/*") {
		return value
	}
	builder := strings.Builder{}
	builder.Grow(len(value))
	for index := 0; index < len(value); {
		if index+1 < len(value) && value[index] == '/' && value[index+1] == '*' {
			end := strings.Index(value[index+2:], "*/")
			if end < 0 {
				break
			}
			if builder.Len() > 0 {
				last := builder.String()[builder.Len()-1]
				if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
					builder.WriteByte(' ')
				}
			}
			index += end + 4
			continue
		}
		builder.WriteByte(value[index])
		index++
	}
	return strings.TrimSpace(builder.String())
}

func splitHTMLColorFunctionTokens(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	tokens := make([]string, 0, 5)
	start := -1
	depth := 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '(':
			if start < 0 {
				start = index
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '/':
			if depth == 0 {
				if start >= 0 {
					tokens = append(tokens, strings.TrimSpace(value[start:index]))
					start = -1
				}
				tokens = append(tokens, "/")
				continue
			}
		case ' ', '\t', '\r', '\n':
			if depth == 0 {
				if start >= 0 {
					tokens = append(tokens, strings.TrimSpace(value[start:index]))
					start = -1
				}
				continue
			}
		}
		if start < 0 {
			start = index
		}
	}
	if start >= 0 {
		tokens = append(tokens, strings.TrimSpace(value[start:]))
	}
	return tokens
}

func splitHTMLColorFunctionArgs(value string) ([]string, string, bool) {
	value = normalizeCSSCommentWhitespace(value)
	if value == "" {
		return nil, "", false
	}
	if strings.Contains(value, ",") {
		parts := splitStyleFunctionArgs(value)
		if len(parts) < 3 || len(parts) > 4 {
			return nil, "", false
		}
		alpha := ""
		if len(parts) == 4 {
			alpha = strings.TrimSpace(parts[3])
		}
		return parts[:3], alpha, true
	}
	tokens := splitHTMLColorFunctionTokens(value)
	if len(tokens) < 3 {
		return nil, "", false
	}
	alpha := ""
	slash := -1
	for index, token := range tokens {
		if token == "/" {
			if slash >= 0 {
				return nil, "", false
			}
			slash = index
		}
	}
	if slash >= 0 {
		if slash != 3 || slash+1 != len(tokens)-1 {
			return nil, "", false
		}
		alpha = strings.TrimSpace(tokens[slash+1])
		tokens = tokens[:slash]
	}
	if len(tokens) != 3 {
		return nil, "", false
	}
	return tokens, alpha, true
}

func applyHTMLBackground(style *ui.Style, value string) {
	if style == nil {
		return
	}
	for _, token := range splitStyleValueTokens(value) {
		if token == "" {
			continue
		}
		if attachment, ok := ui.ParseBackgroundAttachment(token); ok {
			style.SetBackgroundAttachment(attachment)
			continue
		}
		if colorAlphaZero(token) {
			continue
		}
		if color, ok := parseHTMLColor(token); ok {
			style.SetBackground(color)
			continue
		}
		if gradient, ok := parseHTMLLinearGradient(token); ok {
			style.SetGradient(gradient)
		}
	}
}

func colorAlphaZero(value string) bool {
	_, _, _, alpha, ok := parseHTMLColorComponents(value)
	return ok && alpha == 0
}

func extractCSSURLValue(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 5 || !strings.HasSuffix(value, ")") {
		return "", false
	}
	if !strings.EqualFold(strings.TrimSpace(value[:4]), "url(") {
		return "", false
	}
	inner := strings.TrimSpace(value[4 : len(value)-1])
	if len(inner) >= 2 {
		quote := inner[0]
		if (quote == '\'' || quote == '"') && inner[len(inner)-1] == quote {
			inner = inner[1 : len(inner)-1]
		}
	}
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", false
	}
	return inner, true
}

func parseHTMLLinearGradient(value string) (ui.Gradient, bool) {
	args, ok := htmlFunctionArgs(strings.TrimSpace(value))
	if !ok || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "linear-gradient(") {
		return ui.Gradient{}, false
	}
	parts := splitStyleFunctionArgs(args)
	colors := make([]kos.Color, 0, 2)
	direction := ui.GradientVertical
	for _, part := range parts {
		lower := strings.ToLower(strings.TrimSpace(part))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "to right") {
			direction = ui.GradientHorizontal
			continue
		}
		if strings.Contains(lower, "deg") {
			if strings.Contains(lower, "90deg") || strings.Contains(lower, "270deg") {
				direction = ui.GradientHorizontal
			}
			continue
		}
		if color, ok := parseHTMLColor(part); ok {
			colors = append(colors, color)
		}
	}
	if len(colors) < 2 {
		return ui.Gradient{}, false
	}
	return ui.Gradient{
		From:      colors[0],
		To:        colors[1],
		Direction: direction,
	}, true
}

func parseHTMLBoxShadow(value string) (ui.Shadow, bool) {
	return parseHTMLShadowValue(value, true)
}

func parseHTMLTextShadow(value string) (ui.TextShadow, bool) {
	shadow, ok := parseHTMLShadowValue(value, false)
	if !ok {
		return ui.TextShadow{}, false
	}
	return ui.TextShadow{
		OffsetX: shadow.OffsetX,
		OffsetY: shadow.OffsetY,
		Color:   shadow.Color,
	}, true
}

func parseHTMLDropShadow(value string) (ui.Shadow, bool) {
	args, ok := htmlFunctionArgs(strings.TrimSpace(value))
	if !ok || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "drop-shadow(") {
		return ui.Shadow{}, false
	}
	return parseHTMLShadowValue(args, true)
}

func parseHTMLShadowValue(value string, includeBlur bool) (ui.Shadow, bool) {
	shadow := ui.Shadow{}
	tokens := splitStyleValueTokens(value)
	if len(tokens) == 0 {
		return shadow, false
	}
	lengths := make([]int, 0, 4)
	alpha := uint8(255)
	colorSet := false
	for _, token := range tokens {
		lower := strings.ToLower(strings.TrimSpace(token))
		if lower == "" || lower == "inset" {
			continue
		}
		if color, parsedAlpha, ok := parseHTMLColorWithAlpha(lower); ok {
			shadow.Color = color
			alpha = parsedAlpha
			colorSet = true
			continue
		}
		if length, ok := parseHTMLLength(lower); ok {
			lengths = append(lengths, length)
		}
	}
	if len(lengths) < 2 {
		return shadow, false
	}
	shadow.OffsetX = lengths[0]
	shadow.OffsetY = lengths[1]
	if includeBlur && len(lengths) >= 3 {
		shadow.Blur = lengths[2]
	}
	if !colorSet {
		shadow.Color = 0x000000
		alpha = 64
	}
	shadow.Alpha = alpha
	return shadow, true
}

func parseHTMLFontPath(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	sawFamily := false
	for _, part := range parts {
		raw := strings.Trim(strings.TrimSpace(part), `"'`)
		if raw == "" {
			continue
		}
		if fontPathSupported(raw) {
			return raw
		}
		family := normalizeCSSFontFamilyName(part)
		if family == "" || isCSSWideKeyword(family) {
			continue
		}
		sawFamily = true
		if path := lookupFontFamilyPath(currentDocumentFontFamilies, family); path != "" {
			return path
		}
		if path := lookupBundledFontFamilyPath(family); path != "" {
			return path
		}
	}
	if !sawFamily {
		return ""
	}
	return webSansFontPath
}

func lastStyleToken(value string) string {
	tokens := strings.Fields(strings.TrimSpace(value))
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

func applyShellFrameTemplate(app *App, node *Node, ctx *shellRenderContext) {
	if app == nil || node == nil || app.pageFrame == nil || app.pageView == nil {
		return
	}
	if src := strings.TrimSpace(attrValue(node, "src")); src != "" {
		app.startupURL = normalizeURL(src)
	}
	template := ui.Style{}
	applyShellNodeStyles(&template, node, ctx)

	if display, ok := template.GetDisplay(); ok {
		app.pageFrame.Style.SetDisplay(display)
	}
	if margin, ok := template.GetMargin(); ok {
		if ctx != nil && ctx.root != nil && !app.webViewMode {
			hostRoot := ui.Style{}
			applyShellNodeStyles(&hostRoot, ctx.root, ctx)
			if padding, paddingOK := hostRoot.GetPadding(); paddingOK {
				margin.Left += padding.Left
				margin.Right += padding.Right
			}
		}
		app.pageFrame.Style.SetMargin(margin.Top, margin.Right, margin.Bottom, margin.Left)
	}
	if padding, ok := template.GetPadding(); ok {
		app.pageFrame.Style.SetPadding(padding.Top, padding.Right, padding.Bottom, padding.Left)
	}
	if border, ok := template.GetBorderWidth(); ok {
		color, colorOK := template.GetBorderColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorder(border, color)
	}
	if border, ok := template.GetBorderTopWidth(); ok {
		color, colorOK := template.GetBorderTopColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderTop(border, color)
	}
	if border, ok := template.GetBorderRightWidth(); ok {
		color, colorOK := template.GetBorderRightColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderRight(border, color)
	}
	if border, ok := template.GetBorderBottomWidth(); ok {
		color, colorOK := template.GetBorderBottomColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderBottom(border, color)
	}
	if border, ok := template.GetBorderLeftWidth(); ok {
		color, colorOK := template.GetBorderLeftColor()
		if !colorOK {
			color = 0xD7DEE7
		}
		app.pageFrame.Style.SetBorderLeft(border, color)
	}
	if radius, ok := template.GetBorderRadius(); ok {
		app.pageFrame.Style.SetBorderRadius(radius.TopLeft, radius.TopRight, radius.BottomRight, radius.BottomLeft)
	}
	if color, ok := template.GetBackground(); ok {
		app.pageFrame.Style.SetBackground(color)
		app.pageView.Style.SetBackground(color)
	}
	if opacity, ok := template.GetOpacity(); ok {
		app.pageFrame.Style.SetOpacity(opacity)
	}
	if overflow, ok := template.GetOverflow(); ok {
		app.pageView.Style.SetOverflow(overflow)
	}
	if overflowX, ok := template.GetOverflowX(); ok {
		app.pageView.Style.SetOverflowX(overflowX)
	}
	if overflowY, ok := template.GetOverflowY(); ok {
		app.pageView.Style.SetOverflowY(overflowY)
	}
	if minHeight, ok := template.GetMinHeight(); ok && minHeight > 0 {
		app.pageMinHeight = minHeight
		app.pageView.Style.SetMinHeight(minHeight)
	}
	if height, ok := template.GetHeight(); ok && height > 0 {
		if height > app.pageMinHeight {
			app.pageMinHeight = height
		}
	}
	if maxHeight, ok := template.GetMaxHeight(); ok && maxHeight > 0 {
		app.pageView.Style.SetMaxHeight(maxHeight)
	}
	if width, ok := template.GetMinWidth(); ok && width > 0 {
		app.pageFrame.Style.SetMinWidth(width)
	}
	if width, ok := template.GetMaxWidth(); ok && width > 0 {
		app.pageFrame.Style.SetMaxWidth(width)
	}
}
